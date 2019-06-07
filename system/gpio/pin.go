// The gpio package can be used to setup and access the raspberrie's gpio pins.
// If the configuration and access happens via file access an internal loop is implemented in
// the corresponding method that ensures the file access is repeated until a fixed bound of repeats is reached
// or the file access was successfully.
package gpio

import(
	"os"
	"os/exec"
	"log"
	"strings"
	"strconv"
//	"fmt"
)

// Type for the mode of a gpio pin
type PinMode uint8

// Type for the ID of gpio pins on raspberry board
type GpioId uint8

// Constants declatation
const(
	OUTPUT PinMode = 1+iota
	INPUT
	GPIO2 GpioId = iota
	GPIO3
	GPIO4
	GPIO5
	GPIO6
	GPIO7
	GPIO8
	GPIO9
	GPIO10
	GPIO11
	GPIO12
	GPIO13
	GPIO14
	GPIO15
	GPIO16
	GPIO17
	GPIO18
	GPIO19
	GPIO20
	GPIO21
	GPIO22
	GPIO23
	GPIO24
	GPIO25
	GPIO26
	GPIO27

	DIRECTION_FILE_NAME = "direction"
	IN = "in"
	OUT = "out"
	ACTIVE_LOW_FILE_NAME = "active_low"
	ALTRUE = "1"
	ALFALSE = "0"
	VALUE_FILE_NAME = "value"
	FILE_ACCESS_BOUND = 1000
)

var(
	EXPORT_FILE = "/sys/class/gpio/export"
	UNEXPORT_FILE = "/sys/class/gpio/unexport"
	PATH_PREFIX = "/sys/class/gpio/gpio"
)

// struct representing a gpio pin internally
type Pin struct {
	pin GpioId
	mode PinMode
	value bool
	activeLow bool
	path string
}

// Constructor method for the Pin struct
// mode set to INPUT by default
// value set to FALSE by default
// @return pointer to Pin struct with
func NewPin()(p *Pin){
	p = &Pin{mode:INPUT,value:false}
	return
}

// Configures the internal fields of a Pin struct and calls the export function,
// which introduces the pin to the os with the given configuration.
// @param: pin the Pin's id on the raspberry board
// @param: mode weather read or write access is required
// @param: activeLow (optional) true if the pin should be run in active_low mode (raspberry default is false)
func (g *Pin) PinMode(pin GpioId, mode PinMode, activeLow ...bool){
	g.pin = pin
	g.mode = mode

	s := []string{PATH_PREFIX,strconv.Itoa(int(g.pin)),"/"}
    	g.path = strings.Join(s,"")

	if len(activeLow) > 0 {
		g.activeLow = activeLow[0]
	}

	g.export()
	return
}

// Introduces the pin configuration to the raspberry using the WiringPi framework
// @internal: method is in ALPHA mode. WiringPi framework must be installed.
func (g *Pin) exportWiringPi() {
	var cmd *exec.Cmd
	switch g.mode {
		case OUTPUT:
			cmd = exec.Command("gpio", "export", strconv.Itoa(int(g.pin)), "out")
		default:
			cmd = exec.Command("gpio", "export", strconv.Itoa(int(g.pin)), "in")
	}
	err := cmd.Run()
	if err != nil {
		log.Fatal(err) //os.Exit(1)
	}

	g.setActiveLowConfig()

	return
}

// Introduces the pin's configuration to the raspberry via the configuration files.
// If the pin's direction is OUT, the initial value is set to LOW.
func (g *Pin) export() {
	info,_ := os.Stat(g.path)
	if info == nil {
		file,err := os.OpenFile(EXPORT_FILE,os.O_WRONLY,os.ModeExclusive)
		defer file.Close()
		if err != nil {
			log.Fatal(err)
		}
		file.WriteString(strconv.Itoa(int(g.pin)))
	}

	g.setActiveLowConfig()
	g.setDirectionConfig()
	g.SetValue(false)

	return
}

// Unexports the pin which deactivates it for further use.
func (g *Pin) Unexport() {
	file,err := os.OpenFile(UNEXPORT_FILE,os.O_WRONLY,os.ModeExclusive)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	file.WriteString(strconv.Itoa(int(g.pin)))

	return
}

// Sets the value for a gpio pin. This method is only applicable
// to OUT direction pins and as no effect if Pin.mode is INPUT.
// The method sets the associated gpio value at the raspberry via file
// access to the corresponding value file. The function is active_low safe.
// @param val if true the pin is set to mode HIGH else to mode LOW
func (g *Pin) SetValue(val bool){
	if g.mode == INPUT || g.value == val {
		return
	}
	g.value = val
	var setTo string
	if g.value {
		if g.activeLow {
			setTo = "0"
		} else {
			setTo = "1"
		}
	} else {
		if g.activeLow {
			setTo = "1"
		} else {
			setTo = "0"
		}
	}

	attempts := 0
	for success:=false; success != true; {
		file,err := os.OpenFile(g.path+VALUE_FILE_NAME,os.O_WRONLY,os.ModeExclusive)
		defer file.Close()
		attempts++
		if err != nil {
			if attempts > FILE_ACCESS_BOUND {
				log.Fatal(err)
			}
		} else {
			file.WriteString(setTo)
			success = true
		}
	}
	return
}

// Reads and returns the current value from any kind of pins. If the pin's direction
// is set to OUT the internal value is returned directly. For INPUT pins the actual
// value is read from the file each time the function is called.
// This function is active_low safe which means, if the returned value is TRUE the pin's
// physical signal is HIGH.
// @return true if signal at pin is HIGH, false if signal is LOW
func (g *Pin) GetValue()(val bool){
	if g.mode == OUTPUT {
		return g.value
	}
	buf := make([]byte,1,1)
	attempts := 0
	for success:=false; success != true; {
		file,err := os.OpenFile(g.path+VALUE_FILE_NAME,os.O_RDONLY,os.ModeTemporary)
		defer file.Close()
		attempts++
		if err != nil {
			if attempts > FILE_ACCESS_BOUND {
				log.Fatal(err)
			}
		} else {
			file.Read(buf)
			success = true
		}
	}
	file_val,_ := strconv.Atoi(string(buf[0]))

	if g.activeLow {
		if file_val == 1 {
			val = true
		} else {
			val = false
		}
	} else {
		if file_val == 1 {
			val = false
		} else {
			val = true
		}
	}

	g.value = val
	return val
}

// Getter for the GPIO board id of the pin
func (g *Pin) GetGpioId()(GpioId){
	return g.pin
}

// Configures the pin's direction value at the raspberry according to the internal mode value
// of the Pin struct.
func (g *Pin) setDirectionConfig(){
	var setTo string
	switch g.mode {
		case OUTPUT:
			setTo = OUT
		default:
			setTo = IN
	}

	attempts := 0
	for success:=false; success != true; {
		file, err := os.OpenFile(g.path + DIRECTION_FILE_NAME, os.O_WRONLY, os.ModeExclusive)
		defer file.Close()
		attempts++
		if err != nil {
			if attempts > FILE_ACCESS_BOUND {
				log.Fatal(err)
			}
		} else {
			file.WriteString(setTo)
			success = true
		}
	}

	return
}

// Configures the pin's active_low value at the raspberry according to the internal value
// of the Pin struct.
func (g *Pin) setActiveLowConfig(){
	var setTo string
	if g.activeLow {
		setTo = ALTRUE
	} else {
		setTo = ALFALSE
	}
	attempts := 0
	for success:=false; success != true; {
		file,err := os.OpenFile(g.path + ACTIVE_LOW_FILE_NAME,os.O_WRONLY,os.ModeExclusive)
		defer file.Close()
		attempts++
		if err != nil {
			if attempts > FILE_ACCESS_BOUND {
				log.Fatal(err)
			}
		} else {
			file.WriteString(setTo)
			success = true
		}
	}
	return
}
