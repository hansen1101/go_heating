package algorithm

import (
	"math"
	"log"
	"errors"
	"github.com/hansen1101/go_heating/auxiliary"
	"github.com/hansen1101/go_heating/auxiliary/clustering"
	"fmt"
)

// Generic implementation of munkres assignment algorithm.
func Munkres(matrix [][]int)(*[]int){
	// find a matching between clusters of the two buckets
	var markedRows, markedCols, unmarkedRows ,unmarkedCols, assigned, primed, z []int
	var solutionFound bool // false by default
	var step, smallestValueFound int // 0 by default

	for !solutionFound {
		switch step {
		case 0:
			// init distance matrix, assigned and primed sets
			step = munkresAssignmentInit(&matrix, &assigned, &primed, &markedRows, &markedCols, &unmarkedRows, &unmarkedCols)
		case 1:
			// reduce the matrix by rows and columns
			step = munkresRowColReduction(&matrix)
		case 2:
			// generate initial assignment
			step = munkresInitialAssignment(&matrix, &assigned)
		case 3:
			// test for valid assignment
			solutionFound,step = munkresAssignmentTest(&matrix,&assigned,&markedCols,&unmarkedCols)
		case 4:
			// search for an uncovered zero and prime it
			smallestValueFound,step = munkresSearchAndPrimeUncoveredZero(&matrix,&markedCols,&markedRows,&unmarkedCols,&unmarkedRows,&assigned,&primed,&z)
		case 5:
			// augment path such that primes become assigned
			step = munkresAugmentPath(&markedCols,&markedRows,&unmarkedCols,&unmarkedRows,&assigned,&primed,&z)
		case 6:
			// add smallest value to marked rows
			step = munkresUpdateMatrix(&matrix,&markedRows,&unmarkedCols,smallestValueFound)
		}
	}
	return &assigned
}

// @testing: only one case
func munkresAssignmentInit(matrix *[][]int, assigned, primed, markedRows, markedCols, unmarkedRows ,unmarkedCols *[]int)(step int){
	// init assigned
	*assigned = make([]int,len(*matrix),len(*matrix))
	for i,_ := range(*assigned){
		(*assigned)[i] = -1
	}

	// init primed
	*primed = make([]int,len(*matrix),len(*matrix))
	for i,_ := range(*primed){
		(*primed)[i] = -1
	}

	// init marked/unmarked rows and cols sets
	*markedCols = make([]int,0,len(*matrix))
	*markedRows = make([]int,0,len(*matrix))
	*unmarkedCols = make([]int,0,len(*matrix))
	*unmarkedRows = make([]int,0,len(*matrix))
	for i:=0;i<len(*matrix);i++{
		*unmarkedCols = append(*unmarkedCols,i)
		*unmarkedRows = append(*unmarkedRows,i)
	}

	step = 1
	return
}

// @testing: only one case
func munkresRowColReduction(matrix *[][]int)(step int){
	// row reduction
	for _,row := range(*matrix){
		min := math.MaxInt32
		for _,col := range(row) {
			if col < min {
				min = col
			}
		}
		for j,_ := range(row) {
			row[j] -= min
		}

	}

	// col reduction
	for i:=0;i<len(*matrix);i++{
		min := math.MaxInt32
		for j:=0;j<len(*matrix);j++{
			if (*matrix)[j][i] < min {
				min = (*matrix)[j][i]
			}
		}
		for j:=0;j<len(*matrix);j++{
			(*matrix)[j][i] -= min
		}
	}
	step = 2
	return
}

// @testing: only one case
func munkresInitialAssignment(matrix *[][]int, assigned *[]int)(step int){
	// generate initial assignment
	for i:=0;i<len(*matrix);i++{
		for j:=0;j<len(*matrix);j++{
			if (*matrix)[i][j]==0{
				// check for assignments in row and column
				assignmentPossible := true
				for row,col := range *assigned {
					if row == i {
						if col >= 0 {
							// assigment in row
							assignmentPossible = false
						}
					}
					if col == j{
						// assignment in col
						assignmentPossible = false
					}
				}
				if assignmentPossible {
					(*assigned)[i] = j
				}
			}
		}
	}
	step = 3
	return
}

// @testing: only one case
func munkresAssignmentTest(matrix *[][]int, assigned, markedCols, unmarkedCols *[]int)(solutionFound bool,step int){
	// cover columns
	for _,assignedCol := range *assigned {
		if assignedCol >= 0 {
			// append assignedCol to markedCol set and cut out from unmarkedCol set
			markPoint(markedCols,unmarkedCols,assignedCol)
		}
	}
	// test for solution
	if len(*markedCols) == len(*matrix) {
		solutionFound = true
	}
	step = 4
	return
}

// @testing: only one case
func munkresSearchAndPrimeUncoveredZero(matrix *[][]int, markedCols,markedRows,unmarkedCols,unmarkedRows,assigned,primed,z *[]int)(smallestValueFound, step int){
	smallestValueFound = math.MaxInt32

	// search for an uncovered zero and prime it
	step4:
	for _,row := range *unmarkedRows {
		for _,col := range *unmarkedCols {
			if (*matrix)[row][col] == 0 {
				// prime it
				if (*primed)[row] >= 0 {
					/*
					//@debug
					fmt.Println(*matrix)
					fmt.Println(*primed)
					fmt.Println(row)
					fmt.Println(col)
					fmt.Println(*assigned)
					fmt.Println(*markedRows)
					fmt.Println(*markedCols)
					fmt.Println("ur ",*unmarkedRows)
					fmt.Println("uc ",*unmarkedCols)
					fmt.Println(*z)
					//@debug_end
					*/
					log.Fatal("We have more than ONE primed zero in this row.")
				}
				(*primed)[row] = col

				// check for for assigned zero in same row
				if (*assigned)[row] < 0 {
					// append row index of primed zero to z
					*z = make([]int,0)
					*z = append(*z,row)
					step = 5
				} else {
					// uncover col, cover row
					markPoint(unmarkedCols,markedCols,(*assigned)[row])
					markPoint(markedRows,unmarkedRows,row)
					step = 4
				}
				break step4
			} else {
				if smallestValueFound > (*matrix)[row][col] {
					smallestValueFound = (*matrix)[row][col]
				}
				step = 6
			}
		}
	}
	return
}

// @testing: only one case
func munkresAugmentPath(markedCols,markedRows,unmarkedCols,unmarkedRows,assigned,primed,z  *[]int)(step int){
	// search for assignment in column of uncovered primed zero z0
	for starredRow,starredCol := range *assigned {
		if starredCol == (*primed)[(*z)[len(*z)-1]] {
			// append assigned zero z1 to series
			*z = append(*z,starredRow)
			break
		}
	}

	// aufment path
	for i,rowIndex := range *z {
		// check if assignment is following the prime
		if i+1 <= len(*z)-1{
			(*assigned)[(*z)[i+1]]=-1
		}
		(*assigned)[rowIndex]=(*primed)[rowIndex]
	}

	//erase primes and marked lines
	for i,_ := range *primed {
		(*primed)[i] = -1
	}
	for _,row := range *markedRows {
		markPoint(unmarkedRows,markedRows,row)
	}
	for _,col := range *markedCols {
		markPoint(unmarkedCols,markedCols,col)
	}
	step = 3
	return
}

// @testing: only one case
func munkresUpdateMatrix(matrix *[][]int, markedRows, unmarkedCols *[]int, smallestValueFound int)(step int){
	// add smallest value to marked rows
	for _,row := range *markedRows {
		for i:=0;i<len(*matrix);i++{
			(*matrix)[row][i] += smallestValueFound
		}
	}

	// substract smallest value from unmarked columns
	for i:=0;i<len(*matrix);i++{
		for _,col := range *unmarkedCols {
			(*matrix)[i][col] -= smallestValueFound
		}
	}
	step = 4
	return
}

// @testing: successfully
func markPoint(appendSet *[]int, cutoutSet *[]int, value int){
	if appendSet == nil || cutoutSet == nil {
		return
	}
	for i,unmarkedIndex := range *cutoutSet {
		if unmarkedIndex == value {
			cutOutIndex(cutoutSet,i)
			*appendSet = append(*appendSet,value)
			break
		}
	}
}

// @testing: successfully
func cutOutIndex(data *[]int, cutIndex int)(err error){
	if data != nil && cutIndex < len(*data) && cutIndex >= 0 {
		switch cutIndex {
		case 0:
			*data = (*data)[cutIndex + 1:]
		case len(*data) - 1:
			*data = (*data)[:cutIndex]
		default:
			*data = append((*data)[:cutIndex], (*data)[cutIndex + 1:]...)
		}
	} else {
		err = errors.New(auxiliary.CUTOUTERROR)
	}
	return
}

func printClustering(clusterLevel [][]clustering.Cluster, pointDistAlgo clustering.PointDistance)(){
	for i,clustering := range clusterLevel {
		fmt.Printf("\n%d. Level\n",i)
		for _,cluster := range clustering {
			diam,_,_ := cluster.GetDiameter(pointDistAlgo)
			rad,_ := cluster.GetRadius(pointDistAlgo)
			fmt.Printf("Diam:%.2f\t%v\t%+v\tRad:%.2f\tAvg:%.2f\tDens: %.2f\n",
				diam,
				cluster.GetCentroid(),
				cluster.GetClusterItems(),
				rad,
				cluster.GetAveragePairOfPoints(pointDistAlgo),
				cluster.GetDensity(pointDistAlgo))
		}
	}
}
