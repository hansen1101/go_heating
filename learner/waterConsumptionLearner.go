package learner

import (
	"github.com/hansen1101/go_heating/system"
	"time"
	"fmt"
	"math"
	"io"
	"sync"
	"strings"
	"errors"
	"log"
	"github.com/hansen1101/go_heating/auxiliary/clustering"
	"github.com/hansen1101/go_heating/auxiliary/algorithm"
)

const(
	MAX_BUCKET_SIZE = 512
	CUTOUTERROR = "Cut out failed."
	DISTANCE_MATRIX_INT_SCALING_FACTOR = 128
	CLUSTER_DB_TABLE_NAME = "multiDimCluster"
	CLUSTERING_DECAY_HORIZON = 600
)

// Learns WaterConsumptin by unsupervised learning task through clustering.
// The system is able to make forecasts regarding the water requirement if the clustering is
// acurate.
type waterConsumptionLearner struct{
	k int
	epsilon float64
	p int
	sec int
	overlap int
	windowTimeHorizonInSec int64
	clusteringPointList []*bucketPoint

}
type bucket struct {
	timestamp int64
	itemcount int
	record []clustering.Cluster
}
type deltaPoint struct {
	*clustering.GenericPoint
	timestamp int64
}
type bucketPoint struct {
	*clustering.GenericPoint
	volume int
	time time.Time
}

func NewWaterConsumptionLearner(k int, epsilon float64,p,sec,overlap int)(learner *waterConsumptionLearner){
	learner = &waterConsumptionLearner{k:k,epsilon:epsilon,p:p,sec:sec,overlap:overlap}
	learner.clusteringPointList = make([]*bucketPoint,0,CLUSTERING_DECAY_HORIZON)
	learner.windowTimeHorizonInSec = calcTimeHorizonInSec(p,sec,overlap)
	return
}

func calcTimeHorizonInSec(p,sec,overlap int)(int64){
	slotLength := p * (sec - overlap)
	slotSizeSum := float64(0)
	for i:=0;i<=int(math.Log2(float64(MAX_BUCKET_SIZE))-math.Log2(float64(p)));i++ {
		slotSizeSum += math.Pow(float64(2),float64(i))
	}
	maxSlotNumber := float64(2) * slotSizeSum
	return int64(slotLength) * int64(maxSlotNumber)
}

// Constructs a new deltaPoint with a one-dimension underlying GenericPoint
func NewDeltaPoint(dimension int)(point *deltaPoint){
	point = &deltaPoint{
		GenericPoint:clustering.NewGenericPoint(dimension),
	}
	return
}
func CloneDeltaPoint(other clustering.Point)(point *deltaPoint){
	point = NewDeltaPoint(other.Dimensions())
	for dim,coordinate := range other.GetVector() {
		point.SetCoordinate(dim,coordinate.GetValue())
	}
	return
}

func newBucketPoint(dimensions int)(point *bucketPoint){
	point = &bucketPoint{
		GenericPoint:clustering.NewGenericPoint(dimensions),
	}
	return
}

// Setter for the timestamp field
func (p *deltaPoint) SetTimestamp(timestamp int64){
	p.timestamp = timestamp
}
// Setter for the first coordinate of the delta point.
func (p *deltaPoint) SetDelta(value float64){
	p.SetCoordinate(
		0,
		value,
	)
}
func (p *deltaPoint) WeightedEuclideanDistanceTo(other clustering.Point)(distance float64){
	//fmt.Println(reflect.TypeOf(p),reflect.TypeOf(other))
	switch v:=other.(type) {
	case *deltaPoint:
		distance = p.GenericPoint.WeightedEuclideanDistanceTo(v.GenericPoint)
	case *clustering.GenericPoint:
		distance = p.GenericPoint.WeightedEuclideanDistanceTo(v)
	}
	return
}

func (p *bucketPoint) SetTime(timestamp int64)(){
	p.time = time.Unix(timestamp,0)
}
func (p *bucketPoint) SetVolume(vol int)(){
	p.volume = vol
}

// This methods calculates a weighted mean value of the centroids in the clustering collection
// by calculating a sum of the values weighted by the cluster's height for each dimension and
// normalizing it by the total height of the clustering collection.
// Assumes all clusters in the collection are elements from the same vector space.
// @return Point in the vector space of the collection
func getClusteringCenter(collection []clustering.Cluster)(clustering.Point){
	var totalSize int
	var center *deltaPoint
	if len(collection) > 0 {
		center = NewDeltaPoint(collection[0].GetCentroid().Dimensions())
		for _,cluster := range collection{
			// loop over all dimensions of the cluster's centroid and add the value weighted by cluster's height
			for dimension,value := range cluster.GetCentroid().GetVector() {
				if center.GetVector()[dimension] == nil {
					center.SetCoordinate(dimension,value.GetValue().(float64))
				} else {
					center.GetVector()[dimension].AddValue(value.GetValue().(float64) * float64(cluster.GetClusterSize()))
				}
			}

			// sum up height for normalization
			totalSize += cluster.GetClusterSize()

			// update timestamp of center if necessary
			if v,ok := cluster.GetCentroid().(*deltaPoint); ok {
				if v.timestamp > center.timestamp {
					center.timestamp = v.timestamp
				}
			}
		}
		if totalSize > 0 {
			// normalize the vector by the total height of the cluster
			for dimension,_ := range center.GetVector() {
				center.GetVector()[dimension].NormalizeValue(float64(totalSize))
			}
		}
	}
	return center
}

// Calculates a noise value for the clustering collection which is the mean distance
// between each cluster's centroid and the collection center weighted by the height of the cluster.
// @return float64 a weighted mean distance between centroids and center
func CalcClusteringNoise(collection []clustering.Cluster,pointDistAlgo clustering.PointDistance)(noise float64,center clustering.Point){
	center = getClusteringCenter(collection)
	for _,cluster := range collection {
		noise += pointDistAlgo(cluster.GetCentroid(),center) * float64(cluster.GetClusterSize())
	}
	if n := len(collection); n > 0 {
		noise /= float64(n)
	}
	return
}

// Calculates the mean diameter of the clustering collection for the given clustering.PointDistance measure.
// @return float64 the mean diameter of the clusters in the collection
func CalcClusteringDiam(collection []clustering.Cluster,pointDistAlgo clustering.PointDistance)(diam float64){
	for _,cluster := range collection {
		clusterDiam,_,_ := clustering.GetDiam(cluster.GetClusterItems(),pointDistAlgo)
		diam += clusterDiam
	}
	if n := len(collection); n > 0 {
		diam /= float64(n)
	}
	return
}

// The munkres assignment algorithm for cluster specific application. Takes two clusterings and a distance measure and returns
// a pointer to an assignment vector.
// @return *[]int the assignment vector where the index is associated with a cluster in clustering a and the value with the assigned cluster from clustering b
func munkresClusterAssignment(a,b *[]clustering.Cluster, distAlgo clustering.DistanceMeasure,pointDistAlgo clustering.PointDistance)(*[]int){
	matrix := generateClusterDistanceMatrix(a,b,distAlgo,pointDistAlgo)
	return algorithm.Munkres(*matrix)
}

// Takes two clusterings and a distance measure and computes the distance matrix between the two clusterings.
// This method can be used for cluster specific initialization to generic algorithms like the munkres assignment.
// Due to convinient evaluation of the distances, floating values are converted into integers by the use of a constant scaling factor.
// @return *[][]int pointer to the distance matrix
func generateClusterDistanceMatrix(a,b *[]clustering.Cluster, distAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance)(*[][]int){
	var matrix [][]int
	var maxN int
	if maxN = len(*a); len(*b) > maxN {
		maxN = len(*b)
	}

	// init the distance matrix
	matrix = make([][]int,maxN,maxN)
	for i,_ := range(matrix){
		(matrix)[i] = make([]int,maxN,maxN)
	}

	// add distance values to the matrix
	for i,clusterA := range(*a){
		for j,clusterB := range(*b) {
			dist,err := clusterA.DistanceTo(clusterB,distAlgo,pointDistAlgo)
			(matrix)[i][j] = int(dist * float64(DISTANCE_MATRIX_INT_SCALING_FACTOR))
			if err != nil {
				fmt.Println(err.Error())
				log.Fatal("There was a problem with the DistanceTo function")
			}
		}
	}
	return &matrix
}

// @testing: successfully
func cutOutPointerIndex(data *[]*bucket, cutIndex int)(err error){
	if data != nil && cutIndex < len(*data) && cutIndex >= 0 {
		switch cutIndex {
		case 0:
			*data = (*data)[cutIndex + 1:]
		case len(*data) - 1:
			*data = (*data)[:cutIndex]
		default:
			*data = append((*data)[:cutIndex], (*data)[cutIndex + 1:]...)
		}
	} else {
		err = errors.New(CUTOUTERROR)
	}
	return
}

// @testing: successfully
func cutOutPointPointerIndex(data *[]*deltaPoint, cutIndex int)(err error){
	if data != nil && cutIndex < len(*data) && cutIndex >= 0 {
		switch cutIndex {
		case 0:
			*data = (*data)[cutIndex + 1:]
		case len(*data) - 1:
			*data = (*data)[:cutIndex]
		default:
			*data = append((*data)[:cutIndex], (*data)[cutIndex + 1:]...)
		}
	} else {
		err = errors.New(CUTOUTERROR)
	}
	return
}

// Performs a run of agglomerativeClustering with the given points for each distance measure.
// @param points set of deltaPoint values that need to be clustered
// @param maxK upper bound on the number of distinct clusters the algorithm outputs
// @param extractor algorithm.LevelExtractor function that should be applied to obtain the best number of clusterings k
// @param distanceBetween list of clustering.DistanceMeasure
func generateAgglomerativeBucket(points []*deltaPoint,maxK int,extractor algorithm.LevelExtractor,pointDistAlgo clustering.PointDistance,distanceBetween... clustering.DistanceMeasure)(finalCluster []clustering.Cluster,itemcount int,timestamp int64){
	if len(distanceBetween) == 0 {
		log.Fatal("At least one cluster distance measure required for agglomerative clustering.")
	}
	finalCluster = make([]clustering.Cluster,0)
	clusterLevels := make([]*[][]clustering.Cluster,0)

	// cast all points to cluster.Point before clustering
	var interfaceSlice []clustering.Point = GenerateInterfaceSliceFromDeltaSlice(points)

	for _,distanceAlgo := range distanceBetween {
		// first dimension is represents k=#clusters; given index i it holds k = maxK - i
		// second dimension is slice of pointer to clusters
		var a *[][]clustering.Cluster = algorithm.AgglomerativeClustering(interfaceSlice, distanceAlgo, pointDistAlgo)
		clusterLevels = append(clusterLevels,a)
	}
	k := extractor(maxK, distanceBetween[0], clustering.EuclideanDistance, clusterLevels...)
	genericClustering := fetchClusterLevel(k,clusterLevels[0],clustering.EuclideanDistance)

	// generate fullCluster for each generic cluster in clustering
	for _,c := range genericClustering {
		next := newFullCluster(c,pointDistAlgo)
		finalCluster = append(finalCluster,next)
		itemcount += next.clustersize
		if timestamp < next.centroid.timestamp {
			timestamp = next.centroid.timestamp
		}
	}
	return
}

func (learner *waterConsumptionLearner)generateKMeansBucket(points []*deltaPoint,maxK int,extractor algorithm.LevelExtractor,pointDistAlgo clustering.PointDistance,distanceBetween... clustering.DistanceMeasure)(finalCluster []clustering.Cluster,itemcount int,timestamp int64) {
	if len(distanceBetween) == 0 {
		log.Fatal("At least one cluster distance measure required for agglomerative clustering.")
	}
	finalCluster = make([]clustering.Cluster,0)
	clusterLevels := make([][]clustering.Cluster,0)

	// cast all points to cluster.Point before clustering
	var interfaceSlice []clustering.Point = GenerateInterfaceSliceFromDeltaSlice(points)

	for k:=maxK;k>0;k--{
		nextClustering, err := algorithm.KMeans(
			k,
			learner.epsilon,
			interfaceSlice,
			distanceBetween[0],
			pointDistAlgo,
		)
		if err == nil {
			clusterLevels = append(clusterLevels,nextClustering)
		}
	}

	k := extractor(maxK, distanceBetween[0], clustering.EuclideanDistance, &clusterLevels)
	genericClustering := fetchClusterLevel(k,&clusterLevels,clustering.EuclideanDistance)

	// generate fullCluster for each generic cluster in clustering
	for _,c := range genericClustering {
		next := newFullCluster(c,pointDistAlgo)
		finalCluster = append(finalCluster,next)
		itemcount += next.clustersize
		if timestamp < next.centroid.timestamp {
			timestamp = next.centroid.timestamp
		}
	}
	return
}

// Evaluates different lists of clusterings (created with different distance measures and parameters) and
// extracts the best value k for the number of clusterings based on the development of successive diameter
// developments for the clusters per level.
// Implementation of the algorithm.LevelExctractor type
// @return int k lowest number of clusterings after which the clusterings become bad
func extractBestClusterLevel(maxK int, clusterDistAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance, levelsListPerDistanceMeasure... *[][]clustering.Cluster)(k int){
	// list where the results of comparison between all levelLists are saved for final evaluation
	// each slot in the list represents a merge from k to k-1 clusters
	mergeComparisonList := make([]float64,maxK,maxK)

	// init k to be 1
	k = 1

	// loop over all cluster hierarchy lists generated by each distance measure
	for _,levelsPointer := range levelsListPerDistanceMeasure{
		levels := *levelsPointer

		// index for the minimal cluster number that should taken into consideration
		var iMin int

		// calculation can only start from the second level
		if iMin = len(levels) - maxK; iMin < 1 {
			iMin = 1
		}

		// loop over cluster levels in current matrix
		var deltaPrime, diamPrime, distPrime float64
		perLevelMergeComparisonList := make([]float64,maxK,maxK)
		for i,_:= range perLevelMergeComparisonList {
			perLevelMergeComparisonList[i] = 1.0
		}

		// consider every clustering for the distance measure
		for i,kClustering := range levels {
			if i > 0 {
				//@todo optimize the merging rules
				// calculate diam of current and level before

				var diam, // sum of diameter of all clusters in the clustering
				delta, // difference in diameter between this and last level
				totalDist, // sum of distance between any pair of clusters in the clustering
				frac float64

				// calculate diam and totalDist value
				for i,cluster := range kClustering {
					tmp,_,_ := cluster.GetDiameter(pointDistAlgo)
					diam += tmp
					for j,other := range kClustering {
						if j > i {
							d,_ := cluster.DistanceTo(other,clusterDistAlgo,pointDistAlgo)
							//fmt.Print(len(kClustering))
							//fmt.Printf("\tDistance between %s - %s: %.2f\n",cluster,other,d)
							totalDist += d
						}
					}
				}

				// set delta value between two levels
				//delta = diamPrime / float64(len(clustering)) - diam / float64(len(levels[i-1]))
				delta = diam - diamPrime

				frac = 1.0 // fraction of current delta given the last delta
				if deltaPrime > float64(0){
					frac = delta / deltaPrime
				}
				//fmt.Printf("Diameter change is %.1f %% of the last diameter change\n",frac*float64(100))

				// if current distance between last diam and current diamPrime is larger than current diam
				/*
				if i >= iMin-1 {
					fmt.Printf("Level:%d/%d k:%d********* Diam: %.2f Delta %.2f DiamPrime: %.2f DeltaPrime: %.2f Dist: %.2f DeltaDist: %.2f\n",i,i-iMin,len(kClustering),diam,delta,diamPrime,deltaPrime,totalDist,distPrime-totalDist)
				}
				*/

				if diamPrime == float64(0) && diam > float64(0) && totalDist == float64(0) {
					// last clusterlevel is the first that shifted a centroid
					if i > iMin {
						perLevelMergeComparisonList[i-iMin-1] += frac * diam
					}
				}

				if deltaDist := distPrime-totalDist; totalDist > deltaDist && i >= iMin {
					//mergeComparisonList[i-iMin] += diamPrime
					//fmt.Printf("k:%d (%d/%d) gets an addition of %.2f x %.4f",len(kClustering),i,i-iMin,frac,(float64(1.0) / (float64(2.0) * deltaDist)))
					//mergeComparisonList[i-iMin] += delta - diamPrime
					perLevelMergeComparisonList[i-iMin] += frac * (float64(1.0) / (float64(2.0) * deltaDist))
					//fmt.Printf("\t new value: %.2f\n",perLevelMergeComparisonList[i-iMin])
				} else if totalDist < deltaDist && i > iMin {
					// the last index gets a contribution
					if deltaDist > 0.0 && totalDist > 0.0 {
						//fmt.Printf("k:%d (%d/%d) gets Dist contribution of x %.2f",len(kClustering)+1,i,i-iMin-1, (1.0 + (1.0 / (deltaDist / totalDist))))
						perLevelMergeComparisonList[i-iMin-1] *= (1.0 + (1.0 / (deltaDist / totalDist)))
						//fmt.Printf("\t new value: %.2f\n",perLevelMergeComparisonList[i-iMin-1])
					}
				}

				if diamPrime > 0.0 && i > iMin {
					// the last index gets a contribution
					//fmt.Printf("k:%d (%d/%d) gets Diam contribution of x %.2f",len(kClustering)+1,i,i-iMin-1, (1.0+(diam / diamPrime)))
					var factor float64
					if factor = float64(1.0); diamPrime < factor {
						factor = diamPrime
					}
					perLevelMergeComparisonList[i-iMin-1] *= (1.0+(factor * diam / diamPrime))
					//fmt.Printf("\t new value: %.2f\n",perLevelMergeComparisonList[i-iMin-1])
				}
				diamPrime = diam
				deltaPrime = delta
				distPrime = totalDist
			}
		}
		for i,_:= range mergeComparisonList {
			mergeComparisonList[i] += perLevelMergeComparisonList[i]
		}
		//fmt.Printf("%+v\n",perLevelMergeComparisonList)
		//fmt.Println()
	}

	// final evaluation: take the level k before the largest total delta value as the correct value for the number of clusters
	//fmt.Println()
	max := float64(1.0) * float64(len(levelsListPerDistanceMeasure))
	for toStep,deltaTotal := range mergeComparisonList {
		//fmt.Printf("K:%d Score:%.2f ; ",maxK-toStep,deltaTotal)
		if deltaTotal > max {
			max = deltaTotal
			// don't take this, but the split before
			k = maxK - toStep
		}
	}
	//fmt.Println()
	//fmt.Printf("Best value for K: %d\n\n",k)

	return
}

func extractBestKMeansLevel(maxK int, clusterDistAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance, levelsListPerDistanceMeasure... *[][]clustering.Cluster)(k int){
	// list where the results of comparison between all levelLists are saved for final evaluation
	// each slot in the list represents a merge from k to k-1 clusters
	mergeComparisonList := make([]float64,maxK,maxK)

	// init k to be 1
	k = 1

	// loop over all cluster hierarchy lists generated by each distance measure
	for _,levelsPointer := range levelsListPerDistanceMeasure{
		levels := *levelsPointer

		// index for the minimal cluster number that should taken into consideration
		var iMin int

		// calculation can only start from the second level
		if iMin = len(levels) - maxK; iMin < 0 {
			iMin = 0
		}

		// loop over cluster levels in current matrix
		var deltaPrime, diamPrime float64
		perLevelMergeComparisonList := make([]float64,maxK,maxK)
		for i,_:= range perLevelMergeComparisonList {
			perLevelMergeComparisonList[i] = 1.0
		}

		// consider every clustering for the distance measure
		for i,_ := range levels {

			//@todo optimize the merging rules
			// calculate diam of current and level before

			var diam, // sum of diameter of all clusters in the clustering
			diamDelta float64 // difference in diameter between this and last level

			// set delta value between two levels
			diamDelta = diam - diamPrime

			/*
			fmt.Printf("Level:%d/%d k:%d********* Diam: %.2f Delta %.2f DiamPrime: %.2f DeltaPrime: %.2f\n",i,i-iMin,len(kClustering),diam,diamDelta,diamPrime,deltaPrime)
			printCluster(levels[i])
			*/

			// if current distance between last diam and current diamPrime is larger than current diam
			perLevelMergeComparisonList[i-iMin] += float64(1) / ((float64(1) + diamDelta))

			//fmt.Printf("\t new value: %.2f\n",perLevelMergeComparisonList[i-iMin])

			if i > iMin {
				// we can contribute to level before

				// contribute change in diamDelta based on last level's diamDelta to last level
				if deltaPrime > float64(0) {
					perLevelMergeComparisonList[i - iMin - 1] += diamDelta / deltaPrime
					//fmt.Printf("\t new value: %.2f\n", perLevelMergeComparisonList[i - iMin - 1])
				}
			}

			diamPrime = diam
			deltaPrime = diamDelta
		}
		for i,_:= range mergeComparisonList {
			mergeComparisonList[i] += perLevelMergeComparisonList[i]
		}
		//fmt.Printf("%+v\n",perLevelMergeComparisonList)
		//fmt.Println()
	}

	// final evaluation: take the level k before the largest total delta value as the correct value for the number of clusters
	max := float64(1.0) * float64(len(levelsListPerDistanceMeasure))
	for toStep,deltaTotal := range mergeComparisonList {
		if deltaTotal >= max {
			max = deltaTotal
			// don't take this, but the split before
			k = maxK - toStep
		}
	}

	return
}

// Instance of algorithm.LevelExtractor.
// Computes the optimal number of clusters for a given set of clusterings with
// distinct number of clusters over the same set of data points for a set of
// different distance measures.
// @return k int the best number of clusters
func extractBestClusterLevelSilhouette(maxK int, clusterDistAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance, levelsListPerDistanceMeasure... *[][]clustering.Cluster)(k int){
	// list where the results of comparison between all levelLists are saved for final evaluation
	// each slot in the list represents a merge from k to k-1 clusters
	mergeComparisonList := make([]float64,maxK,maxK)

	// init k to be 1
	k = 1

	// loop over all cluster hierarchy lists generated by each distance measure
	for _,levelsPointer := range levelsListPerDistanceMeasure{
		levels := *levelsPointer

		// index for the minimal cluster number that should taken into consideration
		var iMin int

		// calculation can only start from the second level
		if iMin = len(levels) - maxK; iMin < 0 {
			iMin = 0
		}

		// loop over cluster levels in current matrix
		perLevelMergeComparisonList := make([]float64,maxK,maxK)

		for i,_:= range perLevelMergeComparisonList {
			perLevelMergeComparisonList[i] = 1.0
		}

		// consider every clustering for the distance measure
		for i,clusters := range levels {
			if len(clusters) <= maxK {
				var scAverage float64
				for _,cluster := range clusters {
					scI := clustering.ClusterSilhouetteCoefficient(cluster,clusters,pointDistAlgo)
					scAverage += scI
				}
				scAverage /= float64(len(clusters))
				perLevelMergeComparisonList[i-iMin] += scAverage
			}
		}
		for i,_:= range mergeComparisonList {
			mergeComparisonList[i] += perLevelMergeComparisonList[i]
		}
	}

	// final evaluation: take the level k before the largest total delta value as the correct value for the number of clusters
	max := float64(1.0) * float64(len(levelsListPerDistanceMeasure))
	for toStep,deltaTotal := range mergeComparisonList {
		if deltaTotal >= max {
			max = deltaTotal
			// don't take this, but the split before
			k = maxK - toStep
		}
	}

	return
}

// Generates a concrete clustering from a list of clustering levels and k, the number of clusters in the final clustering.
// Calculates the cluster parameters from to full itemset of the original cluster and appends a reduced itemset
// containing only the min and max element of the cluster to the resulting final cluster.
// @return []*halfCluster a list of reduced versions of the original clusters
func fetchClusterLevel(k int, levels *[][]clustering.Cluster,pointDistAlgo clustering.PointDistance)(finalCluster []clustering.Cluster){
	// index of the level where the final clustering is stored
	var level int
	if k < 1 {
		k = 1
	}
	if level = len(*levels)-k; level < 0 {
		level = 0
	}
	finalCluster = make([]clustering.Cluster,0,k)
	for _,cluster := range (*levels)[level] {
		nextCluster := newFullCluster(cluster,pointDistAlgo)
		finalCluster = append(finalCluster,nextCluster)
	}
	return
}

// Tigh loop that sends DataQuery for WaterBufferDelta to an Oracle and
// processes the response
func (learner *waterConsumptionLearner) StreamClustering(logDestination *io.Writer, logMutex *sync.Mutex)(){
	bucketCollection := make(
		map[int]*[]*bucket,
		math.Ilogb(float64(MAX_BUCKET_SIZE))-math.Ilogb(float64(learner.p)),
	)

	// list d_i holds up to p points
	// that represent p successive (1D) temperature deltas
	incomingPoints := make([]*deltaPoint,0,learner.p)

	// generate DataRequest that is sent to the oracle during the loop
	resultEndpoint := make(chan []system.DataResponse)
	query := []system.DataQuery{
		system.WATER_BUFFER_DELTA,
	}
	info := []struct{
		Sec int
		Weight float64
	}{
		{
			Sec:learner.sec,
			Weight:1.0,
		},
	}
	deltaRequest := system.MakeDataRequest(resultEndpoint,query,info)

	var last_timestamp int64 = 0

	// infinite loop
	for {
		// wait until the next dataRequest is issued
		time.Sleep(time.Second * time.Duration(learner.sec-learner.overlap))

		// send data request to oracle
		system.Query_request_chan <- deltaRequest

		// wait for response from oracle
		responses := <- resultEndpoint

		getNextPoint:
		for _,response := range(responses) {

			// ensure that the next data point has not been seen yet
			if response.Considered_data == 0 {
				break getNextPoint
			}

			// ensure that the new data point differs from the last
			if response.TimeStamp == last_timestamp {
				break getNextPoint
			} else {
				last_timestamp = response.TimeStamp
			}

			// generate point with one dimension,
			// set timestamp and coordinate
			// and add to incoming points
			item := NewDeltaPoint(1)

			item.SetTimestamp(response.TimeStamp)

			// normalize delta regarding the length of the considered interval
			// delta/seconds -> value is normalized to temp delta over one second
			value := response.Result / float64(response.Considered_data)
			item.SetDelta(value)

			incomingPoints = append(incomingPoints,item)


			/*@debug
			if len(incomingPoints) % 4 == 0{
				fmt.Printf("[%d| %s]\n",len(incomingPoints),item)
			} else {
				fmt.Printf("[%d| %s] > ",len(incomingPoints),item)
			}
			//@debug_end*/

			// merge buckets if incoming points are full
			if len(incomingPoints) >= learner.p {

				// generate a bucket pointer
				// from incoming points clustering
				next := new(bucket)

				// generate clusterings for a set of distance measures
				next.record,next.itemcount,next.timestamp =
					generateAgglomerativeBucket(
						incomingPoints,
						learner.k,
						//extractBestClusterLevel,		// extractor function
						extractBestClusterLevelSilhouette,
						clustering.WeightedEuclideanDistance,	// point to point distance
						clustering.SingleLink,			// cluster to cluster distance
						clustering.MeanDistance,		// cluster to cluster distance
						clustering.CompleteLink,		// cluster to cluster distance
						)

				//next.record,next.itemcount,next.timestamp = kmeans(learner.k,learner.epsilon,incomingPoints)

				/* @debug
				fmt.Printf("\nResult of initial clustering:\n")
				printCluster(next.record)
				fmt.Println()
				// @debug*/

				// generate a 4D generic Point for the clustering such that clusterings with
				// different numbers of k clusters can be further clusterd [centroid,noise,diam,k]
				// @todo check bucketPoint generation
				point := generateGenericPointForBucket(next)
				fmt.Printf("%v\n",point)
				logMutex.Lock()
				io.Copy(*logDestination, strings.NewReader(fmt.Sprintf("%+v\n",*point)))
				logMutex.Unlock()

				if len(learner.clusteringPointList) >= CLUSTERING_DECAY_HORIZON {
					learner.clusteringPointList = learner.clusteringPointList[:len(learner.clusteringPointList)-1]
				}
				learner.clusteringPointList = append([]*bucketPoint{point}, learner.clusteringPointList...)


				// add to bucket map
				if next != nil {
					// insert clusters into the database
					/*
					for _,c := range next.record {
						if hc,ok := c.(*halfCluster); ok {
							hc.Insert(
								map[string]interface{}{
									"recordtimestamp":next.timestamp,
									"recordsize":next.itemcount,
								},
								clustering.EuclideanDistance,
							)
						}
					}
					*/
					// add bucket to bucketCollection
					sizeKey := next.itemcount
					learner.addAndMergeBuckets(
						next,
						&bucketCollection,
						sizeKey,
						next.timestamp,
						clustering.CentroidDistance,
						clustering.WeightedEuclideanDistance,
					)
				}

				// init new point collection
				incomingPoints = make([]*deltaPoint,0,learner.p)
			}
		}
	}

}
func (learner *waterConsumptionLearner) timeCheckBucket(bucketList *[]*bucket,currentTime int64){
	// checkIndex holds bucket index of items that are safe
	checkIndex := -1
	for checkIndex < len(*bucketList)-1{
		// loop over remaining uncheckd items
		for i:=checkIndex+1;i<len(*bucketList);i++ {
			if (*bucketList)[i].timestamp < currentTime - learner.windowTimeHorizonInSec {
				fmt.Printf("[***********DROPPED]\tBucket %d gets dropped (%d vs %d - %d).\n",
					i,
					(*bucketList)[i].timestamp,
					currentTime,
					currentTime - (*bucketList)[i].timestamp,
				)
				cutOutPointerIndex(bucketList,i)
				break // break inner loop and restart check from checkIndex
			}
			checkIndex = i
		}
	}
}
func (learner *waterConsumptionLearner) addAndMergeBuckets(next *bucket, bucketCollection *map[int]*[]*bucket, sizeKey int, currentTime int64, distAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance)(){
	if next != nil {
		if sizeKey > MAX_BUCKET_SIZE {
			// simply forget the next bucket if its size exceeds max bucket size threshold
			return
		}
		if bucketListPointer, ok := (*bucketCollection)[sizeKey]; ok && bucketListPointer != nil {
			// an entry for this size key exists in the bucket collection

			// first check if old buckets exist that need to be dumped
			learner.timeCheckBucket(bucketListPointer, currentTime)

			if sizeKey < MAX_BUCKET_SIZE {
				// key size of next record is in bounds => key size of potential merger is in bounds
				//fmt.Printf("Bucketcollection of size %d contains pointers to %d buckets.\n",sizeKey,len(*listPointer))
				if len(*bucketListPointer) >= 2 {
					// merge the two oldest cluster records
					recordA := (*bucketListPointer)[0].record
					recordB := (*bucketListPointer)[1].record

					/*//@debug
					fmt.Println("*****************************************************")
					fmt.Printf("Merging two buckets of size %d: [%d/%d]\n\t-%T\n\t-%T\n",
						sizeKey,
						(*bucketListPointer)[0].timestamp,
						(*bucketListPointer)[1].timestamp,
						recordA,
						recordB,
					)
					printCluster(recordA)
					fmt.Println()
					printCluster(recordB)
					//@debug_end*/


					cutOutPointerIndex(bucketListPointer, 1)
					cutOutPointerIndex(bucketListPointer, 0)

					assignment := munkresClusterAssignment(
						&recordA,
						&recordB,
						clustering.CentroidDistance,
						clustering.EuclideanDistance,
					)

					/*//@debug
					fmt.Print("Result of assignment algorithm: ")
					for i, k := range *assignment {
						fmt.Printf("(%d-%d) ", i, k)
					}
					fmt.Println()
					//@debug_end*/

					var nextBucket *bucket
					if sizeKey >= 64 {
						// merge buckets using simple merging heuristig
						// init nextBucket and backupBucket
						var backupBucket *bucket
						nextBucket,backupBucket = combineBuckets(*assignment,recordA,recordB,distAlgo,pointDistAlgo);
						// loop until all clusters have been merged and assigned to nextBucket
						for backupBucket.itemcount > 0 {
							// merged buckets are not optimal, try to find better assignment between backupBucket and nextBucket
							assigned := munkresClusterAssignment(&nextBucket.record, &backupBucket.record, distAlgo, pointDistAlgo)
							nextBucket,backupBucket = combineBuckets(*assigned,nextBucket.record,backupBucket.record,distAlgo,pointDistAlgo);
						}

					} else {
						// merge buckets using a clustering method
						nextBucket = new(bucket)
						points := make([]*deltaPoint,0)
						for _,c := range recordA {
							for _,p := range c.GetClusterItems() {
								points = append(points,p.(*deltaPoint))
							}
						}
						for _,c := range recordB {
							for _,p := range c.GetClusterItems() {
								points = append(points,p.(*deltaPoint))
							}
						}
						if sizeKey >= 32 {
							nextBucket.record,nextBucket.itemcount,nextBucket.timestamp =
								learner.generateKMeansBucket(
									points,
									learner.k,
									//extractBestKMeansLevel,
									extractBestClusterLevelSilhouette,
									clustering.WeightedEuclideanDistance,
									clustering.MeanDistance,		// used to extract best cluster level
								)
						} else {
							// merge buckets using agglomerative procedure
							nextBucket.record,nextBucket.itemcount,nextBucket.timestamp =
								generateAgglomerativeBucket(
									points,
									learner.k,
									//extractBestClusterLevel,		// extractor function
									extractBestClusterLevelSilhouette,
									clustering.WeightedEuclideanDistance,	// point to point distance
									clustering.SingleLink,			// cluster to cluster distance
									clustering.MeanDistance,		// cluster to cluster distance
									clustering.CompleteLink,		// cluster to cluster distance
								)
						}
					}

					/*//@debug
					fmt.Printf("\nResult of cluster merge:\n")
					printCluster(nextBucket.record)
					fmt.Println("Timestamp of the bucket: ",nextBucket.timestamp)
					fmt.Println()
					//@debug_end*/

					// otimal assignmend found, add next bucket to slot of greater sizeKey in collection
					learner.addAndMergeBuckets(nextBucket, bucketCollection, nextBucket.itemcount, currentTime, distAlgo, pointDistAlgo)
				}
			} else {
				// key size reaches max bucket size
				if len(*bucketListPointer) >= 2 {
					// delete the oldest  entry
					var delete_index int
					for i:=0; i<len(*bucketListPointer);i++ {
						if (*bucketListPointer)[i].timestamp > (*bucketListPointer)[delete_index].timestamp {
							delete_index = i
						}
					}

					cutOutPointerIndex(bucketListPointer, delete_index)

					if len(*bucketListPointer) >= 2 {
						log.Fatal(fmt.Sprintf("Oldest timestamp %s, current time %s",(*bucketListPointer)[delete_index].timestamp,currentTime))
						log.Fatal("We will have 3 Buckets that have the maximum length")
					}
				}
			}
		} else {
			// init a new list for 3 buckets of this sizeKey
			list := make([]*bucket, 0, 3)
			(*bucketCollection)[sizeKey] = &list
		}

		// append next bucket to the right slot in the collection
		*(*bucketCollection)[sizeKey] = append(*(*bucketCollection)[sizeKey], next)
	}
	return
}

// In order to further cluster clusterings contained in a bucket this function converts a bucket into a
// point in a multidimensional space such that clusterings with different number k of clusters
// could be compared against each other.
func generateGenericPointForBucket(b *bucket)(p *bucketPoint){
	// generate a new point with 3 dimensions
	p = newBucketPoint(4)

	// set satellite data
	p.SetTime(b.timestamp)
	p.SetVolume(b.itemcount)

	// set vector data
	noise,center := CalcClusteringNoise(b.record,clustering.EuclideanDistance)
	diam := CalcClusteringDiam(b.record,clustering.EuclideanDistance)
	k := len(b.record)
	p.SetCoordinate(0,center.GetCoordinate(0).GetValue().(float64))
	p.SetCoordinate(1,noise)
	p.SetCoordinate(2,diam)
	p.SetCoordinate(3,k)

	return
}

// Tests if there is any intra cluster violation considering for the assignments and merges the clusters
// according to the assignment if no violation is found. In case of violation the algorithm switches the items
// in such a way that the nearest neighbor for at least one pair of clusters is in a different bucket such that
// a successive assignment would merge these two clusters.
// @return nextBucket bucket containing a record of merged clusters
// @return backupBucket bucket containing a nearest neighbor that violated a merge in the current round or an empty bucket
func combineBuckets(assigned []int, recordA,recordB []clustering.Cluster, distAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance)(nextBucket, backupBucket *bucket){
	// init two empty buckets
	nextBucket = new(bucket)
	nextBucket.timestamp = math.MinInt64
	backupBucket = new(bucket)
	backupBucket.timestamp = math.MinInt64

	for a_index, b_index := range assigned {
		// check if assignment index is in bounds sind munkres assignment could work with placeholder rows
		if a_index <= len(recordA)-1 && b_index <= len(recordB)-1 {
			/*@debug_start
			fmt.Printf("Try to merge %+v and %+v\n",recordA[a_index],recordB[b_index])
			//@debug_end*/

			// test if cluster merging is allowed
			mergerIsAllowed := true

			// take distance of centroids between assigned as benchmark for assignment quality check
			distanceBenchmark,_ := recordA[a_index].DistanceTo(recordB[b_index],distAlgo,pointDistAlgo)
			//distanceBenchmark := pointDistAlgo(recordA[a_index].GetCentroid(),recordB[b_index].GetCentroid())

			aNeighbour := testMergerMakesSense(&mergerIsAllowed,a_index,distanceBenchmark,recordA,distAlgo,pointDistAlgo)
			bNeighbour := testMergerMakesSense(&mergerIsAllowed,b_index,distanceBenchmark,recordB,distAlgo,pointDistAlgo)

			/*@debug_start
			fmt.Printf("Merge allowed %v; Distance Benchmark: %.2f\n",mergerIsAllowed,distanceBenchmark)
			//@debug_end*/
			if mergerIsAllowed {
				// generate next record according to assignment and add to bucket
				appendRecordToBucket(
					nextBucket,
					recordA[a_index].CombineWithCluster(recordB[b_index],pointDistAlgo),
				)
			} else {
				/*@debug
				fmt.Printf("Try to merge %+v and %+v\n",a_index,b_index)
				printCluster(recordA)
				fmt.Println()
				printCluster(recordB)
				fmt.Println()
				fmt.Println()
				printCluster(nextBucket.record)
				fmt.Println()
				printCluster(backupBucket.record)
				//@debug_end */

				// put cluster from recordA into backupBucket and recordB into nextBucket
				// since backupBucket becomes recordB in the next iteration step
				if aNeighbour != nil {
					// better merging candidate for recordA is found
					appendAtoBackup := true
					for _,p := range backupBucket.record {
						if aNeighbour == p {
							// the nearest neighbout has been append to backup bucket
							appendAtoBackup = false
							break
						}
					}
					if appendAtoBackup {
						appendRecordToBucket(backupBucket,recordA[a_index])
						appendRecordToBucket(nextBucket, recordB[b_index])
					} else {
						appendRecordToBucket(nextBucket,recordA[a_index])
						appendRecordToBucket(backupBucket, recordB[b_index])
					}
				} else if bNeighbour != nil {
					appendBtoBackup := true
					for _, p := range backupBucket.record {
						if bNeighbour == p {
							// the nearest neighbout has been append to backup bucket
							appendBtoBackup = false
							break
						}
					}
					if appendBtoBackup {
						appendRecordToBucket(nextBucket,recordA[a_index])
						appendRecordToBucket(backupBucket, recordB[b_index])
					} else {
						appendRecordToBucket(backupBucket,recordA[a_index])
						appendRecordToBucket(nextBucket, recordB[b_index])
					}
				}

				/*@debug
				fmt.Println()
				fmt.Println()
				fmt.Println()
				printCluster(nextBucket.record)
				fmt.Println()
				printCluster(backupBucket.record)
				//@debug_end */
			}
		} else if a_index <= len(recordA)-1 {
			appendRecordToBucket(nextBucket,recordA[a_index])

		} else if b_index <= len(recordB)-1 {
			appendRecordToBucket(nextBucket,recordB[b_index])
		}
	}
	return
}

// Searches the nearest neighbor to the cluster at mergeIndex in the record
// considering distances between the cluster centroids for the given point distance measure.
// If there exists a closer cluster than the merging candidate,
// the allowedFalg is set to false and the merging candidate is returned.
// Otherwise the merger is allowed and nil is returned. Therefore a boolean flag is set to
// indicate that a merging violates the invariant that no two clusters are merged if there is a
// better candidate for merging.
// Successive merging should take the boolean flag into consideration before merging the two clusters.
// @return clusterin.Cluster nearest neighbor to record[mergeIndex] or nil
func testMergerMakesSense(allowedFlag *bool, mergeIndex int, distanceBenchmark float64, record []clustering.Cluster, distAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance)(nearestNeighbour clustering.Cluster){
	if mergeCandidate := record[mergeIndex]; *allowedFlag {
		// check for each neighbor cluster if the distance between centroid and merge candidate's centroid is less than benchmark
		for i,neighbourCluster := range record {
			if i != mergeIndex {
				// consider every pairwise distance from merger cluster to any other cluster within the same clustering
				dist,_ := mergeCandidate.DistanceTo(neighbourCluster,distAlgo,pointDistAlgo)
				//dist := pointDistAlgo(neighbourCluster.GetCentroid(),mergeCandidate.GetCentroid())
				if dist < distanceBenchmark {
					// if there is a cluster that is closer within the same clustering
					// than the one which is choosen for merging then a merge is not allowed
					*allowedFlag = false
					distanceBenchmark = dist
					nearestNeighbour = neighbourCluster
				}
			}
		}
	}
	return
}

// Appends the given record to the bucket. Updates timestamp if required and the itemcount field of the bucket.
// This method ensures that a valid timestamp is assigned to the nextBucket iff nextRecord's centroid is a pointer
// to a deltaPoint which has a valid timestamp
func appendRecordToBucket(nextBucket *bucket, nextRecord clustering.Cluster)(){
	if nextRecord != nil {
		if centroid := nextRecord.GetCentroid(); centroid != nil {
			if point,ok := centroid.(*deltaPoint); ok {
				if ts := point.timestamp; nextBucket.timestamp < ts {
					nextBucket.timestamp = ts
				}

			}
		}
	}
	nextBucket.itemcount += nextRecord.GetClusterSize()
	nextBucket.record = append(nextBucket.record, nextRecord)
}

func printCluster(clustering []clustering.Cluster){
	for _,cluster := range clustering {
		switch v := cluster.(type) {
		case *fullCluster:
			fmt.Printf("Diam:%.2f (size: %d)\t%v\t%+v\tRad:%.2f\tAvg:%.2f\tDens: %.2f\n",(*v).diameter,(*v).GetClusterSize(),(*v).centroid,(*v).GetClusterItems(),(*v).radius,(*v).averagePairOfPoints,(*v).density)
		//case *halfCluster:
		//	fmt.Printf("HalfCluster: Diam:%.2f (size: %d)\t%v\t%+v\tRad:%.2f\tAvg:%.2f\tDens: %.2f\n",(*v).diameter,cluster.GetClusterSize(),(*v).centroid,(*v),(*v).radius,(*v).averagePairOfPoints,(*v).density)
		default:
			fmt.Printf("%T: size:%d\tCentroid:%v\n",v,v.GetClusterSize(),v.GetCentroid())
		}
	}
}

// Transforms []*deltaPoint to []clustering.Point by creating a new slice of the generic interface
// type and appendign the deltaPoint pointers to it.
// This method should be used when a cast from a slice with interface instantiations to a slice of interface
// is required (e.g. when clustering a slice of []*deltaPoint)
func GenerateInterfaceSliceFromDeltaSlice(points []*deltaPoint)([]clustering.Point){
	var tmp []clustering.Point = make([]clustering.Point,len(points))
	for i,p := range points {
		//tmp[i] = p
		tmp[i] = clustering.Point(p)
	}
	return tmp
}
