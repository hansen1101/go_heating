package logger

import (
	"database/sql"
	"fmt"
)

var (
	db *sql.DB	// global pointer to database connection for all Logable interactions
	dbName string	// global name of the database schema
)

type Logable interface {
	GetRelationName()(string)
	CreateRelation()()
	Insert(...interface{})()
	Delete(...interface{})()
	Update(...interface{})()
}

// Setter for the database connection pointer that should be used for database interactions
// @param sql.DB pointer to the database which should be used for successive database interactions
// @param string name of the database
func SetDatabase(dbase *sql.DB, dbaseName string){
	db = dbase
	dbName = dbaseName
}

// Queries the mysql information_schema database and check if an entry for
// the given (database,table) tuple exist
// @param database name
// @param table name
// @return true if entry for database.table exists in information_schema.tables Logable
func TableExists(database, table string)(bool){
	var query_string string
	var stmtOut *sql.Rows
	var err error
	query_string = fmt.Sprintf(
		"SELECT table_schema, table_name FROM information_schema.tables " +
			"WHERE table_schema = '%s' " +
			"AND table_name = '%s' " +
			"LIMIT 1;",
		database,table)
	stmtOut, err = db.Query(query_string)
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer func(){
		stmtOut.Close()
	}()
	if !stmtOut.Next() {
		// table does not exists in database
		return false
	}
	return true
}

// Prepares and executes a given sql statement provided as string.
// @param string the mysql statement to execute
func StatementExecute(stmnt_string string)(){
	var err error

	if db == nil {
		return
	}
	stmt,err := db.Prepare(stmnt_string)
	defer func(){
		//@debug database logging fmt.Printf("Modification: %s done.\n",stmnt_string)
		stmt.Close()
	}()

	if err == nil {
		_, err = stmt.Exec()
	}
	if err != nil {
		fmt.Print(err)
	}
}

// Takes a slice of Logable objects and checks if there exists a table for that relation
// in the database schema.
// If no table exists a new table is created for each Logable object is created
// @param slice of Logable objects to check against
func InitDbRelations(logObjs *[]Logable)() {
	for _,obj := range *logObjs {
		if !TableExists(dbName,obj.GetRelationName()){
			obj.CreateRelation()
		}
	}
}
