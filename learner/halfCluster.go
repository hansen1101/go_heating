package learner

import (
	"fmt"
	"github.com/hansen1101/go_heating/system/logger"
	"github.com/hansen1101/go_heating/auxiliary/clustering"
	"encoding/json"
)

// Implements logger.Logable interface
type halfCluster struct {
	//logger.Logable
	*simpleCluster //anonymous pointer to simpleCluster
	seed int64
	name string
}

// Implementation of logger.Logable interface
func (cluster *halfCluster) GetRelationName()(string){return cluster.name}
func (cluster *halfCluster) CreateRelation()(){
	var query_string string

	query_string = fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s(" +
			"id INT NOT NULL AUTO_INCREMENT," +
			"time INT NOT NULL DEFAULT 0," +
			"centroid VARCHAR(255) NULL DEFAULT ''," +
			"min VARCHAR(255) NULL DEFAULT ''," +
			"max VARCHAR(255) NULL DEFAULT ''," +
			"size INT UNSIGNED NULL DEFAULT 0," +
			"diameter FLOAT(7,3) NULL DEFAULT 0.0," +
			"radius FLOAT(7,3) NULL DEFAULT 0.0," +
			"density FLOAT(7,3) NULL DEFAULT 0.0," +
			"averageDistance FLOAT(7,3) NULL DEFAULT 0.0," +
			"category INT UNSIGNED NULL DEFAULT 0," +
			"type INT UNSIGNED NULL DEFAULT 0," +
			"recordTS INT UNSIGNED NULL DEFAULT 0," +
			"recordSize INT UNSIGNED NULL DEFAULT 0," +
			"PRIMARY KEY(id)," +
			"UNIQUE cluster_fingerprint (time,centroid,min,max,size)" +
			")ENGINE=InnoDB DEFAULT CHARSET=latin1",
		cluster.GetRelationName())

	logger.StatementExecute(query_string)
}
func (cluster *halfCluster) Insert(data...interface{})() {
	var query_string string
	var recordTS int
	var recordSize int
	//var pointDistAlgo clustering.PointDistance
	for _,d := range data {
		if v,ok := d.(map[string]interface{}); ok {
			if val,test := v["recordtimestamp"];test {
				recordTS = int(val.(int64))
			}
			if val,test := v["recordsize"]; test {
				recordSize = val.(int)
			}
		}
		/*
		if v,ok := d.(func (clustering.Point,clustering.Point,...interface{})(float64)); ok {
			pointDistAlgo = v
		}
		*/
	}

	var diam,rad,dens,appd float64
	var min,max clustering.Point
	diam,min,max = cluster.GetDiameter()
	rad,_ = cluster.GetRadius()
	dens = cluster.density
	appd = cluster.averagePairOfPoints

	var err error
	var centroidJsonBytes,minJsonBytes,maxJsonBytes []byte
	centroidJsonBytes,err = json.Marshal(cluster.GetCentroid().GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}
	minJsonBytes,err = json.Marshal(min.GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}
	maxJsonBytes,err = json.Marshal(max.GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println(cluster)
	query_string = fmt.Sprintf(
		"INSERT IGNORE INTO %s" +
			"(time,centroid,min,max,size,diameter,radius,density,averageDistance,recordTS,recordSize)" +
			" VALUES " +
			"(%d,%s,%s,%s,%d,%.3f,%.3f,%.3f,%.3f,%d,%d)",
		cluster.GetRelationName(),
		cluster.GetCentroid().(*deltaPoint).timestamp,
		string(centroidJsonBytes),
		string(minJsonBytes),
		string(maxJsonBytes),
		cluster.GetClusterSize(),
		diam,
		rad,
		dens,
		appd,
		recordTS,
		recordSize,
	)

	go logger.StatementExecute(query_string)
}
func (cluster *halfCluster) Delete(data...interface{})(){
	var query_string string

	/*
	var pointDistAlgo clustering.PointDistance
	for _,d := range data {
		if v,ok := d.(func (clustering.Point,clustering.Point,...interface{})(float64)); ok {
			pointDistAlgo = v
		}
	}
	*/

	var err error
	var centroidJsonBytes,minJsonBytes,maxJsonBytes []byte
	centroidJsonBytes,err = json.Marshal(cluster.GetCentroid().GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}
	minJsonBytes,err = json.Marshal(cluster.min.GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}
	maxJsonBytes,err = json.Marshal(cluster.max.GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}

	query_string = fmt.Sprintf(
		"DELETE FROM %s" +
			"WHERE " +
			"time=%d AND " +
			"centroid=%s AND " +
			"min=%s AND " +
			"max=%s AND " +
			"size=%d",
		cluster.GetRelationName(),
		cluster.GetCentroid().(*deltaPoint).timestamp,
		string(centroidJsonBytes),
		string(minJsonBytes),
		string(maxJsonBytes),
		cluster.GetClusterSize(),
	)

	logger.StatementExecute(query_string)
}
func (cluster *halfCluster) Update(data...interface{})(){
	var query_string string

	/*
	var pointDistAlgo clustering.PointDistance
	for _,d := range data {
		if v,ok := d.(func (clustering.Point,clustering.Point,...interface{})(float64)); ok {
			pointDistAlgo = v
		}
	}
	*/

	var err error
	var centroidJsonBytes,minJsonBytes,maxJsonBytes []byte
	centroidJsonBytes,err = json.Marshal(cluster.GetCentroid().GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}
	minJsonBytes,err = json.Marshal(cluster.min.GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}
	maxJsonBytes,err = json.Marshal(cluster.max.GetVector())
	if err != nil {
		fmt.Println("error:", err)
	}

	query_string = fmt.Sprintf(
		"UPDATE %s" +
			"SET " +
			"time=%d, " +
			"centroid=%s, " +
			"min=%s, " +
			"max=%s, " +
			"size=%d, " +
			"diameter=%.3f, " +
			"radius=%.3f, " +
			"density=%d, " +
			"averageDistance=%d, " +
			"WHERE " +
			"time=%d AND " +
			"centroid=%s AND " +
			"min=%s AND " +
			"max=%s AND " +
			"size=%d",
		cluster.GetRelationName(),
		cluster.GetCentroid().(*deltaPoint).timestamp,
		string(centroidJsonBytes),
		string(minJsonBytes),
		string(maxJsonBytes),
		cluster.GetClusterSize(),
		cluster.diameter,
		cluster.radius,
		cluster.density,
		cluster.averagePairOfPoints,
		cluster.GetCentroid().(*deltaPoint).timestamp,
		string(centroidJsonBytes),
		string(minJsonBytes),
		string(maxJsonBytes),
		cluster.GetClusterSize(),
	)

	logger.StatementExecute(query_string)
}

// Implementation of own functions
func (cluster *halfCluster) CombineWithCluster(other clustering.Cluster,distAlgo clustering.PointDistance)(*halfCluster){
	//c := &halfCluster{fullCluster:newFullCluster()}
	c := new(halfCluster)
	c.simpleCluster = cluster.simpleCluster
	c.name = CLUSTER_DB_TABLE_NAME

	tmp := newFullCluster(other,distAlgo)
	if c.simpleCluster != nil {
		c.simpleCluster = c.simpleCluster.CombineWithCluster(tmp.simpleCluster,distAlgo)
	} else {
		c.simpleCluster = tmp.simpleCluster
	}

	return c
}
func (cluster *halfCluster) String()(string){
	return fmt.Sprintf(
		"HalfCluster " +
			"(size: %d)\t" +
			"Centroid: %s\t" +
			"Min/Max: %+v\t" +
			"Diam: %.3f\t" +
			"Rad: %.3f\t" +
			"Avg: %.3f\t" +
			"Dens: %.3f",
		cluster.GetClusterSize(),
		cluster.centroid,
		[]clustering.Point{cluster.min,cluster.max},
		cluster.diameter,
		cluster.radius,
		cluster.averagePairOfPoints,
		cluster.density)
}
func NewHalfCluster(name string, cluster clustering.Cluster, distAlgo clustering.PointDistance)(obj *halfCluster){
	tmp := newFullCluster(cluster,distAlgo)
	obj = &halfCluster{simpleCluster:tmp.simpleCluster}
	obj.name = name
	return
}
