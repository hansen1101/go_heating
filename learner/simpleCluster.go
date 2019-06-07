package learner

import(
	"github.com/hansen1101/go_heating/auxiliary/clustering"
	"errors"
)

type simpleCluster struct {
	clustersize int
	centroid *deltaPoint
	satelite, min,max *deltaPoint
	averagePairOfPoints, density, radius, diameter float64
}
func (cluster *simpleCluster) DistanceTo(other *simpleCluster, pointDistAlgo clustering.PointDistance)(distance float64, err error){
	if other != nil {
		distance = pointDistAlgo(cluster.centroid,other.centroid)
	} else {
		err = errors.New("Distance between two different cluster types is not defined.")
	}
	return
}

func (cluster *simpleCluster) GetCentroid()(clustering.Point){return cluster.centroid}
func (cluster *simpleCluster) SetCentroid(centroid clustering.Point,timestamp int64){
	cluster.centroid = CloneDeltaPoint(centroid)
	cluster.centroid.timestamp = timestamp
}

func (cluster *simpleCluster) GetClusterSize()(int){return cluster.clustersize}
func (cluster *simpleCluster) SetClusterSize(value int)(){cluster.clustersize = value}

// Generates a new simpleCluster from this cluster and another cluster object regarding the clustering.PointDistance.
func (cluster *simpleCluster) CombineWithCluster(other *simpleCluster,distAlgo clustering.PointDistance)(*simpleCluster){
	c := new(simpleCluster)
	c.clustersize = cluster.GetClusterSize()
	if cluster.centroid != nil {
		c.SetCentroid(
			NewDeltaPoint(
				cluster.GetCentroid().Dimensions(),
			),
			cluster.GetCentroid().(*deltaPoint).timestamp,
		)
	}
	if other != nil {
		// combine cluster and other if other is not nil
		c.clustersize += other.GetClusterSize()
		// calculate mean value of the two old centroids for each dimension and assign to new centroid
		for dimension,value := range cluster.GetCentroid().GetVector() {
			if v,ok := value.GetValue().(float64); ok {
				if w,ok1 := other.GetCentroid().GetVector()[dimension].GetValue().(float64); ok1 {
					tmp := (v * float64(cluster.GetClusterSize()) + w * float64(other.GetClusterSize())) / float64(c.GetClusterSize())
					c.GetCentroid().SetCoordinate(dimension,tmp)
				}

			}
		}
		if c.GetCentroid().(*deltaPoint).timestamp < other.GetCentroid().(*deltaPoint).timestamp {
			c.GetCentroid().(*deltaPoint).timestamp = other.GetCentroid().(*deltaPoint).timestamp
		}
	} else {
		// copy cluster if other is nil
		for dimension,value := range cluster.GetCentroid().GetVector() {
			c.GetCentroid().(*clustering.GenericPoint).SetCoordinate(dimension,value)
		}
	}

	return c
}

func (cluster *simpleCluster) getAveragePairOfPoints()(float64){return cluster.averagePairOfPoints}
func (cluster *simpleCluster) GetDiameter()(float64, clustering.Point, clustering.Point){return cluster.diameter,cluster.min,cluster.max}
func (cluster *simpleCluster) GetDensity()(float64){return cluster.density}
func (cluster *simpleCluster) GetRadius()(float64, clustering.Point){return cluster.radius, cluster.satelite}
