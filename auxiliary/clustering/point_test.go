package clustering

import(
	"testing"
	"fmt"
)

type pointTestCase struct {
	vector []interface{}
}

var pointGenerationTest = []pointTestCase{
	pointTestCase{vector:[]interface{}{3.1,2.2,1.2}},
	pointTestCase{vector:[]interface{}{3.1,2.2,1.2}},
	pointTestCase{vector:[]interface{}{3,2.2,1.2}},
	pointTestCase{vector:[]interface{}{3.0,2.2,1.2}},
}

var values = []float64{
	1.0, 0, 3.0, 3.2, 1,0, 3, 0.00, 3.2000, 1,
}

func CoordinateGeneration(t *testing.T){
	var coordinates = make([]*floatCoordinate,len(values))
	for i,v := range values {
		tmp := NewCoordinate(v)
		coordinates[i] = tmp.(*floatCoordinate)
	}
	fmt.Println(coordinates)
	for i,c1 := range coordinates {
		for j:=i+1;j<len(coordinates);j++ {
			fmt.Printf("%+v %+v %v\n",c1.GetValue(),coordinates[j].GetValue(),*c1==*coordinates[j])
		}
	}
}

func estPointGeneration(t *testing.T){
	fmt.Println("Performing TestPointDistance() method")
	var last *GenericPoint
	for _,i := range pointGenerationTest {
		p := NewGenericPoint(len(i.vector))
		fmt.Println(p)
		for dim,val := range i.vector {
			p.SetCoordinate(dim,val)
		}
		fmt.Println(p)
		for dim,val := range i.vector {
			switch v:=val.(type){
			case float64:
				p.SetCoordinate(dim,float64(3)*float64(v))
			case int:
				p.SetCoordinate(dim,float64(3)*float64(v))
			}
		}
		fmt.Println(p)
		p.NormalizeVector(3)
		fmt.Println(p)
		fmt.Println(p.getOrigin())
		fmt.Println()
		if last != nil {
			fmt.Println(p.Equals(last))
		}
		last = p
	}
}
