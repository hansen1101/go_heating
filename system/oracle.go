package system

import (
	"time"
	"fmt"
	"math"
	"sync"
	"errors"
	"os"
	"encoding/csv"
	"bufio"
	"strconv"
)

const (
	BOILER_DELTA DataQuery = iota
	REVERSE_DELTA
	WATER_BUFFER_DELTA
)

type DataQuery int
//type PerceptGenerator func(timestamp *time.Time) (percept *Percept)

type DataResponse struct {
	Id		DataQuery
	Result          float64
	Considered_data int
	TimeStamp	int64
}

// Generator function for DataResponse objects.
func generateResponse(query DataQuery, result float64, dataAmount int, timestamp int64) (DataResponse) {
	return DataResponse{Id:query, Result:result, Considered_data:dataAmount, TimeStamp:timestamp}
}

type Calculation_info struct {
	Sec    int
	Weight float64
}

// A dataRequest contains an endpoint channel where some routine is listening vor a DataResponse for this query.
// The DataQuery slice contains all query flags the request's sender is expecting to receive back.
// Additionally calc_info is handed over for the calculations.
type dataRequest struct {
	Endpoint  chan []DataResponse	// channel for receiving the answer
	request   []DataQuery		// slice of query flags
	calc_info []struct{
		Sec int
		Weight float64
	}				//slice of calc_info []Calculation_info
}

type configRequest struct {
	Endpoint chan int
	percept *Percept
}

type ConfigRequester func(chan int, *Percept)()

var (
	//window []*Percept			// sliding window to store the percept history
	Percept_request_chan chan chan *Percept // channel through which Percept pointer channels can be passed
	Query_request_chan chan *dataRequest 	// channel through which dataRequest could be passed
	Percept_update_chan chan Percept
	//Configuration_update_chan chan Target @TODO
	Configuration_request_chan chan *configRequest
)

func MakeConfigRequest(endpoint chan int, percept *Percept)(){
	if endpoint != nil && Configuration_request_chan != nil {
		Configuration_request_chan <- &configRequest{endpoint,percept}
	} else {
		fmt.Println("[ERROR]\teither channel endpoint %v or request %v does not exists",endpoint,Configuration_request_chan)
	}
}

// Interface function to generate a dataRequest object.
// @param ep channel for the response to be send through
// @param req array of DataQuery objects that need to be calculated
// @param info array of Calculation_info objects where calculation parameters are specified
func MakeDataRequest(ep chan []DataResponse, req []DataQuery, info []struct{
	Sec int
	Weight float64
}) (*dataRequest) {
	return &dataRequest{ep, req, info}
}


// Updates the sliding window by inserting the given percept in a slot index by percepts time in seconds modulo window length.
// The method also ensures that the old data between the currentIndex pointer and the new index is deleted from the window.
// After the function call the currentIndex points to the newly inserted Percept and the counter is up to date.
func updateSlidingWindow(window *([]*Percept), percept *Percept, currentIndex, counter *int)(){
	nextIndex := int(percept.CurrentTime.Unix() % int64(cap(*window)))
	if *currentIndex >= 0 {
		if (*window)[*currentIndex] != nil {
			// case: the slot is not empty
			if percept.CurrentTime.Unix() - (*window)[*currentIndex].CurrentTime.Unix() >= int64(cap(*window)) {
				// case: last percept that was inserted is older than windowLength seconds --> fully wrap around
				*window = make([]*Percept, cap(*window), cap(*window))
				*counter = 0
				//fmt.Printf("History reset %v %v\n", t, (*window)[*currentIndex].CurrentTime)
			} else {
				// case: last entry is not very old --> clear all entries between last entry and new position
				switch {
				case nextIndex - *currentIndex > 0:
					// nextIndex is between current and slice end -> loop to position and clear every slot in between
					for i := 1; i < nextIndex - *currentIndex; i++ {
						if (*window)[*currentIndex + i] != nil {
							(*window)[*currentIndex + i] = nil
							*counter--
						}
					}
				case nextIndex - *currentIndex < 0:
					// we are facing a wrap around
					for i := *currentIndex + 1; i < len((*window)); i++ {
						if (*window)[i] != nil {
							(*window)[i] = nil
							*counter--
						}
					}
					for i := 0; i < nextIndex; i++ {
						if (*window)[i] != nil {
							(*window)[i] = nil
							*counter--
						}
					}
				case nextIndex - *currentIndex == 0:
				//fmt.Println("Newer data available")
				}
			}
		}
		if (*window)[nextIndex] == nil {
			// only increment counter if slot is not occupied
			*counter++
		}
	}
	(*window)[nextIndex] = percept
	*currentIndex = nextIndex
}

// The oracle maintains a sliding window and in order to perform data queries on the system's history.
// The main oracle loop sets up the interface channels for interactions with this go routine and
// starts an infinity loop where it (1.) generates a Percept and updates the sliding window.
// After the window is updated the interface channels are checked in order to perform calculations and
// response to requests (from e.g. agents).
// @param generator a PerceptGenerator function that generates a Percept object
// @param windowLength the length of the sliding window
func Oracle_loop(generator PerceptGenerator, windowLength int) () {

	var currentIndex, nextIndex, counter int
	var systemPercept *Percept

	window := make([]*Percept, windowLength, windowLength)
	currentIndex = 0
	counter = 0	// stores the number of percepts stored in the window

	Percept_request_chan = make(chan chan *Percept)
	Query_request_chan = make(chan *dataRequest)

	for {
		// generate percept
		t := time.Now()
		// fire the fetchSensorDataRoutines in a blocking way
		systemPercept = generator(&t)

		if systemPercept != nil {
			//fmt.Printf("Next percept: %v\n",systemPercept)
			if systemPercept.Valid == false {
				//@todo handle invalid case
				fmt.Print("Percept is not valid.\n")
				// process with currentIndex and do not update the sliding window
			} else {
				// update the sliding window
				// index of next slot corresponds to second reduction
				nextIndex = int(t.Unix() % int64(windowLength))
				if window[currentIndex] != nil {
					// case: the slot is not empty
					if t.Unix() - window[currentIndex].CurrentTime.Unix() >= int64(windowLength) {
						// case: last percept that was inserted is older than windowLength seconds --> fully wrap around
						window = make([]*Percept, windowLength, windowLength)
						counter = 0
						//fmt.Printf("History reset %v %v\n", t, window[currentIndex].CurrentTime)
					} else {
						// case: last entry is not very old --> clear all entries between last entry and new position
						switch {
						case nextIndex - currentIndex > 0:
							// nextIndex is between current and slice end -> loop to position and clear every slot in between
							for i := 1; i < nextIndex - currentIndex; i++ {
								if window[currentIndex + i] != nil {
									window[currentIndex + i] = nil
									counter--
								}
							}
						case nextIndex - currentIndex < 0:
							// we are facing a wrap around
							for i := currentIndex + 1; i < len(window); i++ {
								if window[i] != nil {
									window[i] = nil
									counter--
								}
							}
							for i := 0; i < nextIndex; i++ {
								if window[i] != nil {
									window[i] = nil
									counter--
								}
							}
						case nextIndex - currentIndex == 0:
						//fmt.Println("Newer data available")
						}
					}
				}
				if window[nextIndex] == nil {
					// only increment counter if slot is not occupied
					counter++
				}
				window[nextIndex] = systemPercept
				currentIndex = nextIndex
			}
		}



		// wait for 250ms to perform oracle duties
		select {
		case percept_chan := <-Percept_request_chan:
			// a request for a percept is received through request channel --> handle over current percept
			percept_chan <- window[currentIndex]
		case result_chan := <-Query_request_chan:
			// a dataRequest was received, open the request and answer back to the endpoint
			res := make([]DataResponse, 0, 0)
			for _,req := range result_chan.request {
				for _, j := range result_chan.calc_info {
					if j.Sec > 0 {
						var data func(i int)(int,int64,error)
						switch req {
						case BOILER_DELTA:
							data = func(i int)(int,int64,error){
								return window[i].BoilerMidTemp.GetValue(),window[i].CurrentTime.Unix(),nil
							}
						case REVERSE_DELTA:
							data = func(i int)(int,int64,error){
								return window[i].HReverseRunTemp.GetValue(),window[i].CurrentTime.Unix(),nil
							}
						default:
							data = func(i int)(int,int64,error){
								return window[i].HReverseRunTemp.GetValue(),window[i].CurrentTime.Unix(),nil
							}
						}
						res = append(
							res,
							calculateTempDelta(
								&window,
								j.Weight,
								j.Sec,
								currentIndex,
								req,
								data,
								expWeightedMovingAverage,
								//naiveMean,
							),
						)
					}
				}
			}
			result_chan.Endpoint <- res
		case <-time.After(10 * time.Second):
		//case <-time.After(250 * time.Millisecond):
			// no duties to perform
			fmt.Println("Oracle timed out.")
		}
	}
	return
}

// A service routine that establishes a percept sliding window and processes
// incoming requests from clients (either percept request by main routine or
// query requests from learners).
// The service listens at two distinct channels for either type of request.
// In order to establish the sliding window the service starts a generator and
// updates the sliding window each time the generator hands over a new Percept.
func Percept_Oracle(generator func(chan *Percept)(),windowLength int,processChan chan bool){
	var slidingWindow []*Percept // history of percepts generated
	var currentIndex, counter int // counts of percepts in history and pointer to latest percept
	var windowLock sync.Mutex // lock for the sliding window
	var perceptUpdateChan chan *Percept // channel throug which the generator delivers pointers to currently generated Percept structs

	slidingWindow = make([]*Percept, windowLength, windowLength)
	currentIndex = 0
	counter = 0
	perceptUpdateChan = make(chan *Percept)

	// init package variables
	Percept_request_chan = make(chan chan *Percept)
	Query_request_chan = make(chan *dataRequest)
	//processChan <- true

	// start percept generator routine;
	// generator ensures only valid percepts are send through perceptUpdateChan
	go generator(perceptUpdateChan)

	// spawn sliding window update routine;
	// receives percepts and updates sliding window securely
	go func()(){
		var currentPercept *Percept
		for {
			// wait until a the generator function hands over a new percept pointer
			currentPercept = <-perceptUpdateChan
			if currentPercept != nil && currentPercept.IsValid() {
				windowLock.Lock()
				//@debug deadlock bug fmt.Print("Locked by Update generator...")
				updateSlidingWindow(&slidingWindow,currentPercept,&currentIndex,&counter)
				//@debug deadlock bug fmt.Print(" ...released by Update generator\n")
				windowLock.Unlock()
			}
		}
	}()

	// spawn service routine;
	// either receives update or query request and
	go func()(){
		for {
			//@debug deadlock bug fmt.Println("Waiting for a query signal")
			select {
			case percept_chan := <-Percept_request_chan:
				// a request for a percept pointer is received through request channel --> hand over current percept pointer
				//@debug deadlock bug fmt.Print("Percept Request processor tries to lock...")
					var currentPercept *Percept
					for currentPercept == nil {
						windowLock.Lock()
						currentPercept = slidingWindow[currentIndex]
						//@debug deadlock bug fmt.Print(" ...released by Percept Request processor\n")
						windowLock.Unlock()
						if currentPercept == nil {
							// sleep and hope for sliding window updates
							fmt.Println("Percept request could not be handled since sliding window is empty")
							<-time.After(time.Second * 5)
						}
					}
					percept_chan <- currentPercept
			case result_chan := <-Query_request_chan:
				// a dataRequest was received, open the request and answer back to the endpoint
				resp := make([]DataResponse, 0, 0)
				for _,req := range result_chan.request {
					for _, j := range result_chan.calc_info {
						if j.Sec > 0 {
							var data func(i int) (int, int64, error)
							var meanAlgorithm meanCalc
							switch req {
							case WATER_BUFFER_DELTA:
								data = func(i int) (tempData int, dataTimestamp int64, err error) {
									if slidingWindow[i] == nil {
										err = errors.New("No data item in slot found")
										return
									}
									err = nil
									tempData = slidingWindow[i].BoilerTopTemp.GetValue()
									dataTimestamp = slidingWindow[i].CurrentTime.Unix()
									return
								}
								meanAlgorithm = totalDeltaInterval
							case BOILER_DELTA:
								data = func(i int) (int, int64, error) {
									return slidingWindow[i].BoilerMidTemp.GetValue(), slidingWindow[i].CurrentTime.Unix(), nil
								}
								meanAlgorithm = expWeightedMovingAverage
							case REVERSE_DELTA:
								data = func(i int) (int, int64, error) {
									return slidingWindow[i].HReverseRunTemp.GetValue(), slidingWindow[i].CurrentTime.Unix(), nil
								}
							default:
								data = func(i int) (int, int64, error) {
									return slidingWindow[i].HReverseRunTemp.GetValue(), slidingWindow[i].CurrentTime.Unix(), nil
								}
								meanAlgorithm = naiveMean
							}
							//@debug deadlock bug fmt.Print("Data Request processor tries to lock...")
							windowLock.Lock()
							resp = append(
								resp,
								calculateTempDelta(
									&slidingWindow,
									j.Weight,
									j.Sec,
									currentIndex,
									req,
									data,
									meanAlgorithm,
								),
							)
							//@debug deadlock bug fmt.Print(" ...released by Data Request processor\n")
							windowLock.Unlock()
						}
					}
				}
				// send response back to the endpoint associated with dataRequest
				result_chan.Endpoint <- resp
			}
			//@debug deadlock bug fmt.Println("Query signal processed")
		}
	}()

	defer func(){
		fmt.Println("When you see this line, all oracle routines have been started!")
		// @important: send back signal after package variabels are initialized in order to signal callee that processing is possible
		processChan <- true
	}()
}

// Generates a configuration lookup map
func generate_Configuration(path string)(configuration map[int][]int,ok bool){
	// open file at path and build table
	configuration = make(map[int][]int)
	ok = false
	file, err := os.Open(path)
	if err != nil {
		return
	}
	fileinfo,_ := file.Stat()
	if fileinfo.IsDir() {
		return
	}
	r := csv.NewReader(bufio.NewReader(file))
	records, err_read := r.ReadAll()
	if err_read == nil{
		//parse records
		for row_id,record := range records {
			var key_tmp int
			if row_id > 0 {
				for col_id,temp := range record {
					temp_value,convert_err := strconv.Atoi(temp)
					hour := col_id-1
					if convert_err != nil {
						fmt.Println("conversion failed")
					}
					if col_id == 0 {
						key_tmp = temp_value
						_,ok := configuration[key_tmp]
						if !ok {
							configuration[key_tmp] = make([]int, 24, 24)
						}
					} else {
						configuration[key_tmp][hour] = temp_value
					}
				}
			}
		}
	}
	ok = true
	return
}

func Configuration_Oracle(path string,processChan chan bool,default_target int)(){
	//var windowLock sync.Mutex // lock for the sliding window
	var target int
	var configurationLock sync.Mutex // lock for the sliding window
	var configuration map[int][]int
	var ok bool
	configurationLock.Lock()
	configuration,ok = generate_Configuration(path)
	configurationLock.Unlock()
	Configuration_request_chan = make(chan *configRequest)

	defer func(){
		fmt.Printf("[OK: %v] When you see this line, all configuration oracle has been started!\n",ok)
		// @important: send back signal after package variabels are initialized in order to signal callee that processing is possible
		processChan <- ok
	}()

	if ok {
		go func()(){
			for {
				select {
					case config_request := <-Configuration_request_chan:
						if config_request.percept.IsValid(){
							temp_key := config_request.percept.OutsideTemp.GetValue()
							temp_key = int(math.Round(float64(temp_key)/1000.0))
							hour_index := config_request.percept.CurrentTime.Hour()
							configurationLock.Lock()
							if _,key_exist := configuration[temp_key]; key_exist {
								target = configuration[temp_key][hour_index]
							} else {
								target = default_target
							}
							configurationLock.Unlock()
						}
						config_request.Endpoint <- target
				}
			}
		}()
		go func()(){
			for {
				<-time.After(time.Minute * 5)
				update_configuration,valid := generate_Configuration(path)
				if valid {
					configurationLock.Lock()
					configuration = update_configuration
					configurationLock.Unlock()
				}
			}
		}()
	}
}

// Ensures sec value is in the bounds of the window
func alignBound(window *([]*Percept), sec *int) () {
	if *sec >= len(*window) {
		*sec = len(*window) - 1
	} else if *sec < 0 {
		*sec = 0
	}
	return
}

// Ensures weight is a valid value in the interval between 0.0 and 1.0
func checkAndValidate(weight *float64)(){
	if *weight < 0.0 {
		*weight = 0.0
	} else if *weight > 1.0 {
		*weight = 1.0
	}
	return
}

type meanCalc func(currentTemp int, currentTime int64, lastTemp *int, lastTime *int64, meanValue *float64, weight float64, intervallLength int)()

// A naive MeanCalc for testing purposes only.
// @TODO delete this function in final version.
func naiveMean(i int, j int64, k *int, l *int64, meanValue *float64, w float64, intervallLength int)(){
	*meanValue = 6.3
}

func updateBoilerMean(window *([]*Percept), i int, lastBoilerTemp *int, lastTimeStamp *int64, meanValue *float64) () {
	var currentBoilerTemp int
	var currentTimeStamp int64
	if (*window)[i] != nil {
		currentBoilerTemp = (*window)[i].BoilerMidTemp.GetValue()
		currentTimeStamp = (*window)[i].CurrentTime.Unix()
		if *lastBoilerTemp > math.MinInt32 {
			// case: this is not the first element
			if delta := currentBoilerTemp - *lastBoilerTemp; delta != 0 {
				*meanValue = .85 * *meanValue + 1.15 * (float64(delta) / float64(currentTimeStamp - *lastTimeStamp))
				*meanValue *= .5
			}
		}
		*lastBoilerTemp = currentBoilerTemp
		*lastTimeStamp = currentTimeStamp
	}
}

// A function is of type MeanCalc. Updates the exponentially weighted moving average value by calculating the current delta and
// adding it by a weighted summation. After update calculation the last values are stored to pointer
// fields in order to ensure that successive function calls can access them.
// @param currentTemp latest temperature value
// @param currentTime timestamp for latest temperature value
// @param lastTemp pointer to field where last temperature value is stored
// @param lastTime pointer to field where timestamp of last temperature value is stored
// @param meanValue the field where the overall mean value is stored
// @param weight value for the decay/weight component of the calculation
func expWeightedMovingAverage(currentTemp int, currentTime int64, lastTemp *int, lastTimeStamp *int64, meanValue *float64, weight float64, intervallLength int) () {
	if *lastTimeStamp > 0 {
		// case: update calculation is only performed if lastTemp and lastTime values are already initialized
		if delta := currentTemp - *lastTemp; delta != 0 {
			old_mean := float64(delta) / float64(currentTime - *lastTimeStamp)
			*meanValue = weight * old_mean + (1.0 - weight) * *meanValue
		}
	}
	*lastTemp = currentTemp
	*lastTimeStamp = currentTime
	return
}

func totalDeltaInterval(currentTemp int, currentTime int64, lastTemp *int, lastTimeStamp *int64, meanValue *float64, weight float64, intervallLength int) () {
	if *lastTimeStamp == 0 {
		*lastTemp = currentTemp
		*lastTimeStamp = currentTime
		*meanValue = float64(0)
	} else {
		if delta := currentTemp - *lastTemp; delta != 0 {
			//*meanValue += float64(delta) * float64(currentTime - *lastTimeStamp) / float64(intervallLength)
			*meanValue += float64(delta)
			*lastTemp = currentTemp
			*lastTimeStamp = currentTime
		}
	}
}

// Performs a temperature development calculation by executing the given meanCalc literal on the last sec entries in the sliding window.
// Afterwards a DataResponse for the specified DataQuery flag is generated and returned.
// The function to fetch the temperature and time data handed over in the value literal.
// @param weight factor must be within (0.0,1.0]; the greater the value the more weight is assigned to the latest value during mean updates
// @param sec number of seconds that should be taken into during calculation
// @param currentIndex the index pointing to the head (newest) value in the window
// @param flag DataQuery to build the response for
// @param value func(index)(temperature,time) a function that describes how to get the temperature data und time for the ith element
// @param meanCalc literal indicates the function that should be used to calculate the mean value
func calculateTempDelta(window *([]*Percept), weight float64, sec, currentIndex int, flag DataQuery, value func(int)(int,int64,error), mean meanCalc) (result DataResponse) {

	var lastTemp int // value of last boiler temp
	var lastTimeStamp int64 // value of last timestamp
	var meanValue float64 // mean value from which the result is generated

	// make sure sec value is in bounds
	alignBound(window,&sec)

	checkAndValidate(&weight)

	var firstIndex int
	firstIndex = -1

	update := func (i int)(){
		temp,timestamp,err := value(i)
		if err == nil {
			if firstIndex < 0 {
				firstIndex = i
			}
			mean(
				temp,
				timestamp,
				&lastTemp,
				&lastTimeStamp,
				&meanValue,
				weight,
				sec,
			)
		}
	}

	switch {
	case currentIndex - sec < 0:
		// array overflow; loop from first position after currentIndex to last slot in slice
		for i := cap(*window) - (sec - currentIndex); i < cap(*window); i++ {
			update(i)
		}
		// add a second loop from fist slot in slice up to currentIndex
		for i := 0; i <= currentIndex; i++ {
			update(i)
		}
	default:
		for i := currentIndex - sec; i <= currentIndex; i++ {
			update(i)
		}
	}

	var intervalLength int
	//fmt.Printf("%.2f\tFirst Index: %d; Last Index: %d\n",meanValue,firstIndex,currentIndex)
	if firstIndex < 0 {
		// no data items tracked
		fmt.Println("No items found for mean calculation.")
		lastTimeStamp = math.MaxInt64
		intervalLength = 0
	} else {
		if firstIndex <= currentIndex {
			intervalLength = currentIndex - firstIndex + 1
		} else {
			intervalLength = currentIndex + cap(*window) - firstIndex + 1
		}
	}

	result = generateResponse(flag, meanValue, intervalLength, lastTimeStamp)
	return
}
