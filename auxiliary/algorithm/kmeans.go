package algorithm

import (
	"math"
	"fmt"
	"errors"
	"math/rand"
	"github.com/hansen1101/go_heating/auxiliary/clustering"
)

// Produces a clustering of the given points into k clusters using the k-means clustering approach. The parameter epsilon
// is used to test for convergence of the cluster assignment step.
// @return []cluster the final clustering
// @return int the number of points considered in the clustering
// @return int64 the timestamp of the latest point taken into consideration
func KMeans(k int, epsilon float64, points []clustering.Point, clusterDistAlgo clustering.DistanceMeasure, pointDistAlgo clustering.PointDistance)(finalCluster []clustering.Cluster, err error){
	t := 0

	// start with centroids slices for 4 iterations
	centroids := make([][]clustering.Point,0,4)

	if k <= len(points){

		// init k distinct centroids randomly
		randomCentroids,initErr := initKRandomCentroids(points,k,pointDistAlgo)
		if initErr != nil {
			err = initErr
			return
		}
		centroids = append(
			centroids,
			randomCentroids,
		)

		//@todo check and adjust k if centroids do not differ

		// kmean iteration
		var clusters []clustering.Cluster
		kmeansIteration:
		for {
			t++

			/*//@debug
			fmt.Printf("Kmeans Iteration #%d\n",t)
			//@debug*/

			// cluster assignment
			clusters = clusterAssignment(points,centroids[t-1],pointDistAlgo)

			// centroid update
			centroids = append(centroids,make([]clustering.Point,k,k))
			for i,cluster := range clusters {
				cluster.UpdateCentroid()
				centroids[t][i] = cluster.GetCentroid()
			}

			// check for abort criterion
			sum := float64(0.0)
			for i := 0; i < k; i++ {
				sum += pointDistAlgo(centroids[t-1][i],centroids[t][i])
			}
			if sum <= epsilon {
				break kmeansIteration
			}
		}

		// final assignment
		finalCluster = clusterAssignment(points,centroids[t],pointDistAlgo)

	} else {
		err = errors.New("Not enough points in the set to search for k clusters.")
	}
	return
}

// Helper method for k-means algorithm. Assigns points to clusters according to the minimum euclidean distance
// between the point and any of the centroids representing the cluster.
func clusterAssignment(points []clustering.Point, centroids []clustering.Point, distAlgo clustering.PointDistance)(clusters []clustering.Cluster){
	// init k clusters; one for each centroid
	clusters = make([]clustering.Cluster,len(centroids),len(centroids))

	// cluster assignment
	for _,point := range points {
		var index int	// index of the centroid this point gets assigned to
		var closest float64	// closest current distance found
		index = -1

		// find centroid that is closest to point
		for j,centroid := range(centroids){
			if sse := distAlgo(point,centroid); index < 0 {
				// first loop; initially assign this point to the first centroid
				index = j
				closest = sse
			} else {
				if sse < closest {
					// a better centroid for assignment is found
					index = j
					closest = sse
				} else if sse == closest && clusters[j] == nil {
					// ensure that at least one point gets assigned to each centroid
					index = j
				}
			}
		}

		// assign point to cluster of closest centroid
		if clusters[index] == nil {
			// init new cluster if slot points to nil
			clusters[index] = clustering.NewCluster()
		}
		clusters[index].AddItem(point)
	}
	fixAssignment(clusters,distAlgo)
	return
}

// Ensures that each slot in the clusters slice contains a cluster of size at least 1.
func fixAssignment(clusters []clustering.Cluster,distAlgo clustering.PointDistance)(){
	for i,c := range clusters {
		if c == nil {
			// assign at least one point from the other clusters to this
			clusterIndex := -1
			var pointIndex clustering.Point
			sItmp := float64(0)
			for j,otherCluster := range clusters {
				if otherCluster != nil && otherCluster.GetClusterSize() > 1 {
					for _,p := range otherCluster.GetClusterItems() {
						tmp := clustering.MeanIn(p,otherCluster,distAlgo)
						if tmp >= sItmp {
							sItmp = tmp
							clusterIndex = j
							pointIndex = p
						}
					}
				}
			}
			clusters[clusterIndex].DeleteItem(pointIndex)
			clusters[i] = clustering.NewCluster()
			clusters[i].AddItem(pointIndex)
		}
	}
}

// Selects k centroids out of a set of points according to some point distance measure which is used as
// heuristic for a suitable distribution of centroids in the vector space. As long as the set contains
// k different points, k different centroids will be found.
// @throws error if the number of points in the set is less than k
// @info successive procedure should check and adjust k if centroids do not differ
func initKRandomCentroids(points []clustering.Point, k int, pointDistAlgo clustering.PointDistance)(centroids []clustering.Point,err error){
	// init centroids slice and check for error condition
	centroids = make([]clustering.Point,0,k)

	if len(points) == 0 {
		err = errors.New("Could not init k centroids out of an empty slice of points.")
		return
	}

	// pick first point randomly and append to centroids
	first := rand.Intn(len(points))
	centroids = append(centroids,points[first])

	// append more centroids until k centroids where chosen
	for len(centroids)<k {
		// find point whose minimum distance to any centroids is max and save index of that point
		max,index := float64(0),-1
		for i,nextpoint := range points {
			// minimum distance of this point to a centroid
			min := math.MaxFloat64
			for _,centroid:=range centroids{
				if distance := pointDistAlgo(nextpoint,centroid); distance < min {
					min = distance
				}
			}
			// update the index to the point which currently is the best centroid candidate
			if min > max {
				max = min
				index = i
			}
		}

		// append that point to the slice of centroids
		if index >= 0 {
			centroids = append(centroids,points[index])
		} else {
			// case: all points have 0 distance to the centroids, choose next centroid randomly
			any := rand.Intn(len(points))
			centroids = append(centroids,points[any])

			//@debug
			fmt.Println("Could not find a point for kmeans initialization, so pick one randomly.")
			//@debug_end
		}
	}
	return
}
