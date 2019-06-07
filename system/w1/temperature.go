package w1

import(
	"os"
	"io"
	"time"
	"strings"
	"regexp"
	"strconv"
	"errors"
	"sync"
	"math"
)

const (
	SENSOR_PATH_SUFFIX = "/w1_slave"
)

var (
	SENSOR_PATH_PREFIX = "/sys/bus/w1/devices/"		// this needs to be variable in order to enable the main prgramm to set the w1 device path
	replica_timeout_seconds time.Duration = 15		// timeout in seconds for the replicated TemperatureLookup
	successGenerations, failGenerations int
	statsMutex sync.RWMutex
)

type TemperatureLookupJob struct {
	SensorId, SensorLogic string
	//responseChan chan(Temperature)
	//id int
}
type LookupStat struct {
	Total int
	Success int
	Failed int
}
type TemperatureLookupWorker func(routine TemperatureLookup, request chan(TemperatureLookupJob), interrupt chan(bool))()

type Temperature struct {
	sensor       string
	system_logic string
	value        int	// tempreature value
	valid        bool	// validation flag
}

/**
 * A function that returns a Temperature struct.
 */
type TemperatureLookup func()(Temperature)

func NewTemperature(sensorId, logic string)(*Temperature){
	return &Temperature{
		sensor:sensorId,
		system_logic:logic,
		value:0,
		valid:false,
	}
}

func (t *Temperature) GetValue()(int){
	return t.value
}

func (t *Temperature) AddValue(delta int)(){
	t.value += delta
	return
}

func (t *Temperature) IsValid()(bool){
	return t.valid
}

func (t *Temperature) GetSensorId()(string){
	return t.sensor
}
func (t *Temperature) GetSensorLogic()(string){
	return t.system_logic
}

func (t *Temperature) SetLogic(logic string)(){
	t.system_logic = logic
}

/**
 * @TODO check if logDestination needs to be protected against concurrent access
 */
func logError(sensorId *string, err *error,logDestination *io.Writer, mutex *sync.Mutex, data *[]byte){
	mutex.Lock()
	if *err != nil {
		error_msg := strings.NewReader("[ERROR]\t"+time.Now().String()+"\t["+*sensorId+"]\t"+(*err).Error()+"\n")
		io.Copy(*logDestination, error_msg)
	}
	if data != nil {
		if len(*data) > 0 {
			io.Copy(*logDestination, strings.NewReader(string(*data)))
		}
	}
	mutex.Unlock()
	return
}

/**
 * @TODO check if logDestination needs to be protected against concurrent access by lock or semapohre
 */
func logWarning(sensorId *string, str *string,logDestination *io.Writer, mutex *sync.Mutex,){
	mutex.Lock()
	if len(*str) > 0 {
		error_msg := strings.NewReader("[WARNING]\t"+time.Now().String()+"\t["+*sensorId+"]\t"+*str+"\n")
		io.Copy(*logDestination, error_msg)
	}
	mutex.Unlock()
	return
}

/**
 * Helper function extracts the temperature data from a given file. Returns an error if the file could not be
 * parsed in the expected way. If temperature extraction succeeds the error references a nil pointer.
 * @return the temperature value that could be extracted from the file
 * @return the error that occured during extraction
 */
func extractTempDataFromFile(sensorId *string, file *os.File, logDestination *io.Writer, logMutex *sync.Mutex, buf *[]byte)(temp_value int, err error){
	var n int

	n,err = io.ReadFull(file, *buf)

	if err == nil && n == 0 {
		err = errors.New("File could not be read into the buffer")
	}

	if err != nil {
		// if err is ErrUnexpectedEOF fewer bytes than len(buf) were read -> continue as normal
		if err==io.ErrUnexpectedEOF {
			err = nil
		} else {
			return
		}
	}

	// check CRC flag in sensor's data file
	crc_valid,_ := regexp.Match("[^:]+( : crc=)[a-z0-9]{2}( YES\n)",*buf)
	//crc_check := strings.Contains(string(*buf), "YES")
	if !crc_valid {
		err = errors.New("CRC check failed, YES flag could not be found.")
		return
	}

	// get beginning index of second line
	line_break_pattern := regexp.MustCompile("\n")
	line_break_matches := line_break_pattern.FindAllIndex(*buf,-1)
	second_line := -1
	if line_break_matches != nil {
		if len(line_break_matches) == 2 {
			// file contains 2 line breaks
			second_line = line_break_matches[0][0]
		} else {
			// file format not ok
			warning := "Second line break in file w1_slave missing."
			logWarning(sensorId,&warning,logDestination,logMutex)
		}
	} else {
		err = errors.New("No match for second line in temp data file found.")
		return
	}

	// extract temperature value from second line
	//temp_pattern := regexp.MustCompile("[^(t=) \n][-?0-9]{3,}[^\n]?")
	temp_pattern := regexp.MustCompile("t=(-?[0-9]+)")
	match_results:= temp_pattern.FindAllStringSubmatch(string((*buf)[second_line+1:]),-1)
	if len(match_results) == 0 {
		// no value found by regex
		err = errors.New("No temperature value found by regex.")
		return
	} else {
		if len(match_results) > 1 {
			// more than one result found
			warning := "More than 1 result extracted by single request."
			logWarning(sensorId,&warning,logDestination,logMutex)
		}
		// try extract first match
		temp_value,err = strconv.Atoi(match_results[0][1])
	}
	return
}

/**
 * Initializes a temperature sensor by implementing the TemperatureLookup function
 * @return TemperatureLookup function which creates a complete Temperatur object from the sensor
 */
func Init_Sensor(sensorId string,logDestination *io.Writer, logMutex *sync.Mutex) TemperatureLookup {
	// implementation of a TemperatureLookup, creates a new Temperature, opens the sensor's data file,
	// extracts the temperature value and sets the corresponding fields in the Temperature struct
	text := "initialization of the sensor."
	logWarning(&sensorId, &text, logDestination,logMutex)

	// parses and validates the temperature data for this sensor object
	return func() (data Temperature) {
		data = Temperature{sensor:sensorId}
		var err error
		var file *os.File
		var temp_value int
		buf := make([]byte, 128, 128) //@todo evaluate whether this can be placed globally

		// opens the sensor's data file in read only mode
		sensorpath := SENSOR_PATH_PREFIX+sensorId+SENSOR_PATH_SUFFIX

		//@TODO maybe this should be protected against mutual access -> should not be a problem ;)
		file, err = os.OpenFile(sensorpath, os.O_RDONLY, os.ModeTemporary)
		defer func(){
			//fmt.Printf("Close sensor file at %s\n",sensorpath)
			buf = nil
			file.Close()
		}()

		if err != nil {
			// sensor file could not be opened properly, log error
			logError(&sensorId,&err,logDestination,logMutex,nil)
			return
		}

		// sensor file opened successfully, extract temp data
		temp_value,err = extractTempDataFromFile(&sensorId,file,logDestination,logMutex,&buf)

		if err == nil {
			// data could be extracted without error check if extracted data is valid
			if temp_value > 120000 || temp_value < -60000 {
				err = errors.New("Temperature sensor data corrupted due to electromagnetic interference.")
			} else {
				if temp_value == 85000 {
					// warning, sensor maybe not working correctly
					warning := "Sensor might not be working properly."
					logWarning(&sensorId, &warning, logDestination,logMutex)
				}
				data.valid = true
				data.value = temp_value
				return
			}
		}

		if err != nil {
			logError(&sensorId,&err,logDestination,logMutex,&buf)
		}
		return
	}
}

/**
 * The replicated TemperatureLookup creates a chanel, spawns the replicated TemperatureLookups and
 * returns the first valid Temperature struct that comes in from the chanel.
 * If TemperatureLookups time out, empty Temperature struct is returned.
 * @param replicas slice of TemperaturLookup functions
 * @return pointer to a valid or an empty Temperature
 */
func First(replicas ...TemperatureLookup) (*Temperature) {

	// chanel for passing temperature data from spawned go routines
	c := make(chan Temperature)

	// takes an index, fetches the corresponding TemperatureLookup replica,
	// checks if replica produces valid temperature and passes it through the chanel
	fetchValidTemperatur := func(i int){
		temp := replicas[i]()
		if temp.valid {
			select{
			case <- time.After(replica_timeout_seconds*time.Second):
				break
			case c <- temp:
				break
			}
		}
		return
	}

	// spawn a go routine for each replica
	for i,_ := range replicas {
		go fetchValidTemperatur(i)
	}

	select {
		case <- time.After(replica_timeout_seconds*time.Second):
			return &Temperature{}
		case temperature := <- c :
			return &temperature
	}
}

// implements TemperaturLookup since parses and validates the temperature data for this sensor object
func SensorTemperaturGenerator(sensorId, sensorLogic string,logDestination *io.Writer, logMutex *sync.Mutex) (data Temperature) {
	data = Temperature{sensor:sensorId,system_logic:sensorLogic}
	var err error
	var file *os.File
	var temp_value int
	buf := make([]byte, 128, 128) //@todo evaluate whether this can be placed globally

	// opens the sensor's data file in read only mode
	sensorpath := SENSOR_PATH_PREFIX+sensorId+SENSOR_PATH_SUFFIX

	//@TODO maybe this should be protected against mutual access -> should not be a problem ;)
	file, err = os.OpenFile(sensorpath, os.O_RDONLY, os.ModeTemporary)
	defer func(){
		//fmt.Printf("Close sensor file at %s\n",sensorpath)
		buf = nil
		file.Close()
	}()

	if err != nil {
		// sensor file could not be opened properly, log error
		logError(&sensorId,&err,logDestination,logMutex,nil)
		return
	}

	// sensor file opened successfully, extract temp data
	temp_value,err = extractTempDataFromFile(&sensorId,file,logDestination,logMutex,&buf)

	if err == nil {
		// data could be extracted without error check if extracted data is valid
		if temp_value > 120000 || temp_value < -60000 {
			err = errors.New("Temperature sensor data corrupted due to electromagnetic interference.")
		} else {
			if temp_value == 85000 {
				// warning, sensor maybe not working correctly
				warning := "Sensor might not be working properly."
				logWarning(&sensorId, &warning, logDestination,logMutex)
			}
			data.valid = true
			data.value = temp_value
			return
		}
	}

	if err != nil {
		logError(&sensorId,&err,logDestination,logMutex,&buf)
	}
	return
}

// implements simple TemperatureLookupWorker
func LoopedTemperatureLookupWorker(request chan(TemperatureLookupJob), responseChan chan(Temperature), interrupt chan(bool), logDestination *io.Writer, logMutex *sync.Mutex){
	loop:
	for {
		select {
		case job:=<-request:
			responseChan <- SensorTemperaturGenerator(job.SensorId,job.SensorLogic,logDestination,logMutex)
		case <-interrupt:
			//termination signal received
			break loop
		}
	}
	return
}

func IncrementSuccessLookupCount(){
	statsMutex.Lock()
	if math.MaxInt32 == successGenerations {
		successGenerations -= failGenerations
		failGenerations = 0
	}
	successGenerations++
	statsMutex.Unlock()
}

func IncrementFailLookupCount(){
	statsMutex.Lock()
	if math.MaxInt32 == failGenerations {
		failGenerations -= successGenerations
		successGenerations = 0
	}
	failGenerations++
	statsMutex.Unlock()
}

func GetLookupStats()(stats LookupStat){
	stats = LookupStat{}
	statsMutex.RLock()
	stats.Failed = failGenerations
	stats.Success = successGenerations
	stats.Total = stats.Failed + stats.Success
	statsMutex.RUnlock()
	return
}
