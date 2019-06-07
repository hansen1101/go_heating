package learner

import(
	"testing"
	"errors"
	"fmt"
	"time"
	"math/rand"
	"github.com/hansen1101/go_heating/auxiliary/clustering"
	"github.com/hansen1101/go_heating/auxiliary/algorithm"
)

type cutOutTestCase struct {
	values []int
	index int
	target []int
	err error
}

var cutOutTests = []cutOutTestCase{
	{ []int{1,2}, 0, []int{2}, nil},
	{ []int{1,2}, 1, []int{1}, nil},
	{ []int{1,2}, 2, []int{1,2}, errors.New(CUTOUTERROR)},
	{ []int{1,2}, -1, []int{1,2}, errors.New(CUTOUTERROR)},
	{ []int{1,2,3,4,5,6}, 0, []int{2,3,4,5,6}, nil},
	{ []int{1,2,3,4,5,6}, 1, []int{1,3,4,5,6}, nil},
	{ []int{1,2,3,4,5,6}, 5, []int{1,2,3,4,5}, nil},
	{ []int{1,2,3,4,5,6}, 6, []int{1,2,3,4,5,6}, errors.New(CUTOUTERROR)},
	{ []int{1,2,3,4,5,6}, 3, []int{1,2,3,5,6}, nil},
	{ nil, 3, nil, errors.New(CUTOUTERROR)},
}

type clusterTestCase struct {
	points []clustering.Point
}

var clusterTests []clusterTestCase = []clusterTestCase{
	clusterTestCase{
		points: []clustering.Point{
			&clustering.GenericPoint{clustering.NewFloat(4.13)},
			&clustering.GenericPoint{clustering.NewFloat(4.2)},
			&clustering.GenericPoint{clustering.NewFloat(4.85)},
			&clustering.GenericPoint{clustering.NewFloat(-4.5)},
			&clustering.GenericPoint{clustering.NewFloat(-4.2)},
			&clustering.GenericPoint{clustering.NewFloat(-5.17)},
			&clustering.GenericPoint{clustering.NewFloat(-4.77)},
			&clustering.GenericPoint{clustering.NewFloat(-4.43)},
			&clustering.GenericPoint{clustering.NewFloat(-5.25)},
			&clustering.GenericPoint{clustering.NewFloat(4.43)},
			&clustering.GenericPoint{clustering.NewFloat(4.13)},
			&clustering.GenericPoint{clustering.NewFloat(0.11)},
			&clustering.GenericPoint{clustering.NewFloat(0.2)},
			&clustering.GenericPoint{clustering.NewFloat(-0.3)},
			&clustering.GenericPoint{clustering.NewFloat(-0.2)},
			&clustering.GenericPoint{clustering.NewFloat(-0.05)},
			&clustering.GenericPoint{clustering.NewFloat(10)},
			&clustering.GenericPoint{clustering.NewFloat(12)},
			&clustering.GenericPoint{clustering.NewFloat(11)},
			&clustering.GenericPoint{clustering.NewFloat(8)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
		},
	},
	///*
	clusterTestCase{
		points: []clustering.Point{
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
		},
	},
	//*/
}

var oracle = rand.New(rand.NewSource(0))

var deltas = []float64{
	float64(20),
	float64(21), //1
	float64(18), //2
	float64(16.8),
	float64(-22), //4
	float64(-42.2),
	float64(-32.1),
	float64(0),
	float64(0), //8
	float64(0.35),
	float64(3),
	float64(4),
	float64(5),
	float64(-1),
	float64(-3.75),
	float64(-9),
	float64(-2.9), //16
	float64(9.3),
	float64(-7.8),
	float64(-9),
	float64(-11),
	float64(-8.3),
	float64(-1.75),
	float64(-0.5),
	float64(1.5), //24
	float64(12.6),
}
var points []*deltaPoint = make([]*deltaPoint,0)

type markedTestCase struct {
	attachList *[]int
	detachList *[]int
	attachResultList *[]int
	detachResultList *[]int
	value int
}
var markedTestCases = []markedTestCase{
	{
		&[]int{1,2,3,4,5,6},
		&[]int{7,8,9},
		&[]int{1,2,3,4,5,6,7},
		&[]int{8,9},
		7,
	},
	{
		&[]int{1,2,3,4,5,6},
		&[]int{7,8,9},
		&[]int{1,2,3,4,5,6},
		&[]int{7,8,9},
		12,
	},
	{
		&[]int{1,2,3,4,5,6},
		&[]int{6,6,9},
		&[]int{1,2,3,4,5,6,6},
		&[]int{6,9},
		6,
	},
	{
		nil,
		&[]int{6,6,9},
		nil,
		&[]int{6,6,9},
		6,
	},
	{
		&[]int{6,6,9},
		nil,
		&[]int{6,6,9},
		nil,
		6,
	},
	{
		&[]int{6,6,9},
		&[]int{6,6,9},
		&[]int{6,6,9},
		&[]int{6,6,9},
		1,
	},
}

func GenerateNDeltaPoints(n int)([]*deltaPoint){
	var points []*deltaPoint = make([]*deltaPoint,n)
	for i:=0;i<n;i++ {
		tmp := NewDeltaPoint(1)
		tmp.SetTimestamp(time.Now().Unix())
		tmp.SetDelta(oracle.Float64())
		points[i]=tmp
	}
	return points
}

func GenerateClustersFromDeltaPoints(k int, points []*deltaPoint)([]clustering.Cluster){
	var sample []clustering.Cluster = make([]clustering.Cluster,k)
	for i:=0;i<k;i++{
		size := oracle.Intn(16) + 4
		sample[i] = clustering.NewCluster()
		for j:=0; j<size; j++ {
			next := oracle.Intn(len(points))
			sample[i].AddItem(points[next])
		}
	}
	return sample
}

func GenerateBucketsFromPoints(k int)([]*bucket){
	var sample []*bucket = make([]*bucket,k)
	for i:=0;i<k;i++{
		tmp := new(bucket)
		clusterCount := oracle.Intn(4)+2
		clusterings := GenerateClustersFromDeltaPoints(clusterCount,GenerateNDeltaPoints(100))
		for _,c := range clusterings {
			tmp.record = append(tmp.record,newFullCluster(c,clustering.EuclideanDistance))
		}
		sample[i] = tmp
	}
	return sample
}

func InitDeltaPoints()(){
	for _,i:=range deltas {
		tmp := NewDeltaPoint(1)
		tmp.SetTimestamp(time.Now().Unix())
		tmp.SetDelta(i)
		points = append(points,tmp)
	}
}

func compError(a,b error) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Error() != b.Error(){
		return false
	}
	return true
}

func testEq(a, b *[]int) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(*a) != len(*b) {
		return false
	}

	for i := range *a {
		if (*a)[i] != (*b)[i] {
			return false
		}
	}

	return true
}

func estInitClusterDistanceQueue(t *testing.T){
	fmt.Println("Performing TestInitClusterDistanceQueue() method")
	InitDeltaPoints()
	//dist := clustering.CompleteLink
	dist1 := clustering.SingleLink
	c := algorithm.AgglomerativeClustering(GenerateInterfaceSliceFromDeltaSlice(points),dist1,clustering.WeightedEuclideanDistance)
	for i,k := range *c {
		fmt.Println(i,len(k))
	}
	k := extractBestClusterLevel(5,dist1,clustering.EuclideanDistance,c)
	l := fetchClusterLevel(k,c,clustering.EuclideanDistance)
	fmt.Println()
	fmt.Printf("Best k value: %d\n",k)
	printCluster(l)
	fmt.Println("*****************************************************")
}

func estPointDistance(t *testing.T){
	fmt.Println("Performing TestPointDistance() method")
	points := GenerateNDeltaPoints(5)
	for _,p := range points {
		for _,o := range points {
			if dist:=clustering.WeightedEuclideanDistance(p,o); dist != float64(0)  {
				//fmt.Printf("Distance between %s - %s: %.2f\n",p,o,dist)
			}
		}
	}
}
func estClusterDistance(t *testing.T){
	fmt.Println("Performing TestClusterDistance() method")
	points := GenerateNDeltaPoints(100)
	clusters := GenerateClustersFromDeltaPoints(2,points)
	for _,p := range clusters {
		for _,o := range clusters {
			if dist,err:=p.DistanceTo(o,clustering.SingleLink,clustering.WeightedEuclideanDistance); dist==float64(0) || err == nil {
				fmt.Printf("Distance between %s - %s: %.2f\n",p,o,dist)
			}
		}
	}
}
func TestKMeansClustering(t *testing.T){
	fmt.Println("Performing TestKMeansClustering() method")

	learner := NewWaterConsumptionLearner(5,0.01,16,16,8)

	for _,testcase := range clusterTests {
		points := make([]*deltaPoint,0)
		for _,p := range testcase.points {
			points = append(points,CloneDeltaPoint(p))
		}
		finalCluster,itemcount,timestamp := learner.generateKMeansBucket(
			points,
			learner.k,
			extractBestClusterLevelSilhouette,
			clustering.EuclideanDistance,
			clustering.MeanDistance,
		)
		fmt.Println("*******************************************")
		fmt.Println(itemcount,timestamp)
		printCluster(finalCluster)
		fmt.Println("*******************************************")
		fmt.Println("*******************************************")
	}


}
func estAggloClustering(t *testing.T){
	fmt.Println("Performing TestKMeansClustering() method")

	learner := NewWaterConsumptionLearner(5,0.01,16,16,8)

	for _,testcase := range clusterTests {
		points := make([]*deltaPoint,0)
		for _,p := range testcase.points {
			points = append(points,CloneDeltaPoint(p))
		}
		finalCluster,itemcount,timestamp := generateAgglomerativeBucket(
			points,
			learner.k,
			extractBestClusterLevel,
			clustering.WeightedEuclideanDistance,
			clustering.MeanDistance,
		)
		fmt.Println("*******************************************")
		fmt.Println(itemcount,timestamp)
		printCluster(finalCluster)
		fmt.Println("*******************************************")
		fmt.Println("*******************************************")
	}


}
func estSilhouette(t *testing.T) {
	cluster := clustering.NewCluster()
	for _,p := range clusterTests[0].points {
		cluster.AddItem(p)
	}
	intern,extern:=clustering.MeanOutMinIn(cluster.GetClusterItems()[8],[]clustering.Cluster{cluster},clustering.EuclideanDistance)
	fmt.Println(
		intern,
		extern,
	)
}
