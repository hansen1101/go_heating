package clustering

import (
	"math"
	"errors"
	"reflect"
	"github.com/hansen1101/go_heating/auxiliary"
	"fmt"
)

const (
	DENS_KEY = "density"
	DIAM_KEY = "diameter"
	RAD_KEY = "radius"
	APP_KEY = "averagePairOfPoints"
	CENT_KEY = "centroid"
	SAT_KEY = "radiusSat"
	MIN_KEY = "diamMin"
	MAX_KEY = "diamMax"
	SIZE_KEY = "size"
)

type Cluster interface {
	DistanceTo(Cluster,DistanceMeasure,PointDistance)(float64,error)
	GetClusterItems()([]Point)
	GetClusterSize()(int)
	CombineWithCluster(Cluster,PointDistance)(Cluster)
	AddItem(Point)()
	DeleteItem(Point)()
	GetCentroid()(Point)
	UpdateCentroid()()
	GetAveragePairOfPoints(PointDistance)(float64)
	UpdateAveragePairOfPoints(PointDistance)()
	GetDiameter(PointDistance)(float64, Point, Point)
	UpdateDiameter(PointDistance)()
	GetDensity(PointDistance)(float64)
	UpdateDensity(PointDistance)
	GetRadius(PointDistance)(float64, Point)
	UpdateRadius(PointDistance)
	GetClusterProperties(PointDistance)(map[string]interface{})
	HasMember(Point)(bool)
}

type DistanceMeasure func(a,b Cluster,distAlgo PointDistance)(float64)

// The minimum distance between any two points from each cluster.
func SingleLink(a,b Cluster,distAlgo PointDistance)(float64){
	min := math.MaxFloat64
	for _,aItem := range a.GetClusterItems(){
		for _,bItem := range b.GetClusterItems(){
			tmp := distAlgo(aItem,bItem)
			if tmp < min {
				min = tmp
			}
		}
	}
	return min
}
// The maximum distance between any two points from each cluster.
func CompleteLink(a,b Cluster,distAlgo PointDistance)(float64){
	max := float64(-1)
	for _,aItem := range a.GetClusterItems(){
		for _,bItem := range b.GetClusterItems(){
			tmp := distAlgo(aItem,bItem)
			if tmp > max {
				max = tmp
			}
		}
	}
	return max
}
// The average distance of all pairs of points, one from each cluster
func MeanDistance(a,b Cluster,distAlgo PointDistance)(distance float64){
	if a == nil || b == nil {
		distance = math.MaxFloat64
		return
	}
	n:=0
	for _,p1 := range a.GetClusterItems() {
		for _,p2 := range b.GetClusterItems() {
			distance += distAlgo(p1,p2)
			n++
		}
	}
	distance /= float64(n)
	return
}
func CentroidDistance(a,b Cluster,distAlgo PointDistance)(distance float64){
	if a == nil || b == nil {
		distance = math.MaxFloat64
		return
	}
	distance = distAlgo(a.GetCentroid(),b.GetCentroid())
	return
}
// The distance between the centroids combined with distance between cluster size
func CenterVolumeDistance(a,b Cluster,distAlgo PointDistance)(distance float64){
	exponent := float64(2.0)
	if a == nil || b == nil {
		distance = math.MaxFloat64
		if a != nil {
			// take distance from a to origin
		} else if b != nil {
			// take distance form b to origin
		}
		return
	}

	// get distance between centroids
	centComponent := distAlgo(a.GetCentroid(),b.GetCentroid(),exponent)

	// get distance between cluster size
	repA:=NewGenericPoint(1)
	repB:=NewGenericPoint(1)
	repA.SetCoordinate(0,a.GetClusterSize())
	repB.SetCoordinate(0,b.GetClusterSize())
	distComponent := distAlgo(repA,repB,exponent)

	distance = math.Sqrt(centComponent+distComponent)
	return
}
func VolumeDistance(a,b Cluster,distAlgo PointDistance)(distance float64){
	exponent := float64(2.0)
	if a == nil || b == nil {
		distance = math.MaxFloat64
		if a != nil {
			// take distance from a to origin
		} else if b != nil {
			// take distance form b to origin
		}
		return
	}

	// get distance between cluster size
	repA:=NewGenericPoint(1)
	repB:=NewGenericPoint(1)
	repA.SetCoordinate(0,a.GetClusterSize())
	repB.SetCoordinate(0,b.GetClusterSize())
	distComponent := distAlgo(repA,repB,exponent)

	distance = math.Sqrt(distComponent)
	return
}

// @testing: successfully
func CutOutPointerIndex(cluster *cluster, cutIndex int)(err error){
	if cluster != nil && cutIndex < cluster.GetClusterSize() && cutIndex >= 0 {
		switch cutIndex {
		case 0:
			*cluster = (*cluster)[cutIndex + 1:]
		case len(*cluster) - 1:
			*cluster = (*cluster)[:cutIndex]
		default:
			*cluster = append((*cluster)[:cutIndex], (*cluster)[cutIndex + 1:]...)
		}
	} else {
		err = errors.New(auxiliary.CUTOUTERROR)
	}
	return
}

type cluster []Point

func NewCluster()(Cluster){
	return new(cluster)
}

// Calculates the distance between two clusters iff other parameter is a pointer to an object of type cluster.
func (c *cluster) DistanceTo(other Cluster,clusterDistanceAlgo DistanceMeasure,pointDistanceAlgo PointDistance)(distance float64,err error){
	// check for nil pointers
	if other == nil {
		err = errors.New("pointer to other cluster is nil")
		return
	}

	// check for type compatibility
	if reflect.TypeOf(c) == reflect.TypeOf(other) {
		distance = clusterDistanceAlgo(c,other,pointDistanceAlgo)
	} else {
		distance = clusterDistanceAlgo(c,other,pointDistanceAlgo)
		fmt.Printf("obj type:%T\tparameter type:%T\n",c,other)
		//err = errors.New("pointer to other cluster is not of same type as pointer to this cluster obj")
	}
	return
}

func (c *cluster) GetClusterItems()([]Point) {return *c}

func (c *cluster) GetClusterSize()(int) {return len(*c)}

func (c *cluster) CombineWithCluster(other Cluster,pointDistAlgo PointDistance)(newCluster Cluster) {
	newCluster = NewCluster()
	for _,p := range c.GetClusterItems() {
		newCluster.AddItem(p)
	}
	if other != nil {
		for _,p := range other.GetClusterItems() {
			newCluster.AddItem(p)
		}
	}
	newCluster.UpdateCentroid()
	newCluster.UpdateAveragePairOfPoints(pointDistAlgo)
	newCluster.UpdateDiameter(pointDistAlgo)
	newCluster.UpdateRadius(pointDistAlgo)
	newCluster.UpdateDensity(pointDistAlgo)
	return
}

func (c *cluster) AddItem(p Point)() {
	if p != nil {
		*c = append(*c,p)
	}
}

func (c *cluster) DeleteItem(pointer Point)() {
	if pointer != nil {
		index := -1
		for i,itemPointer := range c.GetClusterItems() {
			if itemPointer==pointer {
				index = i
				break
			}
		}
		CutOutPointerIndex(c,index)
	}
}

// Average of the points in each dimension
func (c *cluster) GetCentroid()(centroid Point) {
	if c.GetClusterSize() > 0 {
		// take the first item and generate a new point according to this rep
		rep := c.GetClusterItems()[0]
		centroid = NewGenericPoint(rep.Dimensions())

		// sum up coordinates of all points in each dimension
		for i:=0;i<centroid.Dimensions();i++{
			for _,p := range c.GetClusterItems() {
				if centroid.GetVector()[i] == nil {
					centroid.SetCoordinate(i,p.GetCoordinate(i).GetValue())
				} else {
					centroid.GetVector()[i].AddValue(p.GetCoordinate(i).GetValue())
				}
			}
		}

		// normalize vecotr
		centroid.NormalizeVector(float64(c.GetClusterSize()))
	}
	return
}
func (c *cluster) UpdateCentroid()() {}
func (c *cluster) GetAveragePairOfPoints(pointDistAlgo PointDistance)(distance float64) {
	pairNumber := (c.GetClusterSize() * (c.GetClusterSize() - 1)) / 2
	if pairNumber > 0 {
		var totalDistance float64
		for i,p := range c.GetClusterItems() {
			for j:=i+1;j<len(c.GetClusterItems());j++ {
				totalDistance += pointDistAlgo(p,(c.GetClusterItems())[j])
			}
		}
		distance = totalDistance / float64(pairNumber)
	}
	return
}
func (c *cluster) UpdateAveragePairOfPoints(pointDistAlgo PointDistance)() {}
func (c *cluster) GetDiameter(pointDistAlgo PointDistance)(diam float64, min,max Point) {
	for i,p := range c.GetClusterItems() {
		if min == nil {
			min = p
		}
		if max == nil {
			max = p
		}
		for j:=i+1;j<len(c.GetClusterItems());j++ {
			if tmp:=pointDistAlgo(p,c.GetClusterItems()[j]);tmp>diam{
				diam = tmp
				min = p
				max = c.GetClusterItems()[j]
			}
		}
	}
	return
}
func (c *cluster) UpdateDiameter(pointDistAlgo PointDistance)() {}
func (c *cluster) GetDensity(pointDistAlgo PointDistance)(dens float64) {
	diam,_,_ := c.GetDiameter(pointDistAlgo)
	dens = float64(c.GetClusterSize()) / diam
	return
}
func (c *cluster) UpdateDensity(pointDistAlgo PointDistance) {}
func (c *cluster) GetRadius(pointDistAlgo PointDistance)(rad float64, satellite Point) {
	centroid := c.GetCentroid()
	for _,p := range c.GetClusterItems() {
		if tmp:=pointDistAlgo(p,centroid); tmp > rad {
			rad = tmp
			satellite = p
		}
	}
	return
}
func (c *cluster) UpdateRadius(pointDistAlgo PointDistance) {}
func (c *cluster) GetClusterProperties(pointDistAlgo PointDistance)(dict map[string]interface{}) {
	dict = make(map[string]interface{})
	cent := c.GetCentroid()
	diam,min,max := c.GetDiameter(pointDistAlgo)
	rad,sat := c.GetRadius(pointDistAlgo)
	app := c.GetAveragePairOfPoints(pointDistAlgo)
	dens := c.GetDensity(pointDistAlgo)
	size := c.GetClusterSize()
	dict[DENS_KEY] = dens
	dict[DIAM_KEY] = diam
	dict[RAD_KEY] = rad
	dict[APP_KEY] = app
	dict[CENT_KEY] = cent
	dict[MIN_KEY] = min
	dict[MAX_KEY] = max
	dict[SAT_KEY] = sat
	dict[SIZE_KEY] = size
	return
}
func (c *cluster) HasMember(point Point)(contains bool){
	contains = false
	for _,pointPointer := range c.GetClusterItems() {
		if pointPointer == point {
			contains = true
			break
		}
	}
	return
}
func (c *cluster) String()(string){
	return fmt.Sprintf(
		"size: %d - centroid: %s\n",
		c.GetClusterSize(),
		c.GetCentroid(),
	)
}
