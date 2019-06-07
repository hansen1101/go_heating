package learner

import (
	"github.com/hansen1101/go_heating/auxiliary/clustering"
	"fmt"
	"math"
	"errors"
)

type fullCluster struct {
	clustering.Cluster
	*simpleCluster
}
func newFullCluster(cluster clustering.Cluster, distAlgo clustering.PointDistance)(obj *fullCluster){
	obj = new(fullCluster)
	obj.Cluster = cluster

	// generate the simpleCluster
	obj.simpleCluster = new(simpleCluster)

	obj.clustersize = cluster.GetClusterSize()

	obj.UpdateData(distAlgo)

	return
}

func (cluster *fullCluster) CombineWithCluster(other clustering.Cluster,distAlgo clustering.PointDistance)(clustering.Cluster){
	if other != nil {
		tmp := cluster.Cluster.CombineWithCluster(other,distAlgo)
		c := newFullCluster(tmp,distAlgo)
		return c
	} else {
		return cluster
	}
}
func (cluster *fullCluster) DistanceTo(other clustering.Cluster,clusterDistAlgo clustering.DistanceMeasure,pointDistAlgo clustering.PointDistance)(distance float64,err error){
	if other != nil {
		if v,ok := other.(*fullCluster); ok {
			distance,err = cluster.Cluster.DistanceTo(v.Cluster,clusterDistAlgo,pointDistAlgo)
		} else {
			distance,err = cluster.Cluster.DistanceTo(other,clusterDistAlgo,pointDistAlgo)
		}
	} else {
		distance = math.MaxFloat64
		err = errors.New("Distance between two different cluster types is not defined.")
	}
	return
}

func (cluster *fullCluster) GetClusterSize()(int){
	if cluster.simpleCluster != nil {
		return cluster.simpleCluster.GetClusterSize()
	} else if cluster.Cluster != nil {
		return cluster.Cluster.GetClusterSize()
	} else {
		return 0
	}
}
func (cluster *fullCluster) UpdateCentroid()(){
	// loop over all points in cluster and assign largest timestamp to centroid
	var tmp int64 = 0
	for _,p := range cluster.GetClusterItems() {
		if v,ok := p.(*deltaPoint); ok {
			if v.timestamp > tmp {
				tmp = v.timestamp
			}

		} else {
			fmt.Println("The returned cluster does not contain deltaPoints...")
		}
	}
	cluster.SetCentroid(cluster.Cluster.GetCentroid(),tmp)
}
func (cluster *fullCluster) GetCentroid()(clustering.Point){return cluster.centroid}

// Average distance between all pairs of points in the cluster
func (cluster *fullCluster) UpdateAveragePairOfPoints(distAlgo clustering.PointDistance){
	cluster.averagePairOfPoints = cluster.Cluster.GetAveragePairOfPoints(distAlgo)
}
func (cluster *fullCluster) getAveragePairOfPoints(distAlgo clustering.PointDistance)(float64){return cluster.Cluster.GetAveragePairOfPoints(distAlgo)}

// Diameter is the max distance between two points in the cluster
func (cluster *fullCluster) UpdateDiameter(distAlgo clustering.PointDistance){
	diam,min,max := cluster.GetDiameter(distAlgo)
	cluster.diameter,cluster.min,cluster.max = diam,min.(*deltaPoint),max.(*deltaPoint)
}
func (cluster *fullCluster) GetDiameter(distAlgo clustering.PointDistance)(float64, clustering.Point, clustering.Point){
	var Min, Max *deltaPoint = cluster.min, cluster.max
	diam,min,max := cluster.Cluster.GetDiameter(distAlgo)
	if Min == nil || distAlgo(min,cluster.min) != float64(0) {
		Min = CloneDeltaPoint(min)
	}
	if Max == nil || distAlgo(max,cluster.max) != float64(0) {
		Max = CloneDeltaPoint(max)
	}
	return diam,Min,Max
}

// Calculates the ratio points / diameter
func (cluster *fullCluster) UpdateDensity(distAlgo clustering.PointDistance){
	cluster.density = cluster.Cluster.GetDensity(distAlgo)
}
func (cluster *fullCluster) GetDensity(distAlgo clustering.PointDistance)(float64){return cluster.Cluster.GetDensity(distAlgo)}

// Radius is the max distance between a point and the centroid
func (cluster *fullCluster) SetRadius(distAlgo clustering.PointDistance){
	rad,sat := cluster.GetRadius(distAlgo)
	cluster.radius,cluster.satelite = rad,sat.(*deltaPoint)
}
func (cluster *fullCluster) GetRadius(distAlgo clustering.PointDistance)(float64, clustering.Point){
	var Sat *deltaPoint = cluster.satelite
	rad,sat := cluster.Cluster.GetRadius(distAlgo)
	if distAlgo(sat,cluster.satelite) != float64(0) {
		Sat = CloneDeltaPoint(sat)
	}
	return rad,Sat
}

func (cluster *fullCluster) UpdateData(distAlgo clustering.PointDistance){
	cluster.UpdateCentroid()
	cluster.UpdateAveragePairOfPoints(distAlgo)
	cluster.UpdateDiameter(distAlgo)
	cluster.UpdateRadius(distAlgo)
	cluster.UpdateDensity(distAlgo)
}
