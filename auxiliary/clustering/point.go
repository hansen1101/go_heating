package clustering

import (
	"math"
	"fmt"
	"log"
	"reflect"
)

type Point interface {
	PoweredDistanceTo(Point,float64)(float64)
	EuclideanDistanceTo(Point)(float64)
	WeightedEuclideanDistanceTo(Point)(float64)
	Dimensions()(int)
	SetCoordinate(int,interface{})()
	GetCoordinate(int)(Coordinate)
	GetVector()([]Coordinate)
	Equals(Point)(bool)
	NormalizeVector(float64)()
}

type PointDistance func (Point,Point,...interface{})(float64)
func WeightedEuclideanDistance(p1,p2 Point,data...interface{})(distance float64){
	if p1 != nil {
		distance = p1.WeightedEuclideanDistanceTo(p2)
	}
	return
}
func EuclideanDistance(p1,p2 Point,data...interface{})(distance float64){
	if p1 != nil {
		distance = p1.EuclideanDistanceTo(p2)
	}
	return
}
func PoweredDistance(p1,p2 Point,data...interface{})(distance float64){
	var exp float64
	for _,e := range data {
		if v,ok := e.(float64); ok {
			exp = v
		}
	}
	if p1 != nil {
		distance = p1.PoweredDistanceTo(p2,exp)
	}
	return
}

func GetDiam(points []Point,distAlgo PointDistance)(diam float64, min, max Point){
	for i,p1 := range points {
		for j:=i+1;j<len(points);j++{
			if tmp:=distAlgo(p1,points[j]);tmp>diam{
				diam = tmp
				min = p1
				max = points[j]
			}
		}
	}
	return
}

type GenericPoint []Coordinate

func NewGenericPoint(dimensions int)(point *GenericPoint){
	point = new(GenericPoint)
	for i:=0;i<dimensions;i++{
		// append nil coordinate for each dimension
		*point = append(*point,nil)
	}
	return
}

func (p *GenericPoint) PoweredDistanceTo(other Point, exponend float64)(distance float64) {
	if other == nil {
		distance = math.MaxFloat64
	} else {
		if p.Dimensions() == other.Dimensions() {
			for dimension,coordinate := range *p {
				distance += coordinate.PoweredDistanceTo(other.GetVector()[dimension],exponend)
			}
		} else {
			log.Fatal("Points differ in dimensions. We have a vector space violation...")
		}
	}
	return
}

//@testing successful
// distance measure obeys properties of symmetry, triangle inequality, monotony
func (p *GenericPoint) EuclideanDistanceTo(other Point)(distance float64){
	if other == nil {
		distance = math.MaxFloat64
	} else {
		if delta := p.PoweredDistanceTo(other,2); p != other && delta > float64(0){
			distance = math.Sqrt(delta)
		}
	}
	return
}

//@testing successful
// distance measure obeys properties of symmetry, triangle inequality, monotony
func (p *GenericPoint) WeightedEuclideanDistanceTo(other Point)(distance float64){
	if other == nil {
		distance = math.MaxFloat64
		return
	}
	if reflect.TypeOf(p) == reflect.TypeOf(other) {
		max := getPairwiseMaxDistanceToZero(p,other.(*GenericPoint))
		distance = math.Pow(p.EuclideanDistanceTo(other.(*GenericPoint)),2)

		//@todo check if delta is computed in such a way that far clusters have little distance to each other
		delta := math.Pow(max.EuclideanDistanceTo(max.getOrigin()),2)
		if denumerator := math.Abs(delta); denumerator != float64(0){
			distance /= denumerator
		}
	}
	return
}
func (p *GenericPoint) Dimensions()(int){
	return len(*p)
}
func (p *GenericPoint) SetCoordinate(dimension int, value interface{}){
	if dimension < p.Dimensions() {
		if (*p)[dimension] == nil {
			(*p)[dimension] = NewCoordinate(value)
		} else {
			err := (*p)[dimension].SetValue(value)
			if err != nil {
				// type of value did not fit coordinate's type
				fmt.Println(err.Error())
			}
		}
	}
}
func (p *GenericPoint) GetCoordinate(dimension int)(c Coordinate){
	if dimension < len(*p) && dimension >= 0 {
		c = (*p)[dimension]
	}
	return
}
func (p *GenericPoint) GetVector()([]Coordinate){
	return *p
}
func (p *GenericPoint) Equals(other Point)(equal bool){
	if p.Dimensions() != other.Dimensions(){
		return
	}
	for i,coordinate := range p.GetVector() {
		if coordinate.GetValue() != other.GetVector()[i].GetValue() {
			return
		}
	}
	equal = true
	return
}
func (p *GenericPoint) NormalizeVector(denominator float64)(){
	for _,coordPointer := range *p {
		coordPointer.NormalizeValue(denominator )
	}
}

func (p *GenericPoint) String()(string){
	s := "("
	for dim,val := range *p {
		if val != nil {
			s+=fmt.Sprintf("%s",val)
		} else {
			s+=fmt.Sprintf("%T",val)
		}
		if dim < len(*p)-1 {
			s+=fmt.Sprint(" ; ")
		}
	}
	s += ")"
	return s
}
func (p *GenericPoint) getOrigin()(origin *GenericPoint){
	origin = NewGenericPoint(p.Dimensions())
	for dimension,coordinate := range *p {
		origin.SetCoordinate(dimension,coordinate.GetZeroValue())
	}
	return
}

//@testing successful
func getPairwiseMaxDistanceToZero (p1,p2 *GenericPoint)(max *GenericPoint){
	if p1 != nil && p2 != nil {
		p1Delta := p1.EuclideanDistanceTo(p1.getOrigin())
		p2Delta := p2.EuclideanDistanceTo(p2.getOrigin())
		if max = p1; math.Abs(p2Delta) > math.Abs(p1Delta) {
			max = p2
		}
	} else if p1 != nil {
		max = p1
	} else {
		max = p2
	}
	return
}