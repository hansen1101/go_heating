package system

import (
	"fmt"
	"time"
	"github.com/hansen1101/go_heating/system/gpio"
)

const (
	OFF_FREQ = 0.0
	ACC_BASIS = 50.0
)

type Pump struct {
	state bool
	current, max_freq, min_freq, acceleration, delta float64
	power_gpio, inc_gpio, dec_gpio *gpio.Pin
}

func NewPump(max,min,acc,delta float64, power,inc,dec *gpio.Pin)(p *Pump){
	p = &Pump{
		state:false,
		current:OFF_FREQ,
		max_freq:max,
		min_freq:min,
		acceleration:acc,
		delta:delta,
		power_gpio:power,
		inc_gpio:inc,
		dec_gpio:dec}
	return
}

func (p *Pump) DestructPump(){
	defer func() {
		(p.power_gpio).Unexport()
		(p.inc_gpio).Unexport()
		(p.dec_gpio).Unexport()
	}()
	p.Deactivate()
}

func (p *Pump) toggle() (new_state bool) {
	new_state = !p.state
	p.power_gpio.SetValue(new_state)
	<-time.After(time.Duration(int((p.min_freq+p.delta)*1000*p.acceleration/ACC_BASIS)) * time.Millisecond)
	return
}

func (p *Pump) Activate() {
	if !p.state {
		p.state = p.toggle()
		p.current = p.min_freq
	}
	return
}

func (p *Pump) Deactivate() {
	if p.state {
		p.UpdateFrequencyTo(p.min_freq)
		p.state = p.toggle()
		p.current = OFF_FREQ
	}
	return
}

func (p *Pump) UpdateFrequencyTo(target float64) {
	p.updateFrequencyBy(target-p.current)
	return
}

func (p *Pump) updateFrequencyBy(steps float64) {
	// relais to use depends on case
	var relais *gpio.Pin

	// update current field of struct depending on parameter
	p.current += steps
	if p.current > p.max_freq {
		steps -= p.current - p.max_freq
		p.current = p.max_freq
	} else if p.current < p.min_freq {
		steps += p.min_freq - p.current
		p.current = p.min_freq
	}

	switch {
		case steps < 0.0:
			// trigger decrease relais
			relais = p.dec_gpio
			steps *= -1
		//	break
		case steps > 0.0:
			// trigger increase relais
			relais = p.inc_gpio
		//	break
		default:
			return
	}

	// activate relais
	relais.SetValue(true)

	defer relais.SetValue(false)

	<-time.After(time.Duration(int((steps+p.delta)*1000*p.acceleration/ACC_BASIS)) * time.Millisecond)

	/*
    	select {
		// wait until pump reached target frequency
		case <-time.After(time.Duration(int((steps+p.delta)*1000*p.acceleration/ACC_BASIS)) * time.Millisecond):
			// deactivate relais
			relais.SetValue(false)
			break
    	}*/

	return
}

func (p *Pump) GetState()(bool,float64){
	return p.state,p.current
}

func (p *Pump) IsActive()(bool){
	return p.state
}

func (p *Pump) GetMinFreq()(float64) {
	return p.min_freq
}

func (p *Pump) GetCurrentFreq()(float64) {
	return p.current
}

func (p *Pump) GetMaxFreq()(float64) {
	return p.max_freq
}

func (p *Pump) String()(string){
	return fmt.Sprintf("Pump State [active:%v frequency:%.2f]\tGPIO fingerprint [%v:%v %v:%v %v:%v]\n",
		p.state,
		p.current,
		p.power_gpio.GetGpioId(),
		p.power_gpio.GetValue(),
		p.inc_gpio.GetGpioId(),
		p.inc_gpio.GetValue(),
		p.dec_gpio.GetGpioId(),
		p.dec_gpio.GetValue())
}
