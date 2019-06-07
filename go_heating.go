package main

import (
	"os"
	"os/signal"
	"time"
	"syscall"
	"fmt"
	"log"
	"strings"
	"sync"
	"io"
	"runtime"
	"math"
	"math/rand"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"github.com/hansen1101/go_heating/learner"
	"github.com/hansen1101/go_heating/system"
	"github.com/hansen1101/go_heating/system/logger"
	"github.com/hansen1101/go_heating/system/gpio"
	"github.com/hansen1101/go_heating/system/w1"
	"github.com/hansen1101/go_heating/agent"
)

const(
	RELAIS_1 = gpio.GPIO17
	RELAIS_2 = gpio.GPIO18
	RELAIS_3 = gpio.GPIO22
	RELAIS_4 = gpio.GPIO27
	RELAIS_5 = gpio.GPIO5
	RELAIS_6 = gpio.GPIO25
	RELAIS_7 = gpio.GPIO23
	RELAIS_8 = gpio.GPIO24
	BUTTON_IN = gpio.GPIO6
	CHIMNEY_LED = gpio.GPIO13
	W1_REPLICATION_LEVEL = 4
	WORKER_MAX_REPLICATION_LEVEL = 50
	W1_SENSOR_COUNT = 9
	PERCEPT_HISTORY_LENGTH = 300
	//OUTSIDE_SENSOR = "28-000006980f74"
	OUTSIDE_SENSOR = "28-000007c5f668" // OK Outside Temperature
	OUTSIDE = "OUTSIDE"
	//BOILER_MID_SENSOR = "28-000005d8afbd"
	BOILER_TOP_SENSOR = "28-0000030f9a99" // OK Boiler Top [TWO]
	TWO = "TWO" // Boiler Top [TWO]
	//BOILER_TOP_SENSOR =  "28-000006a335b2"
	BOILER_MID_SENSOR=  "28-0000030f4ff5" // OK Boiler Mid [TPO]
	TPO = "TPO"
	//KETTLE_SENSOR = "28-000005d8f178"
	H_REVERSERUN_SENSOR = "28-0000075c5fd6" // OK Ruecklauf Heizkreis
	H_REV = "H_rev"
	//H_FORERUN_SENSOR = "28-000006981117"
	H_FORERUN_SENSOR = "28-000007c5f57f" // OK Vorlauf Heizkreis
	H_FOR = "H_for"
	//H_REVERSERUN_SENSOR = "28-00000698a022"
	W_FORERUN_SENSOR= "28-0000030f64da" // OK Boiler Bottom [TPU]
	TPU = "TPU"
	//W_FORERUN_SENSOR = "28-00000697b512"
	KETTLE_SENSOR = "28-000007c5cf02" // OK Kessel
	KETTLE = "Kettle"
	//W_REVERSERUN_SENSOR = "28-000007c61ca9"
	W_REVERSERUN_SENSOR = "28-0000075d9c18" // OK Raum
	W_REV = "W_rev"
	//W_INTAKE_SENSOR = "28-000007c61ca9"
	W_INTAKE_SENSOR = "28-0000075d9c18" // Raum
	ROOM = "Room"

	DATABASE_USER string = "heating_logger"
	DATABASE_PASSWD string = "heating"
	DATABASE_NAME string = "heating_controller"		// name of the database schema for logging
	TABLE string = "datalog"			// name of the database table

	DEBUG = false
	DEFAULT_MIN_BOILER_TEMP int = 30000
)

var(
	radiatorPump,
	boilerPump *system.Pump

	burner,
	boilerPump_on,
	boilerPump_inc,
	boilerPump_dec,
	radiatorPump_on,
	radiatorPump_inc,
	radiatorPump_dec,
	triangle_switch,
	chimney_button,
	chimney_led *gpio.Pin

	outsideSensor,
	boilerMidSensor,
	boilerTopSensor,
	kettleSensor,
	hForeRunSensor,
	hReverseRunSensor,
	wForeRunSensor,
	wReverseRunSensor,
	wIntakeSensor []w1.TemperatureLookup

	// map: logical sensor name -> sensor ids (e.g. 'kettle' => '28-00000123456')
	sensorIds map[string]string


	//logfile *os.File
	logfile io.Writer
	logmutex sync.Mutex

	systemAgent agent.HeatingAgent

	// sState is logable
	sState *system.ActorState

	sAction *system.Action

	applyAction system.RollOut

	config_path string = "/usr/local/share/heating_config/config.csv"
	log_path = "/var/log/go_heating.log"
)

// Initializes the GPIO pins used to control the systems actuators
func initGPIO() {
	burner = gpio.NewPin()
	burner.PinMode(RELAIS_2,gpio.OUTPUT,true)

	boilerPump_on = gpio.NewPin()
	boilerPump_on.PinMode(RELAIS_1,gpio.OUTPUT,true)

	boilerPump_inc = gpio.NewPin()
	boilerPump_inc.PinMode(RELAIS_8,gpio.OUTPUT,true)

	boilerPump_dec = gpio.NewPin()
	boilerPump_dec.PinMode(RELAIS_5,gpio.OUTPUT,true)

	radiatorPump_on = gpio.NewPin()
	radiatorPump_on.PinMode(RELAIS_3,gpio.OUTPUT,true)

	radiatorPump_inc = gpio.NewPin()
	radiatorPump_inc.PinMode(RELAIS_6,gpio.OUTPUT,true)

	radiatorPump_dec = gpio.NewPin()
	radiatorPump_dec.PinMode(RELAIS_7,gpio.OUTPUT,true)

	triangle_switch = gpio.NewPin()
	triangle_switch.PinMode(RELAIS_4,gpio.OUTPUT,true)

	chimney_button = gpio.NewPin()
	chimney_button.PinMode(BUTTON_IN,gpio.INPUT,true)

	chimney_led = gpio.NewPin()
	chimney_led.PinMode(CHIMNEY_LED,gpio.OUTPUT,true)
}

// Performs a cleanup and unexports the GPIO pins
func cleanupGPIO(){
	burner.Unexport()
	boilerPump_on.Unexport()
	boilerPump_inc.Unexport()
	boilerPump_dec.Unexport()
	radiatorPump_on.Unexport()
	radiatorPump_inc.Unexport()
	radiatorPump_dec.Unexport()
	chimney_button.Unexport()
	chimney_led.Unexport()
	return
}

// Initializes the mapping of logical sensor names that are used during system routines to sensor ids
func initW1()(){
	sensorIds = make(map[string]string,9)
	sensorIds[TPO] = BOILER_MID_SENSOR
	sensorIds[TWO] = BOILER_TOP_SENSOR
	sensorIds[KETTLE] = KETTLE_SENSOR
	sensorIds[H_FOR] = H_FORERUN_SENSOR
	sensorIds[H_REV] = H_REVERSERUN_SENSOR
	sensorIds[TPU] = W_FORERUN_SENSOR
	sensorIds[ROOM] = W_INTAKE_SENSOR
	sensorIds[W_REV] = W_REVERSERUN_SENSOR
	sensorIds[OUTSIDE] = OUTSIDE_SENSOR
}

// Maps a w1.Temperature pointer to the corresponding field for a given percept.
// This function implements the logic of the hardware sensor setup and the
// corresponding internal processing of the sensor data. Change the assignment
// here if the system has another architecture.
// @return pointer to a w1.temperature struct
// @todo import the mapping from an external configuration file
func SetTempPointerForSensor(percept *system.Percept, temp *w1.Temperature)(*w1.Temperature){
	switch temp.GetSensorLogic() {
	case OUTSIDE:
		percept.OutsideTemp = temp
		return percept.OutsideTemp
	case TPO:
		percept.BoilerMidTemp = temp
		return percept.BoilerMidTemp
	case TWO:
		percept.BoilerTopTemp = temp
		return percept.BoilerTopTemp
	case KETTLE:
		percept.KettleTemp = temp
		return percept.KettleTemp
	case H_FOR:
		percept.HForeRunTemp = temp
		return percept.HForeRunTemp
	case H_REV:
		percept.HReverseRunTemp = temp
		return percept.HReverseRunTemp
	case TPU:
		percept.WForeRunTemp = temp
		return percept.WForeRunTemp
	case W_REV:
		percept.WReverseRunTemp = temp
		return percept.WReverseRunTemp
	case ROOM:
		percept.WIntakeTemp = temp
		return percept.WIntakeTemp
	default:
		return nil
	}
}

// Initializes the system's actuators
func initActors() {
	radiatorPump = system.NewPump(50.0,50.0,25.0,0.2,radiatorPump_on,radiatorPump_inc,radiatorPump_dec)
	boilerPump = system.NewPump(50.0,15.0,25.0,0.2,boilerPump_on,boilerPump_inc,boilerPump_dec)
}

// Instance of PerceptGenerator, thus creates a new percept for a given timestamp.
// Fetches data for all temperature sensors in a replicated fashion (pointer to
// first valid temperature data is taken from each sensor).
// Assigns each incoming temperature data to the corresponding percept field.
// @param pointer to the timestamp the percept is generated for
// @return pointer to the generated percept
// @todo this is not elegant
func fetchSensorData(timestamp *time.Time)(percept *system.Percept) {

	outside := make(chan *w1.Temperature)
	go func(){outside <- w1.First(outsideSensor...)}()

	boilerM := make(chan *w1.Temperature)
	go func(){boilerM <- w1.First(boilerMidSensor...)}()

	boilerT:= make(chan *w1.Temperature)
	go func(){boilerT <- w1.First(boilerTopSensor...)}()

	kettle := make(chan *w1.Temperature)
	go func(){kettle <- w1.First(kettleSensor...)}()

	hfor := make(chan *w1.Temperature)
	go func(){hfor<-w1.First(hForeRunSensor...)}()

	hrev := make(chan *w1.Temperature)
	go func(){hrev<-w1.First(hReverseRunSensor...)}()

	wfor := make(chan *w1.Temperature)
	go func(){wfor<-w1.First(wForeRunSensor...)}()

	wrev := make(chan *w1.Temperature)
	go func(){wrev<-w1.First(wReverseRunSensor...)}()

	win := make(chan *w1.Temperature)
	go func(){win<-w1.First(wIntakeSensor...)}()

	// generate percept
	percept = new(system.Percept)
	percept.Valid = true
	percept.CurrentTime = *timestamp

	var temp *w1.Temperature

	for i:=0; i<W1_SENSOR_COUNT; i++{
		select {
		case temp = <-outside:
			percept.OutsideTemp = temp
		case temp = <-boilerM:
			percept.BoilerMidTemp = temp
		case temp = <-boilerT:
			percept.BoilerTopTemp = temp
		case temp = <-kettle:
			percept.KettleTemp = temp
		case temp = <-hfor:
			percept.HForeRunTemp = temp
		case temp = <-hrev:
			percept.HReverseRunTemp = temp
		case temp = <-wfor:
			percept.WForeRunTemp = temp
		case temp = <-wrev:
			percept.WReverseRunTemp = temp
		case temp = <-win:
			percept.WIntakeTemp = temp
		}
		if !temp.IsValid() {
			percept.Valid = false
		}
	}

	return
}

// Generates a system.Percept struct and sends pointer to it through the given
// updateChan channel. The functions throttles the required worker routines
// that generate sensor data by itself. Starts an infinite loop where percepts
// are generated and passed via updateChan. Workers are spawned or teminated
// according to runtime stats.
// @info make sure initW1() is called before and global sensorIds variable is
// set properly.
func pooledPerceptGenerator(updateChan chan *system.Percept)(){

	// channel through which TemperatureLookupJob are issued, all workers are sitting at
	// the other side of the channel awaiting lookup jobs
	var requestQueue chan w1.TemperatureLookupJob
	requestQueue = make(chan w1.TemperatureLookupJob)

	// channel through which the TemperatureLookupWorker workers send back the temperature
	// structs generated as response to issued TemperatureLookupJob jobs
	var responseQueue chan w1.Temperature
	responseQueue = make(chan w1.Temperature)

	// slice of chanels through which termination signals can be send
	// to TemperatureLookupWorker routines to coordinate worker pool size
	var pool []chan bool

	var flags map[string]*w1.Temperature
	var percept *system.Percept
	var currentWorkerPoolSize, benchPoolSize, benchPoolSize1 int
	var wokerPoolsStats map[int]*struct{
		n int
		mean float64
		deviation float64
	}


	//pool = make([]chan bool,W1_REPLICATION_LEVEL,W1_REPLICATION_LEVEL)
	pool = make([]chan bool,0)
	flags = make(map[string]*w1.Temperature,W1_SENSOR_COUNT)
	currentWorkerPoolSize = W1_REPLICATION_LEVEL
	benchPoolSize = 0
	benchPoolSize1 = 0
	wokerPoolsStats = make(map[int]*struct{n int; mean float64; deviation float64})

	// function literal that spawns one TemperatureLookupWorker and appends its
	// interrupt channel to the slice of interrupt channels
	spawn := func()(){
		interrupter := make(chan bool)
		go w1.LoopedTemperatureLookupWorker(
			requestQueue,
			responseQueue,
			interrupter,
			&logfile,
			&logmutex,
			)
		pool = append(pool, interrupter)
		return
	}

	// function literal that sends termination signal to the TemperatureLookupWorker
	// which is associated to the pool's last interrupt channel and updates the pool
	terminate := func()(){
		pool[len(pool)-1] <- true
		newpool := make([]chan bool,len(pool)-1)
		copy(newpool,pool[:len(pool)-1])
		pool = newpool
		return
	}

	// issues a TemperatureLookupJob for the given parameters to the requestQueue chanel
	lookup := func(sensorId,logic string){
		requestQueue <- w1.TemperatureLookupJob{sensorId,logic}
	}

	// initial spawn of TemperaturLookupWorker, starts currentWorkerPoolSize worker go routines
	for i:=0; i< currentWorkerPoolSize; i++ {
		spawn()
	}

	// main loop generates a new percept and sends pointer back to this methods callee through updateChan
	for {
		jobDone := false
		attempts := 0
		failures := 0

		// generate a new pointer
		percept = new(system.Percept)
		start := time.Now()

		// queue up jobs temperature lookups
		for logic,sensorId := range sensorIds {
			flags[logic]=nil // set the current temperature pointer in flags map to nil for this sencor

			// put a temperature lookup job for this sensor to the requestQueue
			//@todo use buffered requestQueue to prevent go routine spawning; drawback blocking if buffer is full
			go lookup(sensorId,logic)
			attempts++
		}

		// invariant: either valid data collected or lookup job for sensor is still running
		for !jobDone {
			// collect next response from workers
			temp := <-responseQueue

			//@todo: make sure no old values are accepted
			if temp.IsValid() {
				// update percept, set flag to temperature pointer or nil and update counter
				flags[temp.GetSensorLogic()]=SetTempPointerForSensor(percept,&temp)
				w1.IncrementSuccessLookupCount()
			} else {
				// update fail counter and reschedule temperature lookup for the corresponding sensor
				w1.IncrementFailLookupCount()
				failures++
				go lookup(temp.GetSensorId(),temp.GetSensorLogic())
				attempts++
			}

			// job is done if all pointer in flag map are non-nil
			jobDone = true
			for _,done := range flags {
				if done == nil {
					// since no valid data exists, do not terminate collection
					jobDone = false
				}
			}
		}

		// at this point all flags are set => all temperatures are valid
		finish := time.Now()
		percept.SetTime(finish)
		percept.Validate()

		updateChan <- percept

		jobDuration := finish.Sub(start)
		//fmt.Printf("Lookup Job took %2.4f seconds\n",jobDuration.Seconds())
		//fmt.Printf("Lookup Failure rate is %.3f\t%d\t%d\n\n",float64(failures)/float64(attempts),failures,attempts)
		//fmt.Printf("Result of Job: %s\n",percept)
		//evaluate if replication level needs to be increased

		// evaluation for system throttling and statistical tracking
		if stat,ok:=wokerPoolsStats[currentWorkerPoolSize]; ok {
			//update cummulative moving average (CMA)
			mean := ((*stat).mean * float64((*stat).n) + jobDuration.Seconds()) / float64((*stat).n + 1)
			deviation := ((*stat).deviation * math.Sqrt(float64((*stat).n)) + math.Abs(mean - jobDuration.Seconds())) / math.Sqrt(float64((*stat).n + 1))
			(*stat).mean = mean
			(*stat).deviation = deviation
			(*stat).n = (*stat).n + 1

			//fmt.Printf("current stats for cap %d - len %d - size %d are: %v\n",cap(pool),len(pool),workerPoolSize, *stat)

			// each 50 runs: take worker pool adjustment into consideration
			if (*stat).n % 50 == 0 {
				if benchPoolSize > 0 {
					oldStat,_ := wokerPoolsStats[benchPoolSize]
					if (*oldStat).mean > (*stat).mean {
						// currentWorkerPoolSize is better than benchmark
						update := 0
						if currentWorkerPoolSize == benchPoolSize - 1 {
							r := rand.New(rand.NewSource(int64((*stat).mean*100000.0)))
							update = r.Intn((*oldStat).n)
						}
						if update < 1000 {
							benchPoolSize1 = benchPoolSize
							benchPoolSize = currentWorkerPoolSize
							if currentWorkerPoolSize < WORKER_MAX_REPLICATION_LEVEL {
								currentWorkerPoolSize++
								spawn()
							}
						}
					} else {
						//benchmark was superior
						tmpStat,_ := wokerPoolsStats[benchPoolSize1]
						if (*tmpStat).mean > (*stat).mean {
							benchPoolSize = currentWorkerPoolSize
						} else {
							benchPoolSize = benchPoolSize1
							benchPoolSize1 = currentWorkerPoolSize
						}
						if currentWorkerPoolSize > 1 {
							currentWorkerPoolSize--
							terminate()
						}
					}
				} else {
					// spawn a new worker if benchmark has size 0
					benchPoolSize = currentWorkerPoolSize
					benchPoolSize1 = currentWorkerPoolSize
					currentWorkerPoolSize++
					spawn()
				}
			}
		} else {
			// no stats for workerPoolSize existing
			stat = &struct {
				n int
				mean float64
				deviation float64
			}{
				n:1,
				mean:jobDuration.Seconds(),
				deviation: 0.0,
			}
			wokerPoolsStats[currentWorkerPoolSize] = stat
		}

		// wait few seconds for next update
		time.Sleep(1*time.Second)
	}
}

func generateReward()(int){
	return 1
}

// Implementation of a system.RollOut type.
// Simply checks if burner and pumps are available
// and performs activation/deactivation as specified
// by the given system.Action parameter
func DefaultRollOut(a *system.Action)(){
	if burner == nil || boilerPump == nil || radiatorPump == nil {
		fmt.Println("Rollout not possible.")
		return
	}

	if triangle_switch != nil {
		triangle_switch.SetValue(a.GetTriangleState())
		<-time.After(time.Second * 5) // wait a moment for switch to adjust position
	}

	burnerWasOn := burner.GetValue()
	burnerIsOn := a.GetBurnerState()
	burner.SetValue(burnerIsOn)
	if !burnerWasOn && burnerIsOn {
		<-time.After(time.Second * 15) // wait a moment for switch to adjust position
	}

	if a.GetWPumpState() {
		boilerPump.Activate()
	} else {
		boilerPump.Deactivate()
	}

	if a.GetHPumpState() {
		radiatorPump.Activate()
	} else {
		radiatorPump.Deactivate()
	}

	return
}

func simple_routine(systemAgent agent.HeatingAgent, lastLog *time.Time)(break_loop bool){

	// generate percept; generate channel for receiving a percept pointer and hand it over to the oracle
	c := make(chan *system.Percept)

	// assume someone will wait on the Percept_request_chan in order to process this request
	system.Percept_request_chan <- c

	// wait for oracle to send percept back
	// @todo: ensure invariant that oracle sends back valid percepts only
	systemPercept := <- c
	defer func(){
		c = nil
	}()

	//fmt.Println("Fresh percept received by simple routine...")
	/*
	q := system.MakeDataRequest(
		make(chan []system.DataResponse),
		[]system.DataQuery{system.REVERSE_DELTA},
		[]struct{
			Sec int
			Weight float64
		}{
			{65,0.8},
			{120,0.6},
		},
	)
	system.Query_request_chan <- q
	response_silce := <- q.Endpoint
	fmt.Printf("Response received: %v\n",response_silce)
	fmt.Printf("Percept received: %s\n",systemPercept)
	*/


	var sPrimeState *system.ActorState
	next_action := systemAgent.GetAction(systemPercept)

	// security check
	if systemPercept.KettleTemp.GetValue() > 65000 {
		next_action.SetBurnerState(false)
		// return strong negative reward to agent
	}

	fmt.Println(next_action)

	if next_action != nil {
		// roll out action
		applyAction(next_action)

		// sState transition
		sPrimeState = sState.Successor(next_action).(*system.ActorState)
		sPrimeState.SetTimeStamp(systemPercept.CurrentTime)
	}

	// generate reward
	//systemReward := sState.Reward(sAction,sPrimeState)

	fmt.Println(sState)
	fmt.Println(sPrimeState)
	fmt.Println(sState.Equals(sPrimeState))

	// update sState field ensure invarient sState != nil holds
	if now := time.Now(); sPrimeState != nil && !sState.Equals(sPrimeState) {
		// state transition

		// insert old state and set lastLog to zero

		transitionLogTime := systemPercept.CurrentTime.Add(time.Second * -10)
		sStatePercept := *systemPercept
		sStatePercept.SetTime(transitionLogTime)
		sStatePercept.Insert()
		sState.SetTimeStamp(transitionLogTime)
		sState.Insert()

		systemPercept.Insert()
		sPrimeState.Insert()
		*lastLog = now
		//*lastLog = time.Time{}

		// update system state
		sState = sPrimeState
		agent.SetLastState(sState)

	} else if now.Sub(*lastLog).Seconds() > 180 {
		systemPercept.Insert()
		sPrimeState.Insert()
		*lastLog = now
	}

	return
}

func main(){

	var err error
	record := runtime.MemStats{}

	// set environment, loop through all environment variables until GOPATH is found
	// and set w1 and gpio paths
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if pair[0] == "GOPATH" {
			w1.SENSOR_PATH_PREFIX = pair[1]+"/src/github.com/hansen1101/go_heating/filesystem/sys/bus/w1/devices/"
			gpio.EXPORT_FILE = pair[1]+"/src/github.com/hansen1101/go_heating/filesystem/sys/class/gpio/export"
			gpio.UNEXPORT_FILE = pair[1]+"/src/github.com/hansen1101/go_heating/filesystem/sys/class/gpio/unexport"
			gpio.PATH_PREFIX = pair[1]+"/src/github.com/hansen1101/go_heating/filesystem/sys/class/gpio/gpio"
			config_path = pair[1]+"/src/github.com/hansen1101/go_heating/filesystem/heating_config/config.csv"
			log_path = pair[1]+"/src/github.com/hansen1101/go_heating/log/go_heating.log"
		}
	}

	// init gpio pins
	initGPIO()

	// push cleanup on defer stack
	defer func(){
		fmt.Println("Cleanup GPIO Pins.")
		cleanupGPIO()
	}()

	// create log file if not exists, otherwise open file for appending error logs
	logfile,err = os.OpenFile(log_path,os.O_WRONLY|os.O_APPEND|os.O_CREATE,os.ModePerm)
	//logfile = ioutil.Discard

	if err != nil {
		// could not create log file
		log.Fatal(err)
	}
	defer logfile.(*os.File).Close()

	// since the sensing is executing by multiple go routines concurrent access to log file must be secured
	logmutex = sync.Mutex{}

	// init w1 sensors and start temperature recording
	initW1()

	// Percept_Oracle starts pooledPerceptGenerator and two tight loops waiting for
	// Percept requests and Query request and the system chanels
	processChan := make(chan bool)
	go system.Percept_Oracle(
		pooledPerceptGenerator,
		PERCEPT_HISTORY_LENGTH,
		processChan,
		)
	<-processChan // wait for processing signal from system.Percept_Oracle routine

	go system.Configuration_Oracle(
		config_path,
		processChan,
		DEFAULT_MIN_BOILER_TEMP,
		)
	config_oracle_available := <-processChan // wait for processing signal from system.Percept_Oracle routine

	// @TODO include oracle loop
	//go system.Oracle_loop(fetchSensorData,PERCEPT_HISTORY_LENGTH )
	//system.Oracle_loop(fetchSensorData,PERCEPT_HISTORY_LENGTH )

	// init actors
	initActors()

	// introduce actuators to agent
	agent.SetPumpW(boilerPump)
	agent.SetPumpH(radiatorPump)
	agent.SetBurner(burner)

	sState = &system.ActorState{
		Time:time.Now(),
	}
	agent.SetLastState(sState)

	if !DEBUG {
		// establish a connection to a local database
		var db *sql.DB
		db, err = sql.Open(
			"mysql",
			fmt.Sprintf("%s:%s@unix(/var/run/mysqld/mysqld.sock)/%s?charset=utf8", DATABASE_USER, DATABASE_PASSWD, DATABASE_NAME),
		)

		if err != nil {
			// could not establish database connection
			log.Fatal(err)
		}

		// push database connection closing on defer stack to save resources
		defer func(){
			fmt.Println("Close db connection.")
			db.Close()
		}()

		// introduce database connection to system's logger package through which database access is encapsulated
		logger.SetDatabase(db,DATABASE_NAME)

		// @TODO: include all relations implementing logger.Logable interface that need to be logged to db here
		relations := []logger.Logable{
			sState,
			&system.Percept{},
		}

		logger.InitDbRelations(&relations)
	}
	// Set up and register channel to receive os signals for interrupting the process
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// set initial action
	sAction = system.NewAction(
		boilerPump.GetCurrentFreq(),
		radiatorPump.GetCurrentFreq(),
		boilerPump.IsActive(),
		radiatorPump.IsActive(),
		burner.GetValue(),
		triangle_switch.GetValue(),
		)

	// set rollout method for performing action transitions
	applyAction = DefaultRollOut

	systemAgent = agent.NewSimpleHeatingAgent(config_oracle_available)

	streamLearner := learner.NewWaterConsumptionLearner(
		5,
		float64(0.01),
		16, // p
		14, // sec
		6, // overlap
	)
	go streamLearner.StreamClustering(
		&logfile,
		&logmutex,
		)

	var lastLogTs time.Time

	//@debug deadlock bug fmt.Println("Starting the main Loop")
	loop:
	for {
		if simple_routine(systemAgent,&lastLogTs) {
			// simple routine signaled system shutdown
			break loop
		}

		runtime.ReadMemStats(&record)

		//time.Sleep(5*time.Second)
		select {
		case c := <-sigs:
			// process received an interrupt signal
			fmt.Printf("Signal: %v received. Now breaking the system loop and terminating...\n", c)
			break loop
		case <-time.After(time.Second * 5):
			//fmt.Print("no signal reached.\n")
			break
		}

		//@debug print memory info for debugging purposes
		/*
		fmt.Printf("Bytes alloced %d\tfree %d\treleased %d\nObjects alloced %d\nGC next %d\nStack inuse %d\nnumber of go routines: %d\n\n",
			record.HeapAlloc,
			record.Frees,
			record.HeapReleased,
			record.HeapObjects,
			record.LastGC,
			record.StackInuse,
			runtime.NumGoroutine(),
		)
		*/
		//@debug deadlock bug fmt.Println("Looping in the main Loop")
	}
}
