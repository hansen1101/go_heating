package system

import (
	"time"
	"fmt"
	"github.com/hansen1101/go_heating/system/logger"
)

const (
	BURNER_TRANSITION = 7
	H_PUMP_FREQ_TRANSITION = 5
	W_PUMP_TOGGLE_TRANSITION = 3
	H_PUMP_TOGGLE_TRANSITION = 16
	TRIANGLE_TRANSITION = 32
	SYSTEM_STATE_TABLE = "system_states"
)

var (
	burner_transition_penalty Reward = -1000
	w_pump_transition_penalty Reward = -10
	no_transit_reward Reward = 2
	h_pump_freq_reward Reward = 1
)

type Reward int

type State interface{
	Successor(*Action)(State)
	CompareTo(State)(int)
	Equals(State)(bool)
	Reward(*Action,State)(Reward)
	IsTerminal()(bool)
}

type SystemState interface {
	State
	GetBurnerState()(bool)
	GetTriangleState()(bool)
	GetWFrequency()(int)
	GetHFrequency()(int)
	GetWState()(bool)
	GetHState()(bool)
}


// implements SystemState interface, logger.Logable interface
type ActorState struct{
	//SystemState
	//logger.Logable

	time.Time
	wPumpState, hPumpState, burnerState, triangleState bool
	wPumpFreq, hPumpFreq int
}

// ActorState implements the Stringer interface.
func (s *ActorState) String() string {
	return fmt.Sprintf("[STATE]\t[Time: %v]\t[B:%v] - [W:%v (%d)] - [H:%v (%d)]",s.Time,s.burnerState,s.wPumpState,s.wPumpFreq,s.hPumpState,s.hPumpFreq)
}

func (s *ActorState) SetTimeStamp(t time.Time)(){
	s.Time = t
	return
}

func approxPumpFreq(exactValue float64)(approxValue int){
	rest := int(exactValue) % 5
	approxValue = int(exactValue) - rest
	return
}

// implementation of State interface

func (s *ActorState) Successor(a *Action)(sPrime State){
	if a != nil {
		sPrime = &ActorState{
			burnerState:a.GetBurnerState(),
			triangleState:a.GetTriangleState(),
			wPumpState:a.GetWPumpState(),
			wPumpFreq:approxPumpFreq(a.GetWPumpThrottle()),
			hPumpState:a.GetHPumpState(),
			hPumpFreq:approxPumpFreq(a.GetHPumpThrottle()),
		}
	}
	return
}
func (s *ActorState) CompareTo(sPrime State)(means int){
	if s.burnerState != sPrime.(*ActorState).burnerState {
		means += BURNER_TRANSITION
	}
	if s.triangleState != sPrime.(*ActorState).triangleState {
		means += TRIANGLE_TRANSITION
	}
	if s.wPumpState != sPrime.(*ActorState).wPumpState {
		means += W_PUMP_TOGGLE_TRANSITION
	}
	if s.hPumpState != sPrime.(*ActorState).hPumpState {
		means += H_PUMP_TOGGLE_TRANSITION
	}
	if s.hPumpFreq != sPrime.(*ActorState).hPumpFreq {
		means += H_PUMP_FREQ_TRANSITION
	}
	return
}
func (s *ActorState) Equals(sPrime State)(bool){
	if sPrime != nil && s.CompareTo(sPrime) == 0 {
		return true
	}
	return false
}
func (s *ActorState) Reward(a *Action, sPrime State)(reward Reward){
	switch s.CompareTo(sPrime) {
	case 0:
		// no transition
		return no_transit_reward
	case 3:
		// w_pump toggled
		return w_pump_transition_penalty
	case 5:
		// h_pump_freq_transition
		return h_pump_freq_reward
	case 7:
		// burner toggle
		return burner_transition_penalty
	default:
		// w_pump toggle and h_pump_freq transition
		return w_pump_transition_penalty
	}
}
func (s *ActorState) IsTerminal()(bool){
	return false
}

// implementation of SystemState interface
func (s *ActorState) GetBurnerState()(bool){
	return s.burnerState
}
func (s *ActorState) GetTriangleState()(bool){
	return s.triangleState
}
func (s *ActorState) GetWFrequency()(int){
	return s.wPumpFreq
}
func (s *ActorState) GetHFrequency()(int){
	return s.hPumpFreq
}
func (s *ActorState) GetWState()(bool){
	return s.wPumpState
}
func (s *ActorState) GetHState()(bool){
	return s.hPumpState
}

// implementation of Logable interface
func (s *ActorState) GetRelationName()(string){
	return SYSTEM_STATE_TABLE
}
func (s *ActorState) CreateRelation()(){
	var query_string string

	query_string = fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s(" +
		"s_id INT NOT NULL AUTO_INCREMENT," +
		"time INT NOT NULL DEFAULT 0," +
		"burnerState BIT(1) NULL DEFAULT 0," +
		"triangleState BIT(1) NULL DEFAULT 0," +
		"wPumpState BIT(1) NULL DEFAULT 0," +
		"wPumpFreq INT UNSIGNED NULL DEFAULT 0," +
		"hPumpState BIT(1) NULL DEFAULT 0," +
		"hPumpFreq INT UNSIGNED NULL DEFAULT 0," +
		"PRIMARY KEY(s_id)," +
		"UNIQUE system_values (time,burnerState,wPumpState,hPumpFreq)" +
		")ENGINE=InnoDB DEFAULT CHARSET=latin1",
		s.GetRelationName())

	logger.StatementExecute(query_string)
}
func (s *ActorState) Insert(val ...interface{})(){
	var query_string string

	var burner, triangle, wPump, hPump int
	if s.burnerState {
		burner = 1
	}

	if s.triangleState {
		triangle = 1
	}

	if s.wPumpState {
		wPump = 1
	}

	if s.hPumpState {
		hPump = 1
	}

	query_string = fmt.Sprintf(
		"INSERT IGNORE INTO %s" +
		"(time,burnerState,triangleState,wPumpState,hPumpState,hPumpFreq)" +
		" VALUES " +
		"(%d,b'%d',b'%d',b'%d',b'%d',%d)",
		s.GetRelationName(),s.Time.Unix(),burner,triangle,wPump,hPump,s.hPumpFreq)

	logger.StatementExecute(query_string)
}
func (s *ActorState) Delete(val ...interface{})(){
	//@todo
}
func (s *ActorState) Update(val ...interface{})(){
	//@todo
}
