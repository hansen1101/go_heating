package agent

import (
	"github.com/hansen1101/go_heating/system"
	"fmt"
)

const (
	BOILER_TARGET_FALLBACK = 40000
	BOILER_MAX_TOP = 45000
)

type SimpleHeatingAgent struct {
	config_chan chan int
	config_request_generator system.ConfigRequester
}

func NewSimpleHeatingAgent(config_oracle_available bool)(a *SimpleHeatingAgent){
	if config_oracle_available {
		a = &SimpleHeatingAgent{make(chan int),system.MakeConfigRequest}
	} else {
		a = &SimpleHeatingAgent{}
	}
	return
}

//type Policy func(system.SystemState)(system.Action)

// Implementation of HeatingAgent interface
func (self *SimpleHeatingAgent) GetAction(percept *system.Percept)(action *system.Action){
	// init a new action with default roll out
	action = new(system.Action)

	// update state by processing percept

	// requesting percepts reward

	// request action from policy
	boiler_target := BOILER_TARGET_FALLBACK
	if self.config_chan != nil {
		fmt.Printf("[Agent]\ttry to get boiler target...")
		self.config_request_generator(self.config_chan,percept)
		boiler_target = <-self.config_chan
	}
	fmt.Printf(" received: %d\n",boiler_target)
	burnerOn,triangleOn := waterNeedsHeating(percept,getState().GetBurnerState(),boiler_target)
	timePassedSinceLastTransition := percept.CurrentTime.Sub(getState().Time)
	action.SetBurnerState(burnerOn)
	action.SetTriangleState(triangleOn)
	if timePassedSinceLastTransition.Minutes() < 1.5 {
		action.SetWPumpState(getState().GetWState())
	} else {
		action.SetWPumpState(energyIsAvailable(percept))
	}
	action.SetHPumpState(radiatorsNeedEnergy(percept))

	// update internal fields

	return
}

func getState()(state *system.ActorState){
	if lastState != nil {
		state = lastState.(*system.ActorState)
	}
	return
}

func waterNeedsHeating(percept *system.Percept, burnerIsOn bool, boilerTarget int)(burner bool,triangle bool){
	triangle = true
	burner = false
	if percept.BoilerTopTemp.GetValue() >= 45000 {
		triangle = false
	}
	if percept.BoilerMidTemp.GetValue() < boilerTarget - 2000 && !burnerIsOn {
		burner= true
	} else if percept.BoilerMidTemp.GetValue() < boilerTarget + 3500 && burnerIsOn {
		burner= true
	}
	return
}

func energyIsAvailable(percept *system.Percept)(bool){
	if percept.WForeRunTemp.GetValue() + 300 < percept.KettleTemp.GetValue() {
	//if percept.BoilerMidTemp.GetValue() < percept.KettleTemp.GetValue() {
		return true
	} else {
		return false
	}
}

func radiatorsNeedEnergy(percept *system.Percept)(bool){
	if percept.OutsideTemp.GetValue() < 19000 && (percept.CurrentTime.Hour() >= 5 && percept.CurrentTime.Minute() >= 30 || percept.CurrentTime.Hour() >= 6) && percept.CurrentTime.Hour() < 23 {
		return true
	} else {
		return false
	}
}
