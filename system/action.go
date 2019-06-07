package system

import (
	"bytes"
	"fmt"
	"github.com/hansen1101/go_heating/system/logger"
)

const(
	ACTION_TABLE = "actions"
)

// A RollOut is a function responsible for controlling the actors according to an action
type RollOut func(*Action)()

type Action struct {
	//logger.Logable
	hPumpThrottle, wPumpThrottle float64
	hPumpState, wPumpState, burnerState, triangleState bool
}

func NewAction(wFreq, hFreq float64, h,w,b,t bool)(a *Action){
	a = &Action{
		wPumpThrottle:wFreq,
		hPumpThrottle:hFreq,
		hPumpState:h,
		wPumpState:w,
		burnerState:b,
		triangleState:t,
	}
	return
}

func (self *Action) String()(string){
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("\n[ACTION]\tBurner:%v | HeatPump:%v | BoilerPump:%v | Triangle:%v\n",self.burnerState,self.hPumpState,self.wPumpState,self.triangleState))
	buffer.WriteString(fmt.Sprintln())
	return buffer.String()
}

// Implementation of relation interface
func (a *Action) GetRelationName()(string){
	return ACTION_TABLE
}
func (a *Action) CreateRelation()(){
	var stmnt_string string

	stmnt_string = fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s(" +
		"a_id INT NOT NULL AUTO_INCREMENT," +
		"burnerState BIT(1) NULL DEFAULT 0," +
		"wPumpState BIT(1) NULL DEFAULT 0," +
		"hPumpState BIT(1) NULL DEFAULT 0," +
		"hPumpFreq FLOAT SIGNED NULL DEFAULT 0.0," +
		"triangleState BIT(1) NULL DEFAULT 0,"+
		"PRIMARY KEY(a_id)," +
		"UNIQUE action_settings (burnerState,wPumpState,hPumpState,hPumpFreq,triangleState)," +
		"INDEX action_value (burnerState,wPumpState,hPumpState,hPumpFreq,triangleState)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1",
		a.GetRelationName())

	logger.StatementExecute(stmnt_string)
}
func (a *Action) Insert(val ...interface{})(){
	var stmnt string

	var burner, wPump, hPump, triangle int
	var hFreq float64
	if a.burnerState {
		burner = 1
	}

	if a.wPumpState {
		wPump = 1
	}

	hFreq = a.hPumpThrottle
	if a.hPumpState {
		hPump = 1
	} else {
		hPump = 0
		hFreq = 0.0
	}

	if a.triangleState {
		triangle = 1
	}

	stmnt = fmt.Sprintf(
		"INSERT IGNORE INTO %s" +
		"(burnerState,wPumpState,hPumpState,hPumpFreq,triangleState)" +
		" VALUES " +
		"(b'%d',b'%d',b'%d',%.2f,b'%d')",
		a.GetRelationName(),burner,wPump,hPump,hFreq,triangle)

	logger.StatementExecute(stmnt)
}
func (a *Action) Delete()(){
	//@todo
}
func (a *Action) Update(val ...interface{})(){
	//@todo
}

func (a *Action) GetBurnerState()(bool){
	return a.burnerState
}
func (a *Action) GetHPumpState()(bool){
	return a.hPumpState
}
func (a *Action) GetWPumpState()(bool){
	return a.wPumpState
}
func (a *Action) GetHPumpThrottle()(float64){
	return a.hPumpThrottle
}
func (a *Action) GetWPumpThrottle()(float64){
	return a.wPumpThrottle
}
func (a *Action) GetTriangleState()(bool){
	return a.triangleState
}

func (a *Action) SetBurnerState(v bool)(){
	a.burnerState = v
}
func (a *Action) SetHPumpState(v bool)(){
	a.hPumpState = v
}
func (a *Action) SetWPumpState(v bool)(){
	a.wPumpState = v
}
func (a *Action) SetHPumpThrottle(v float64)(){
	a.hPumpThrottle = v
}
func (a *Action) SetWPumpThrottle(v float64)(){
	a.wPumpThrottle = v
}
func (a *Action) SetTriangleState(v bool)(){
	a.triangleState = v
}
