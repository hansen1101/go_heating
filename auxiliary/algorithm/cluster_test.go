package algorithm

import(
	"testing"
	"fmt"
	"github.com/hansen1101/go_heating/auxiliary/clustering"
)

type clusterTestCase struct {
	points []clustering.Point
}

type kMeansAssignmentTest struct {
	points []clustering.Point
	centroids []clustering.Point
}

var clusterGenerationTest = []clusterTestCase{
	clusterTestCase{
		points:[]clustering.Point{
			&clustering.GenericPoint{
				clustering.NewFloat(4.13),
				clustering.NewFloat(4.2),
				clustering.NewFloat(4.85),
			},
			&clustering.GenericPoint{
				clustering.NewFloat(-4.5),
				clustering.NewFloat(-4.2),
				clustering.NewFloat(-5.17),
			},
			&clustering.GenericPoint{
				clustering.NewFloat(6),
				clustering.NewFloat(2.0),
				clustering.NewFloat(-3.5),
			},
			&clustering.GenericPoint{
				clustering.NewFloat(8),
				clustering.NewFloat(1.0),
				clustering.NewFloat(-5),
			},
			&clustering.GenericPoint{
				clustering.NewFloat(3),
				clustering.NewFloat(0.2),
				clustering.NewFloat(-1),
			},
			&clustering.GenericPoint{
				clustering.NewFloat(-0.2),
				clustering.NewFloat(1.0),
				clustering.NewFloat(1.0),
			},
		},
	},
	clusterTestCase{
		points:[]clustering.Point{
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
			&clustering.GenericPoint{clustering.NewFloat(6.3)},
			&clustering.GenericPoint{clustering.NewFloat(12.5)},
			&clustering.GenericPoint{clustering.NewFloat(10)},
			&clustering.GenericPoint{clustering.NewFloat(8)},
			&clustering.GenericPoint{clustering.NewFloat(7)},
			&clustering.GenericPoint{clustering.NewFloat(6)},
			&clustering.GenericPoint{clustering.NewFloat(5.5)},
			&clustering.GenericPoint{clustering.NewFloat(4)},
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
			&clustering.GenericPoint{clustering.NewFloat(0)},
		},
	},
}

var clusterAssignmentTest = []kMeansAssignmentTest{
	kMeansAssignmentTest{
		points : []clustering.Point{
			&clustering.GenericPoint{clustering.NewFloat(0.1)},
			&clustering.GenericPoint{clustering.NewFloat(0.2)},
			&clustering.GenericPoint{clustering.NewFloat(0.3)},
			&clustering.GenericPoint{clustering.NewFloat(0.4)},
			&clustering.GenericPoint{clustering.NewFloat(0.5)},
			&clustering.GenericPoint{clustering.NewFloat(0.6)},
			&clustering.GenericPoint{clustering.NewFloat(0.7)},
			&clustering.GenericPoint{clustering.NewFloat(0.7)},
			&clustering.GenericPoint{clustering.NewFloat(1.2)},
		},
		centroids : []clustering.Point{
			&clustering.GenericPoint{clustering.NewFloat(-1.5)},
			&clustering.GenericPoint{clustering.NewFloat(-1)},
			&clustering.GenericPoint{clustering.NewFloat(0)},
		},
	},
}

func estKMeansClustering(t *testing.T) {
	fmt.Println("Performing TestClusterGeneration() method")
	for _,testcase := range clusterGenerationTest {
		clusters,err := KMeans(
			4,
			0.01,
			testcase.points,
			clustering.MeanDistance,
			clustering.WeightedEuclideanDistance,
		)
		if err != nil {
			fmt.Println(err)
		}
		for i,c := range clusters {
			fmt.Println(i)
			fmt.Println(c)
		}
	}
}
func estClusterGeneration(t *testing.T) {
	fmt.Println("Performing TestClusterGeneration() method")
	c := clustering.NewCluster()
	for _,testcase := range clusterGenerationTest {
		for _,p := range testcase.points {
			c.AddItem(p)
		}
		clusters,err := KMeans(4,0.01,testcase.points,clustering.SingleLink,clustering.EuclideanDistance)
		test := AgglomerativeClustering(testcase.points,clustering.SingleLink,clustering.EuclideanDistance)
		printClustering(*test,clustering.EuclideanDistance)
		if err != nil {
			fmt.Println(err)
		}
		for i,c := range clusters {
			fmt.Println(i)
			fmt.Println(c)
			//fmt.Println(c.GetClusterProperties(clustering.EuclideanDistance))
		}
	}
}

func TestKMeansAssignment(t *testing.T) {
	for _,assignmentTest := range clusterAssignmentTest {
		c := clusterAssignment(assignmentTest.points,assignmentTest.centroids,clustering.EuclideanDistance)
		fmt.Print(c)
	}
}
