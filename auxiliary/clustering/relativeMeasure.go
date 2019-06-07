package clustering

import (
)

func MeanIn(xI Point, cluster Cluster, pointDistAlgo PointDistance)(mean float64){
	if cluster.GetClusterSize() > 1 {
		for _,xJ := range cluster.GetClusterItems() {
			if xJ != xI {
				mean += pointDistAlgo(xI,xJ)
			}
		}
		mean /= float64(cluster.GetClusterSize()-1)
	}
	return
}

func MeanOutMinIn(xI Point, clustering []Cluster, pointDistAlgo PointDistance)(meanOutMin,meanIntern float64){
	meanOutSet := false
	for _,cluster := range clustering {
		if !cluster.HasMember(xI) {
			var mean float64
			for _,xJ := range cluster.GetClusterItems() {
				mean += pointDistAlgo(xI,xJ)
			}
			mean /= float64(cluster.GetClusterSize())
			if !meanOutSet || mean < meanOutMin {
				meanOutMin = mean
				meanOutSet = true
			}
		} else {
			meanIntern = MeanIn(xI,cluster,pointDistAlgo)
		}
	}
	return
}

func PointSilhouetteCoefficient(xI Point,clustering []Cluster,pointDistAlgo PointDistance)(sI float64){
	meanOutMin, meanInternal := MeanOutMinIn(xI,clustering,pointDistAlgo)
	var max float64
	if max = meanOutMin; max < meanInternal {
		max = meanInternal
	}
	sI += meanOutMin
	sI -= meanInternal
	if max != float64(0){
		sI /= max
	}
	return
}

func ClusterSilhouetteCoefficient(cluster Cluster,clustering []Cluster, pointDistAlgo PointDistance)(sci float64){
	for _,xJ := range cluster.GetClusterItems() {
		sci += PointSilhouetteCoefficient(xJ,clustering,pointDistAlgo)
	}
	sci /= float64(cluster.GetClusterSize())
	return
}