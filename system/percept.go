package system

import (
	"bytes"
	"fmt"
	"github.com/hansen1101/go_heating/system/logger"
	"time"
	"github.com/hansen1101/go_heating/system/w1"
)

const (
	PERCEPT_TABLE = "percepts"
)

type Percept struct {
	//logger.Relation
	CurrentTime time.Time
	OutsideTemp, BoilerMidTemp, BoilerTopTemp, KettleTemp, HForeRunTemp, HReverseRunTemp, WForeRunTemp, WReverseRunTemp, WIntakeTemp *w1.Temperature
	Valid bool
}

// A PerceptGenerator is a function that generates a Percept for a given time
type PerceptGenerator func(timestamp *time.Time) (percept *Percept)

func (p *Percept) String()(string){
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("\nPercept at:\t%s\n",p.CurrentTime.String()))
	buffer.WriteString(fmt.Sprintf("%s Temperature Value:\t%d\t(%v)\n",p.OutsideTemp.GetSensorLogic(),p.OutsideTemp.GetValue(),p.OutsideTemp.IsValid()))
	buffer.WriteString(fmt.Sprintf("%s Temperature Value:\t%d\t(%v)\n",p.BoilerTopTemp.GetSensorLogic(),p.BoilerTopTemp.GetValue(),p.BoilerTopTemp.IsValid()))
	buffer.WriteString(fmt.Sprintf("BoilerMidTemp Temperature Value:\t%d\t(%v)\n",p.BoilerMidTemp.GetValue(),p.BoilerMidTemp.IsValid()))
	buffer.WriteString(fmt.Sprintf("KettleTemp Temperature Value:\t%d\t(%v)\n",p.KettleTemp.GetValue(),p.KettleTemp.IsValid()))
	buffer.WriteString(fmt.Sprintf("HForeRun Temperature Value:\t%d\t(%v)\n",p.HForeRunTemp.GetValue(),p.HForeRunTemp.IsValid()))
	buffer.WriteString(fmt.Sprintf("HReverse Temperature Value:\t%d\t(%v)\n",p.HReverseRunTemp.GetValue(),p.HReverseRunTemp.IsValid()))
	buffer.WriteString(fmt.Sprintf("WForeRun Temperature Value:\t%d\t(%v)\n",p.WForeRunTemp.GetValue(),p.WForeRunTemp.IsValid()))
	buffer.WriteString(fmt.Sprintf("WReverse Temperature Value:\t%d\t(%v)\n",p.WReverseRunTemp.GetValue(),p.WReverseRunTemp.IsValid()))
	buffer.WriteString(fmt.Sprintln())
	return buffer.String()
}
func (p *Percept) GetRelationName()(string) {
	return PERCEPT_TABLE
}
func (p *Percept) CreateRelation()() {
	var stmnt_string string

	stmnt_string = fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s(" +
			"p_id INT NOT NULL AUTO_INCREMENT," +
			"time INT NOT NULL DEFAULT 0," +
			"OutsideTemp INT SIGNED NULL DEFAULT 0," +
			"BoilerMidTemp INT SIGNED NULL DEFAULT 0," +
			"BoilerTopTemp INT SIGNED NULL DEFAULT 0," +
			"KettleTemp INT SIGNED NULL DEFAULT 0," +
			"H1ForeRunTemp INT SIGNED NULL DEFAULT 0," +
			"H1ReverseRunTemp INT SIGNED NULL DEFAULT 0," +
			"H2ForeRunTemp INT SIGNED NULL DEFAULT 0," +
			"WForeRunTemp INT SIGNED NULL DEFAULT 0," +
			"WReverseRunTemp INT SIGNED NULL DEFAULT 0," +
			"PRIMARY KEY(p_id)," +
			"UNIQUE value_key (" +
			"time," +
			"OutsideTemp," +
			"BoilerMidTemp," +
			"BoilerTopTemp," +
			"KettleTemp," +
			"H1ForeRunTemp," +
			"H1ReverseRunTemp," +
			"H2ForeRunTemp," +
			"WForeRunTemp," +
			"WReverseRunTemp)" +
			")ENGINE=InnoDB DEFAULT CHARSET=latin1",
		p.GetRelationName())

	logger.StatementExecute(stmnt_string)
}
func (p *Percept) Insert(val ...interface{})() {
	var stmnt_string string

	stmnt_string = fmt.Sprintf(
		"INSERT IGNORE INTO %s" +
			"(" +
			"time," +
			"OutsideTemp," +
			"BoilerMidTemp," +
			"BoilerTopTemp," +
			"KettleTemp," +
			"H1ForeRunTemp," +
			"H1ReverseRunTemp," +
			"H2ForeRunTemp," +
			"WForeRunTemp," +
			"WReverseRunTemp)" +
			" VALUES " +
			"(%d,%d,%d,%d,%d,%d,%d,%d,%d,%d)",
		p.GetRelationName(),
		p.CurrentTime.Unix(),
		p.OutsideTemp.GetValue(),
		p.BoilerMidTemp.GetValue(),
		p.BoilerTopTemp.GetValue(),
		p.KettleTemp.GetValue(),
		p.HForeRunTemp.GetValue(),
		p.HReverseRunTemp.GetValue(),
		p.WIntakeTemp.GetValue(),
		p.WForeRunTemp.GetValue(),
		p.WReverseRunTemp.GetValue(),
	)

	go logger.StatementExecute(stmnt_string)
}
func (p *Percept) Delete(val ...interface{})() {
	//@todo
}
func (p *Percept) Update(val ...interface{})() {
	//@todo
}
func (p *Percept) GetBoilerDelta(successor *Percept)(tempDelta int,recordingDuration time.Duration){
	if successor == nil {
		return
	} else {
		tempDelta = p.BoilerMidTemp.GetValue() - successor.BoilerMidTemp.GetValue()
		recordingDuration = p.CurrentTime.Sub(successor.CurrentTime)
	}
	return
}
func (p *Percept) GetKettleDelta(successor *Percept)(tempDelta int,recordingDuration time.Duration){
	if successor == nil {
		return
	} else {
		tempDelta = p.KettleTemp.GetValue() - successor.KettleTemp.GetValue()
		recordingDuration = p.CurrentTime.Sub(successor.CurrentTime)
	}
	return
}
func (p *Percept) Validate()(bool){
	//@todo needs to be implemented
	p.Valid = true
	return true
}
func (p *Percept) SetTime(t time.Time)(){
	p.CurrentTime = t
}
func (p *Percept) IsValid()(bool){
	return p.Valid
}
