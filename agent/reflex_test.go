package agent

import (
	"testing"
	"github.com/hansen1101/go_heating/system"
	"github.com/hansen1101/go_heating/system/gpio"
	"time"
)

type testcase struct {
	agent ReflexAgent
	percept system.Percept
}

var h_pump = system.NewPump(50.0,15.0,25.0,0.2,nil,nil,nil)
var w_pump = system.NewPump(50.0,15.0,25.0,0.2,nil,nil,nil)
var burner = gpio.NewPin()

var agents = []ReflexAgent{
	ReflexAgent{w_pump,h_pump,burner},
}

var percepts = []system.Percept {
	system.Percept{time.Now(),12500, 43000, 56000, 38000, 32000, 26000, 32000, 31000, 53000},
}

func TestGetHSestatestatetPoint(t *testing.T){
	var params = []struct{
		b,w bool
		h float64
	} {
		{true,true,15.2},
		{true,true,65.4},
		{true,true,-15.7},
		{true,true,-0.0},
		{true,false,15.3},
		{true,false,65.5},
		{true,false,-15.0},
		{true,false,0.2},
		{false,false,15.0},
		{false,false,65.3},
		{false,false,-15.6},
		{false,false,0.0},
	}

	for _,param := range params {
		state := agents[0].initStateHistory(param.b,param.w,param.h)
		if len(state) != 1 || cap(state) != 1 {
			t.Error(
				"For", param,
				"expected cap", 1,
				"len", 1,
				"got cap", cap(state),
				"len", *state[0],
			)
		}
	}
}
