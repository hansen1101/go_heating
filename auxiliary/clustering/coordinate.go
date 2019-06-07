package clustering

import (
	"errors"
	"math"
	"reflect"
	"log"
	"fmt"
)

type Coordinate interface {
	GetValue()interface{}
	SetValue(interface{})(error)
	PoweredDistanceTo(Coordinate,float64)(float64)
	GetAbsoluteDistanceTo(Coordinate)(float64)
	AddValue(interface{})()
	NormalizeValue(float64)()
	GetZeroValue()(interface{})
}

func NewCoordinate(value interface{})(c Coordinate){
	switch v:=value.(type){
	case float64:
		c = NewFloat(v)
	case int:
		c = NewFloat(float64(v))
	}
	return
}

type floatCoordinate float64

func NewFloat(value float64)(coordinate *floatCoordinate){
	coordinate = new(floatCoordinate)
	*coordinate = floatCoordinate(value)
	return
}

func (c *floatCoordinate) GetValue()(interface{}){
	return float64(*c)
}
func (c *floatCoordinate) SetValue(value interface{})(err error){
	if value == nil {
		*c = floatCoordinate(0)
	} else if v,ok := value.(float64); ok {
		*c = floatCoordinate(v)
	} else if v,ok := value.(floatCoordinate); ok {
		*c = v
	} else if v,ok := value.(*floatCoordinate); ok {
		*c = *v
	} else {
		err = errors.New("Wrong type used; Value must be a float64 or nil.")
	}
	return
}
func (c *floatCoordinate) PoweredDistanceTo(other Coordinate, exponend float64)(distance float64) {
	if c.ValueOfEqualType(other) {
		// we can calculate the dinstace between two points of equal value type
		if other == nil {
			distance = math.MaxFloat64
		} else {
			distance = math.Pow(float64(*c) - other.GetValue().(float64),exponend)
		}
	} else {
		log.Fatal("Coordinates of different type cannot be distance compared...")
	}
	return
}
func (c *floatCoordinate) GetAbsoluteDistanceTo(other Coordinate)(distance float64){
	if c.ValueOfEqualType(other) {
		// we can calculate the dinstace between two points of equal value type
		if other == nil {
			distance = math.MaxFloat64
		} else {
			distance = math.Abs(float64(*c) - other.GetValue().(float64))
		}
	} else {
		log.Fatal("Coordinates of different type cannot be distance compared...")
	}
	return
}
func (c *floatCoordinate) AddValue(value interface{})() {
	if v, ok := value.(float64); ok {
		*c += floatCoordinate(v)
	}
}
func (c *floatCoordinate) NormalizeValue(denominator float64)() {
	if denominator != float64(0) {
		*c /= floatCoordinate(denominator)
	}
}
func (c *floatCoordinate) GetZeroValue()(interface{}){return float64(0)}

func (c *floatCoordinate) ValueOfEqualType(other Coordinate)(equality bool){
	if other != nil {
		if reflect.TypeOf(c.GetValue()) == reflect.TypeOf(other.GetValue()) {
			// types of value fields are equal
			equality = true
		} else {
			switch other.GetValue().(type) {
			case int:
				equality = true
			case uint:
				equality = true
			default:
				equality = false
			}
		}
	}
	return
}
func (c *floatCoordinate) String()(string){
	return fmt.Sprintf("%.2f",*c)
}