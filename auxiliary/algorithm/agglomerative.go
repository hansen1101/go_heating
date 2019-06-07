package algorithm

import (
	"log"
	"math"
	"container/heap"
	"github.com/hansen1101/go_heating/auxiliary/datatype"
	"github.com/hansen1101/go_heating/auxiliary/clustering"
)

type nodeData struct {
	a clustering.Cluster
	b clustering.Cluster
}
type LevelExtractor func(int, clustering.DistanceMeasure, clustering.PointDistance, ...*[][]clustering.Cluster)(int)

// Takes a set of n points and a distance measure and computes a hierarchical clustering containing n levels.
func AgglomerativeClustering(points []clustering.Point,clusterDistAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance)(clusterlevels *[][]clustering.Cluster){
	var distanceMatrix *datatype.PriorityQueue
	var clusterToItemPointer *map[clustering.Cluster][]*datatype.Item
	var clusterLevel [][]clustering.Cluster
	clusterLevel = make([][]clustering.Cluster,0,0)

	// init 1 item clusters
	clusterLevel = append(clusterLevel,initClustersFromPoints(points))

	// init distance matrix
	distanceMatrix,clusterToItemPointer = initClusterDistanceQueue(clusterLevel[0],clusterDistAlgo,pointDistAlgo)

	// pop next merger clusters, delete associated nodes from queue, update distances
	// until full clustering is generated
	for distanceMatrix.Len() > 0 {

		// pop first node containing clusters to merge
		mergerNode := heap.Pop(distanceMatrix).(*datatype.Item).GetValue().(*nodeData)
		mergerClusters := make([]clustering.Cluster, 0, 2)
		mergerClusters = append(mergerClusters, mergerNode.a)
		mergerClusters = append(mergerClusters, mergerNode.b)

		newCluster := clustering.NewCluster()
		for _,mergeClusters := range mergerClusters {
			for _,p := range mergeClusters.GetClusterItems() {
					newCluster.AddItem(p)
			}
		}

		//newclustering.Cluster.UpdateData(pointDistAlgo)
		newCluster.UpdateCentroid()

		// add the new cluster to the pointer dict
		(*clusterToItemPointer)[newCluster] = make([]*datatype.Item, 0, 0)

		// delete all affected node
		for _, oldClusterDictKey := range mergerClusters {
			if itemList, ok := (*clusterToItemPointer)[oldClusterDictKey]; ok {
				for _, item := range itemList {
					if item.Index >= 0 {
						distanceMatrix.Update(item, item.GetValue(), int(math.MinInt32))
						heap.Pop(distanceMatrix)
					}
				}
			}
			delete((*clusterToItemPointer), oldClusterDictKey)
		}

		// generate next cluster level and append clusters and add node of new cluster to distance matrix
		clusterLevel = append(clusterLevel, make([]clustering.Cluster, 0))
		clusterLevel[len(clusterLevel) - 1] = append(clusterLevel[len(clusterLevel) - 1], newCluster)

		// append all other clusters from the pointer dict to the next level that surived the deletion step
		for cluster, _ := range *clusterToItemPointer {
			if cluster != newCluster {
				clusterLevel[len(clusterLevel) - 1] = append(clusterLevel[len(clusterLevel) - 1], cluster)
				generateAndLinkDistanceQueueNodes(newCluster, cluster, clusterDistAlgo, pointDistAlgo, distanceMatrix, clusterToItemPointer)
			}
		}
	}
	clusterlevels = &clusterLevel
	return
}

// Initializes empty list for saving node pointers a cluster is associated with in the clusterToItemPointer map. Generates a new auxiliary Item
// from the two given clusters a and b, adds it to the priority queue, updates it considering the distanceMeasure between a and b and appends the
// pointer to the node into the pointer lists of a and b.
// Helper function for the agglomerative clustering approach.
func generateAndLinkDistanceQueueNodes(a,b clustering.Cluster, clusterDistAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance, pq *datatype.PriorityQueue, clusterToItemPointer *map[clustering.Cluster][]*datatype.Item){
	// init list for a and b in pointer dict if necessary
	if _,okA := (*clusterToItemPointer)[a]; !okA {
		(*clusterToItemPointer)[a] = make([]*datatype.Item,0,0)
	}
	if _,okB := (*clusterToItemPointer)[b]; !okB {
		(*clusterToItemPointer)[b] = make([]*datatype.Item,0,0)
	}

	item := &datatype.Item{}
	value := &nodeData{a,b}

	// execute the clusterDistAlgo for clustering (e.g. singleLink) for the clusters a and b given an algo
	// for computing distances between pairs of points
	dist,err := a.DistanceTo(b,clusterDistAlgo,pointDistAlgo)
	if err != nil {
		log.Fatal("There was an error with the DistanceTo function when generating cluster merge item.")
	}

	// set the priority for the new item and push it to the queue
	prio := int(float64(2048) * dist)

	heap.Push(pq,item)
	(*clusterToItemPointer)[a] = append((*clusterToItemPointer)[a],item)
	(*clusterToItemPointer)[b] = append((*clusterToItemPointer)[b],item)
	pq.Update(item,value,prio)
}

// Initializes distance queue and pointer list for each item and executes the generateclustering.ClusterMergeItem algorithm exactly once for
// each cluster combination. The priority of a node is calculated from the clusters and the given distance measure.
// Helper function for the agglomerative clustering approach.
// @return PriorityQueue where every node represents the distance between a distinct pair of clusters
func initClusterDistanceQueue(clusters []clustering.Cluster,clusterDistAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance)(*datatype.PriorityQueue, *map[clustering.Cluster][]*datatype.Item){
	n := len(clusters) // equals n
	n = (n*(n-1))/2 // equals n over 2 = n!/(2!*(n-2)!) which is the number of nodes in the queue
	clusterToItemPointer := make(map[clustering.Cluster][]*datatype.Item)
	distanceQueue := make(datatype.PriorityQueue,0,n)
	heap.Init(&distanceQueue)
	// consider all possible cluster combinations exactly once
	for i:=0;i<len(clusters);i++{
		for j:=i+1;j<len(clusters);j++{
			generateAndLinkDistanceQueueNodes(clusters[i],clusters[j],clusterDistAlgo,pointDistAlgo,&distanceQueue,&clusterToItemPointer)
		}
	}
	return &distanceQueue, &clusterToItemPointer
}

// Generates a list of clusters containing exactly one point from a ginven list of points.
// @return []clustering.Cluster the first cluster level where every cluster consists of only a single point
func initClustersFromPoints(points []clustering.Point)(clusterLevel []clustering.Cluster){
	clusterLevel = make([]clustering.Cluster,0,0)
	for _,p := range points {
		tmp := clustering.NewCluster()
		tmp.AddItem(p)
		tmp.UpdateCentroid()
		clusterLevel = append(clusterLevel,tmp)
	}
	return
}
