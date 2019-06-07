package agent

import (
	"time"
	"math"
	"fmt"
	"container/list"
	"github.com/hansen1101/go_heating/system"
	"github.com/hansen1101/go_heating/system/logger"
)

const(
	STATE_w0_h0 StateID = 0
	STATE_w1_h0 StateID = 4
	STATE_w0_h1 StateID = 2
	STATE_w1_h1 StateID = 8
	ADP_STATE_TABLE = "adp_states"
)

var(
	seqHistory = list.New()
	sequences = make(map[int][]*(list.List))
	states = make(map[string]int)
	known_states = make(map[*ReflexState]int)
	base_delta_transitions = make(map[string]map[int]int)
	base_potential_transitions = make(map[int]map[int]int)
	delta_transitions = make(map[string]map[int]int)
	potential_transitions = make(map[int]map[int]int)
	total_count = 0
	transitionTimeStamp time.Time
	w_lower_bound = 49000
	w_target_bound = 52000
	h_deviation = 11000
	circulation_interval = 10.0		// interval used to calculate the circulation value
	circulatin_interval_count = 6
	reverse_heating_delta_change_count = 5
	Potential_Count = [1000][1000]int{}
	Test = "String"
	debug_system_action *system.Action
	action_record = make(map[ReflexStateHash]*system.Action)
)

type StateID uint8
type Model map[ReflexStateActionHash]map[ReflexStateHash]float64

type RewardFunction map[ReflexStateHash]system.Reward

type UtilityFunction map[ReflexStateHash]float64
type ActionPicker func(*system.Percept,system.Reward)(action *system.Action,log bool)
type StateGenerator func(*system.Percept)(AiState)
type HashLookupTable map[ReflexStateHash]AiState
type CountTable map[ReflexStateHash]int
type QStateTable map[ReflexStateActionHash]int
type TransitionTable map[ReflexStateActionHash]map[ReflexStateHash]int

type ReflexAgent struct {
	Agent
	seqBeginningState 	AiState
	previousState     	AiState
	previousAction 		*ReflexAction
	previousPercept		*system.Percept
	//previousPerceptTime	time.Time

	boilerStableDuration	float64

	transitionDuration 	time.Duration
	lastWEnergyLoad  	time.Time

	state_counter		CountTable
	state_hash_lookup	HashLookupTable
	policy			Policy
	utilities		UtilityFunction
	n_sa			QStateTable
	n_sPrime_sa		TransitionTable
	discount_factor		float64
	transition_model	Model
	rewards			RewardFunction
	actionFunction		ActionPicker
	generatorFunction	StateGenerator
}

/**
 * Agent Interface functions
 */
func (agent *ReflexAgent) CalculateAction(percept *system.Percept, reward system.Reward)(action *system.Action,log bool) {
	return agent.actionFunction(percept, reward)
}

type ReflexStateHash string
type ReflexStateActionHash string

type ReflexState struct {
	AiState
	a *ADPState
	id                                               StateID
	w_active, h_active, next_w_active, next_h_active bool

	Percept                                          *system.Percept

	circulationValue                                 float64
	hForeReverseDiffDelta                            float64

	hReverseDelta                                    float64
	boilerDelta                                      float64
	boilerDeltaSince                                 int		//
	boilerStableDuration                             float64	//
	hForeRunTempDelta                                float64
	kettleDelta                                      float64

	wStartIndicator                                  float64
	hStartIndicator                                  float64

	action                                           *ReflexAction

	counter                                          int
	timeState                                        int		//
	wEnergyRequirement                               int		//
	wEnergyPotential                                 int		//
	wLoadPotential                                   int		//
	hEnergyRequirement                               int
	hEnergyPotential                                 int
}

func (s *ReflexState) Successor(a *system.Action)(system.State){
	return nil
}
func (s *ReflexState) CompareTo(sPrime system.State)(int){
	return 0
}
func (s *ReflexState) Reward(a *system.Action,sPrime system.State)(system.Reward){
	return system.Reward(1)
}
func (s *ReflexState) getStateHash()(ReflexStateHash){
	//@todo
	return "TEST"
}
func (s *ReflexState) Terminal()(bool){
	return false
}
func (s *ReflexState) Equals(sPrime system.State)(bool){
	return false
}

type ADPState struct {
	AiState
				      //*system.ActorState

	hash                  string
	kettleLevel           int     //@done

	circulationValue      float64 //@done

	hForeReverseDiffDelta float64
	hReverseDelta         float64
	hForeRunTempDelta     float64
	hEnergyPotential      int
	hEnergyRequirement    int

	boilerDelta           int     //@done
	wEnergyRequirement    int
	wEnergyPotential      int     //@done
	wLoadPotential        int     //@done
}

func (a *ADPState) GetRelationName()(string){
	return ADP_STATE_TABLE
}
func (a *ADPState) CreateRelation()(){
	var stmnt_string string

	stmnt_string = fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s(" +
			"s_id INT NOT NULL AUTO_INCREMENT, " +
			"kettleLevel INT UNSIGNED NULL DEFAULT NULL," +
			"circulationValue FLOAT UNSIGNED NULL DEFAULT NULL," +
			"hForeReverseDiffDelta FLOAT SIGNED NULL DEFAULT NULL," +
			"hReverseDelta FLOAT SIGNED NULL DEFAULT NULL," +
			"hForeRunTempDelta FLOAT SIGNED NULL DEFAULT NULL," +
			"hEnergyPotential INT UNSIGNED NULL DEFAULT NULL," +
			"hEnergyRequirement INT UNSIGNED NULL DEFAULT NULL," +
			"boilerDelta INT SIGNED NULL DEFAULT NULL," +
			"wEnergyRequirement INT UNSIGNED NULL DEFAULT NULL," +
			"wEnergyPotential INT UNSIGNED NULL DEFAULT NULL," +
			"wLoadPotential INT SIGNED NULL DEFAULT NULL," +
			"uValue FLOAT SIGNED NULL DEFAULT 0.0," +
			"counter BIGINT UNSIGNED NULL DEFAULT 0," +
			"PRIMARY KEY(s_id)," +
			"UNIQUE value_key (kettleLevel,circulationValue,hForeReverseDiffDelta,hReverseDelta,hForeRunTempDelta,hEnergyPotential,hEnergyRequirement,boilerDelta,wEnergyRequirement,wEnergyPotential,wLoadPotential)" +
			") ENGINE=InnoDB DEFAULT CHARSET=latin1",
		a.GetRelationName())

	logger.StatementExecute(stmnt_string)
}
func (a *ADPState) Insert(val ...interface{})(){
	var stmnt_string string

	stmnt_string = fmt.Sprintf(
		"INSERT IGNORE INTO %s" +
			"(kettleLevel,circulationValue,hForeReverseDiffDelta,hReverseDelta,hForeRunTempDelta,hEnergyPotential,hEnergyRequirement,boilerDelta,wEnergyRequirement,wEnergyPotential,wLoadPotential,uValue,counter)" +
			" VALUES " +
			"(%d,%.2f,%.2f,%.2f,%.2f,%d,%d,%d,%d,%d,%d,%.2f,%d)",
		a.GetRelationName(),a.kettleLevel,a.circulationValue,a.hForeReverseDiffDelta,a.hReverseDelta,a.hForeRunTempDelta,a.hEnergyPotential,a.hEnergyRequirement,a.boilerDelta,a.wEnergyRequirement,a.wEnergyPotential,a.wLoadPotential,val[0].(float64),val[1].(int))

	logger.StatementExecute(stmnt_string)
}
func (a *ADPState) Delete()() {
	//@todo
}
func (a *ADPState) Update(val ...interface{})() {
	var stmnt_string string

	stmnt_string = fmt.Sprintf(
		"UPDATE %s " +
		"SET uValue = %.2f, counter = %d " +
		"WHERE " +
		"kettleLevel = %d AND " +
		"circulationValue = %.2f AND " +
		"hForeReverseDiffDelta = %.2f AND " +
		"hReverseDelta = %.2f AND " +
		"hForeRunTempDelta = %.2f AND " +
		"hEnergyPotential = %d AND " +
		"hEnergyRequirement = %d AND " +
		"boilerDelta = %d AND " +
		"wEnergyRequirement = %d AND " +
		"wEnergyPotential = %d AND " +
		"wLoadPotential = %d",
		a.GetRelationName(),val[0].(float64),val[1].(int),a.kettleLevel,a.circulationValue,a.hForeReverseDiffDelta,a.hReverseDelta,a.hForeRunTempDelta,a.hEnergyPotential,a.hEnergyRequirement,a.boilerDelta,a.wEnergyRequirement,a.wEnergyPotential,a.wLoadPotential)

	logger.StatementExecute(stmnt_string)
}

func (s *ADPState) Successor(a *system.Action)(system.State){
	return nil
}
func (s *ADPState) CompareTo(sPrime system.State)(int){
	return 0
}
func (s *ADPState) Reward(a *system.Action,sPrime system.State)(system.Reward){
	return system.Reward(1)
}
func (s *ADPState) getStateHash()(ReflexStateHash){
	return ReflexStateHash(fmt.Sprintf("kd%d_cv%.2f_,frdd%.2f_rd%.2f_fd%.2f_hep%d_her%d_,bd%d_wep%d_wep%d_wlp%d",s.kettleLevel,s.circulationValue,s.hForeReverseDiffDelta,s.hReverseDelta,s.hForeRunTempDelta,s.hEnergyPotential,s.hEnergyRequirement,s.boilerDelta,s.wEnergyPotential,s.wEnergyRequirement,s.wLoadPotential))
}
func (s *ADPState) Terminal()(bool){
	switch burner.GetValue() {
	case true:
	case false:
	}
	return false
}
func (s *ADPState) Equals(sPrime system.State)(bool){
	return *s==*sPrime.(*ADPState)
}
func (s *ADPState) String() string {
	return fmt.Sprintf("{[Kettle: %d],[Circulation: %.2f],[BoilerDelta: %d],[W-Potential: %d],[W-Load: %d]}",s.kettleLevel,s.circulationValue,s.boilerDelta,s.wEnergyPotential,s.wLoadPotential)
}

type ReflexAction struct {
	*system.Action
	//system.Action
}

func (a *ReflexAction) RollOut()(){
	fmt.Println("Called from outside")
	//(*a.Action).RollOut(a.Action)
	//a.Action.RollOut(&(a.Action))
}

func NewReflexAgent(a *system.Action)(r *ReflexAgent){
	r = &ReflexAgent{}
	r.previousAction = initAction()
	debug_system_action = a
	r.state_counter = make(map[ReflexStateHash]int)
	r.n_sa = make(map[ReflexStateActionHash]int)
	r.n_sPrime_sa = make(map[ReflexStateActionHash]map[ReflexStateHash]int)
	r.utilities = UtilityFunction{}
	r.transition_model = Model{}
	r.rewards = RewardFunction{}
	r.policy = func(system.State)(*system.Action){
		return &system.Action{
			//RollOut:DefaultRollOut,
		}
	}
	r.actionFunction = r.calculateReflexAction
	return
}

func NewADPAgent(a *system.Action)(r *ReflexAgent){
	r = &ReflexAgent{}
	r.previousAction = initAction()
	debug_system_action = a
	r.state_counter = CountTable{}
	r.state_hash_lookup = HashLookupTable{}
	r.n_sa = QStateTable{}
	r.n_sPrime_sa = TransitionTable{}
	r.utilities = UtilityFunction{}
	r.transition_model = Model{}
	r.rewards = RewardFunction{}
	r.discount_factor = 0.975
	r.policy = func(s system.State)(a *system.Action){
		if action_record[s.(*ADPState).getStateHash()] != nil {
			return action_record[s.(*ADPState).getStateHash()]
		} else {
			return &system.Action{
				//RollOut:DefaultRollOut
			}
		}
	}
	r.actionFunction = r.PassiveAdpLearning
	r.generatorFunction = r.generateADPState
	return
}

func (r *ReflexAgent) ResetReflexAgent()(){
	r.seqBeginningState = nil
	r.previousState = nil
	r.previousAction = initAction()
	r.transitionDuration = time.Duration(0)
	r.lastWEnergyLoad = time.Time{}
}

// type StateGenerator func(*system.Percept)(AiState)
func (r *ReflexAgent) generateADPState(p *system.Percept)(AiState){
	s := &ADPState{}

	s.calcWEnergyPotential(p)
	s.calcLoadPotential(p)

	if r.previousPercept != nil {
		// circulation value feature
		s.calcCirculationValue(r.previousState, &p.CurrentTime, &r.previousPercept.CurrentTime, r.previousAction.Action)

		var delta int
		var duration time.Duration

		// boiler delta feature
		delta,duration = p.GetBoilerDelta(r.previousPercept)
		if delta == 0 {
			s.boilerDelta = delta
			r.boilerStableDuration += duration.Seconds()
		} else {
			transition_seconds := int(r.boilerStableDuration+duration.Seconds())
			s.boilerDelta = approxBoilerDelta( delta * 6 / transition_seconds )
			r.boilerStableDuration = 0.0
		}

		//s.calcEnergyReq(r.previousState)

		// kettle level
		switch {
		case p.KettleTemp.GetValue() > 75000:
			s.kettleLevel = 5
		case p.KettleTemp.GetValue() > 65000:
			s.kettleLevel = 4
		case p.KettleTemp.GetValue() > 55000:
			s.kettleLevel = 3
		case p.KettleTemp.GetValue() > 45000:
			s.kettleLevel = 2
		case p.KettleTemp.GetValue() > 35000:
			s.kettleLevel = 1
		default:
			s.kettleLevel = 0
		}

		action_record[r.previousState.(*ADPState).getStateHash()]=r.previousAction.Action // @todo UNDO

	}

	//fmt.Printf("next state:%v\n",s)

	//@todo
	/*


		hForeReverseDiffDelta                            float64
		hReverseDelta                                    float64
		hForeRunTempDelta                                float64
		hEnergyPotential                                 int
		hEnergyRequirement                               int

		wEnergyRequirement                               int


	*/
	return s
}

/**
 * AiState Interface functions
 */
func (s *ADPState) calcWEnergyPotential(p *system.Percept)(){
	var temp float64
	temp = math.Pow( float64((p.BoilerMidTemp.GetValue() + p.BoilerTopTemp.GetValue()) / 2),2.0)
	temp *= 0.001 / float64(p.BoilerTopTemp.GetValue())
	s.wEnergyPotential = approxEnergyPotential(int(temp))
	return
}

func (s *ADPState) calcLoadPotential(p *system.Percept)(){
	s.wLoadPotential = int(float64(p.KettleTemp.GetValue() / 2 + (p.WForeRunTemp.GetValue() + p.WReverseRunTemp.GetValue()) / 4 - p.BoilerMidTemp.GetValue())/1000)
	s.wLoadPotential = approxLoadPotential(s.wLoadPotential)
	return
}

//@todo
func (s *ADPState) calcWEnergyReq(lastState AiState){
	// increase in boiler delta
	s.wEnergyRequirement = 0
	if lastState == nil {
		return
	} else {
		s.wEnergyRequirement = s.boilerDelta - lastState.(*ADPState).boilerDelta
	}
	return
}

// Calculates the circlation value of the heating circuit depending on the current state and the recorded state history
// @param current state of the system which is not included in the system's state history
func (s *ADPState) calcCirculationValue(predecessor AiState, percept_time, predecessor_time *time.Time, transitionAction *system.Action)(){
	if predecessor == nil || transitionAction == nil || predecessor_time == nil || percept_time == nil {
		s.circulationValue = 0.0
		fmt.Println("Circulation value could not be calculated.")
		return
	}
	gamma := .95
	duration_base := 15.0
	exp_sum_bound := 1.75
	freq := transitionAction.GetHPumpThrottle()
	freq_eff:=predecessor.getCirculationValue() * radiator_pump.GetMaxFreq()
	var divider, numerator float64
	if freq_eff > transitionAction.GetHPumpThrottle() {
		// frequency was decreased, freq_eff converges from above
		numerator = freq_eff
		divider = math.Max(transitionAction.GetHPumpThrottle(),5.0)
	} else {
		numerator = transitionAction.GetHPumpThrottle()
		divider = math.Max(freq_eff,5.0)
	}
	d := math.Min(numerator / divider,exp_sum_bound)

	approx_delta := math.Abs(freq_eff - freq)
	b1 := math.Abs(predecessor.getCirculationValue() - freq / radiator_pump.GetMaxFreq()) / 2.0 + d
	exp1 := math.Pow(b1,2.0) * -1.0
	exp2 := math.Pow(2.0,exp1) * -1.0
	exp3 := math.Pow(2.0,exp2)
	h := math.Pow(exp3,approx_delta) // invariant 1>=h>0
	duration := float64((*percept_time).Sub(*predecessor_time).Seconds())
	exact_value := (duration / duration_base * h * freq / radiator_pump.GetMaxFreq() + predecessor.getCirculationValue() * gamma)/(duration / duration_base * h + gamma)
	s.circulationValue = approxCirculationValue(exact_value)
	return
}

func (s *ADPState) getCirculationValue()(float64){
	return s.circulationValue
}

/**
 * State approximization functions
 */
func approxEnergyPotential(exactValue int)(approxValue int){
	var group_index, group_size int
	switch {
	case exactValue < 48:
		group_size = 4
	case exactValue < 58:
		group_size = 2
	default:
		group_size = 1
	}
	group_index = exactValue % group_size
	approxValue = exactValue - group_index
	return
}

func approxLoadPotential(exactValue int)(approxValue int){
	var sign, abs_value int
	if exactValue < 0 {
		sign = -1
	} else {
		sign = 1
	}
	abs_value = exactValue * sign

	var base, reminder, div, group_size int
	switch {
	case abs_value > 18:
		div = 18
		group_size = 2
	case abs_value > 12:
		div = 12
		group_size = 3
	case abs_value > 4:
		div = 4
		group_size = 4
	default:
		div = 2
		group_size = 2
	}
	base = abs_value / div
	reminder = abs_value % div / group_size
	approxValue = base * div + reminder * group_size
	approxValue *= sign
	return
}

func approxBoilerDelta(exactValue int)(approxValue int){
	var abs_value int
	var sign int
	if exactValue < 0 {
		sign = -1
	} else {
		sign = 1
	}
	abs_value = exactValue * sign
	var base, reminder, div, mod int
	switch {
	case abs_value > 320:
		div = 320
		mod = 80
	case abs_value > 154:
		div = 154
		mod = 32
	case abs_value > 74:
		div = 74
		mod = 16
	case abs_value > 34:
		div = 34
		mod = 8
	case abs_value > 12:
		div = 14
		mod = 4
	case abs_value > 4:
		div = 4
		mod = 2
	default:
		div = 1
		mod = 1
	}
	base = abs_value / div
	reminder = abs_value % div / mod
	approxValue = base * div + reminder * mod
	approxValue *= sign
	return
}

func approxCirculationValue(exactValue float64)(approxValue float64){
	approxValue = float64(int((exactValue + 0.005) * 100.0))/100.0
	return
}

/**
 * ReflexState functions
 */

// type StateGenerator func(*Percept)(AiState)
func (r *ReflexAgent) generateReflexState(p *system.Percept)(s *ReflexState){
	s = &ReflexState{Percept:p}

	//s.State = system.ActorState{}

	if s.w_active && s.h_active {
		s.id = STATE_w1_h1
	} else if s.w_active && !s.h_active {
		s.id = STATE_w1_h0
	} else if !s.w_active && s.h_active {
		s.id = STATE_w0_h1
	}

	// set actual state features
	s.setTimeState(p)
	s.calcWEnergyPotential(p)
	s.calcLoadPotential(p)
	s.calcBoilerDelta(r.previousState)
	s.calcWEnergyReq(r.previousState)

	var action *system.Action
	var current, previous *time.Time
	if r.previousAction == nil {
		action = nil
	} else {
		action = r.previousAction.Action
	}
	if r.previousState == nil {
		previous = nil
	} else {
		previous = &r.previousPercept.CurrentTime
	}
	current = &p.CurrentTime
	s.calcCirculationValue(r.previousState,current,previous,action)
	/*
	*/
	return
}

func (s *ReflexState) calcBoilerDelta(lastState AiState)(){
	s.boilerDeltaSince = 0
	s.boilerStableDuration = 0.0
	if lastState == nil {
		return
	} else if lastState.(*ReflexState).Percept != nil {
		temp_diff := s.Percept.BoilerMidTemp.GetValue() - lastState.(*ReflexState).Percept.BoilerMidTemp.GetValue()
		time_diff := s.Percept.CurrentTime.Sub(lastState.(*ReflexState).Percept.CurrentTime)
		s.boilerStableDuration = lastState.(*ReflexState).boilerStableDuration + time_diff.Seconds()
		if temp_diff == 0 || int(s.boilerStableDuration)==0 {
			s.boilerDeltaSince = lastState.(*ReflexState).boilerDeltaSince
		} else {
			s.boilerDeltaSince = temp_diff * 6 / int(s.boilerStableDuration)
			s.boilerStableDuration = 0.0
		}

		s.boilerDeltaSince = approxBoilerDelta(s.boilerDeltaSince)
	}
	return
}

func (r *ReflexAgent) calculateBoilerDelta(s, lastState *ReflexState)(int){
	boilerDeltaSince := 0
	boilerStableDuration := 0.0
	if lastState == nil {
		return boilerDeltaSince
	} else if lastState.Percept != nil {
		temp_diff := s.Percept.BoilerMidTemp.GetValue() - lastState.Percept.BoilerMidTemp.GetValue()
		time_diff := s.Percept.CurrentTime.Sub(lastState.Percept.CurrentTime)
		boilerStableDuration = lastState.boilerStableDuration + time_diff.Seconds()
		if temp_diff == 0 || int(boilerStableDuration)==0 {
			boilerDeltaSince = lastState.boilerDeltaSince
		} else {
			boilerDeltaSince = temp_diff * 6 / int(boilerStableDuration)
			boilerStableDuration = 0.0
		}

		boilerDeltaSince = approxBoilerDelta(boilerDeltaSince)
	}
	return boilerDeltaSince
}

func (s *ReflexState) setTimeState(p *system.Percept)(){
	s.timeState = p.CurrentTime.Hour()
}

func (s *ReflexState) GetIdentifier()(hash string){
	hash = fmt.Sprintf("ER:%d,EP:%d,LP:%d",s.boilerDeltaSince,s.wEnergyPotential,s.wLoadPotential)
	return
}

func (r *ReflexAgent) GetStateHash()(){
	for _,slice := range sequences {
		for _, l := range slice {
			if head := (*l).Back(); head != nil {
				if !head.Value.(*ReflexState).action.GetBurnerState() {
					var last int
					var init bool
					last = 0
					init = false
					for ; head != nil; head = head.Prev() {
						key := head.Value.(*ReflexState).boilerDeltaSince + 500
						if init {
							var key_count, total_count int
							for j := 0; j < len(Potential_Count[last]); j++ {
								if j == key {
									key_count = Potential_Count[last][j]
								}
								total_count += Potential_Count[last][j]
							}
							if total_count > 0 {
								fmt.Printf("%v (%.4f),", key - 500, float64(key_count) / float64(total_count))
							}
						}
						last = key
						init = true
					}
					//fmt.Printf("%v",(*l).Back().Value.(*ReflexState).boilerDeltaSince)
					fmt.Println()
				}
			}
		}
	}
	return
}

func (s *ReflexState) SetBurnerVal(number int){

	if s.action == nil {
		s.action = initAction()
	}

	if number == 0 {
		s.action.SetBurnerState(false)
	} else {
		s.action.SetBurnerState(true)
	}

	return
}

func (s *ReflexState) SetWPumpVal(number int){

	if s.action == nil {
		s.action = initAction()
	}

	if number == 0 {
		s.action.SetWPumpState(false)
	} else {
		s.action.SetWPumpState(true)
	}

	return
}

/*
func (r *ReflexAgent) addToSequenceAndGenerateRecord(sPrimeState AiState)(){
	if r.seqBeginningState == nil {
		r.seqBeginningState = sPrimeState
		seqHistory.Init()
		r.transitionDuration = time.Duration(0)*time.Second
	} else if r.seqBeginningState.action != nil {

		// check layer transition took place
		if sPrimeState.action.GetBurnerState() != r.seqBeginningState.action.GetBurnerState() || sPrimeState.action.GetWPumpState() != r.seqBeginningState.action.GetWPumpState() {
			seqHistory.PushFront(sPrimeState)
			key := seqHistory.Front().Value.(*ReflexState).Percept.CurrentTime.Hour()
			sequences[key] = append(sequences[key], seqHistory)
			seqHistory = list.New()
			r.seqBeginningState = sPrimeState
			r.transitionDuration = time.Duration(0)*time.Second
		} else {
			sState := seqHistory.Front().Value.(*ReflexState) // r.previousState

			if !(r.seqBeginningState.action.GetBurnerState() || r.seqBeginningState.action.GetWPumpState()) {
				// burner off and w_pump off and no layer transition

				h:= sPrimeState.Percept.CurrentTime.Hour()
				//m:= r.lastState.percept.CurrentTime.Minute()
				d:= sPrimeState.Percept.CurrentTime.Weekday()
				if d <= 4 && (h > 21 || h < 6) || d > 4 && h < 6 {
					//beginner := seqHistory.Back().Value.(*ReflexState) // r.seqBeginningState
					if sState.boilerDeltaSince == sPrimeState.boilerDeltaSince && sState.wEnergyPotential == sPrimeState.wEnergyPotential && sState.wLoadPotential == sPrimeState.wLoadPotential {
						// not internal state transition occured
						sPrimeState.Percept.CurrentTime.Sub(sState.Percept.CurrentTime)
						r.transitionDuration += sPrimeState.Percept.CurrentTime.Sub(sState.Percept.CurrentTime)*time.Second
					} else {
						r.transitionDuration = 0
					}

					if base_delta_transitions[fmt.Sprintf("BT:%d x DE:%d",sState.Percept.BoilerMidTemp.GetValue(),sState.boilerDeltaSince)] == nil {
						base_delta_transitions[fmt.Sprintf("BT:%d x DE:%d",sState.Percept.BoilerMidTemp.GetValue(),sState.boilerDeltaSince)] = make(map[int]int)
					}
					base_delta_transitions[fmt.Sprintf("BT:%d x DE:%d",sState.Percept.BoilerMidTemp.GetValue(),sState.boilerDeltaSince)][sPrimeState.wEnergyRequirement]++

					if base_potential_transitions[sState.wEnergyPotential] == nil {
						base_potential_transitions[sState.wEnergyPotential] = make(map[int]int)
					}
					base_potential_transitions[sState.wEnergyPotential][sPrimeState.wEnergyPotential]++
				}
			}

			if delta_transitions[fmt.Sprintf("BT:%d x DE:%d",sState.Percept.BoilerMidTemp.GetValue(),sState.boilerDeltaSince)] == nil {
				delta_transitions[fmt.Sprintf("BT:%d x DE:%d",sState.Percept.BoilerMidTemp.GetValue(),sState.boilerDeltaSince)] = make(map[int]int)
			}
			delta_transitions[fmt.Sprintf("BT:%d x DE:%d",sState.Percept.BoilerMidTemp.GetValue(),sState.boilerDeltaSince)][sPrimeState.wEnergyRequirement]++

			if potential_transitions[sState.wEnergyPotential] == nil {
				potential_transitions[sState.wEnergyPotential] = make(map[int]int)
			}
			potential_transitions[sState.wEnergyPotential][sPrimeState.wEnergyPotential]++
		}
	}
	seqHistory.PushFront(sPrimeState)
}
*/

func (s *ADPState) getStateActionHash(a *system.Action)(ReflexStateActionHash){
	state_hash := s.getStateHash()
	var action_hash = "B:false_W:false_H:false_HFREQ:0.00"
	if a != nil {
		action_hash = fmt.Sprintf("B:%v_W:%v_H:%v_HFREQ:%.2f",a.GetBurnerState(),a.GetWPumpState(),a.GetHPumpState(),a.GetHPumpThrottle())
	}
	return ReflexStateActionHash(fmt.Sprintf("STATE:%s_ACTION:%s",state_hash,action_hash))
}

func (r *ReflexAgent) SetLastAction(a *system.Action)(){
	r.previousAction.Action.SetBurnerState(a.GetBurnerState())
	r.previousAction.Action.SetWPumpState(a.GetWPumpState())
	r.previousAction.Action.SetHPumpThrottle(a.GetHPumpThrottle())
	r.previousAction.Action.SetHPumpState(a.GetHPumpState())

	//@todo
	r.previousAction.Action.Insert()
	return
}

func (r *ReflexAgent) PassiveAdpLearning(p *system.Percept, rPrime system.Reward)(a *system.Action,log bool){
	var sPrime AiState
	var stateTransition bool
	var sPrimeHash ReflexStateHash

	sPrime = r.generatorFunction(p)
	sPrimeHash = sPrime.(*ADPState).getStateHash()

	defer func(){
		r.state_counter[sPrimeHash]++
		var total int
		for _,count := range r.state_counter {
			total += count
		}
		if r.previousPercept != nil {
			r.transitionDuration += p.CurrentTime.Sub(r.previousPercept.CurrentTime)
		}
		if stateTransition || r.previousPercept == nil {
			r.transitionDuration = time.Duration(0)
		}
		r.previousPercept = p

		r.previousPercept.Insert()
		if r.state_counter[sPrimeHash] == 1 {
			r.previousState.(*ADPState).Insert(r.utilities[sPrimeHash],r.state_counter[sPrimeHash])
		} else {
			// update value and counter
			r.previousState.(*ADPState).Update(r.utilities[sPrimeHash],r.state_counter[sPrimeHash])
		}
		//fmt.Printf("States: %d Totals: %d \n",len(r.state_counter),total)
	}()

	if r.state_counter[sPrimeHash] == 0 {
		// state is new
		r.state_hash_lookup[sPrimeHash] = sPrime
		r.utilities[sPrimeHash]=float64(rPrime)
		r.rewards[sPrimeHash]=rPrime
	}
	if r.previousState != nil {

		// check if transition to a new state happened
		stateTransition = !r.previousState.(*ADPState).Equals(sPrime)

		var sStateActionHash ReflexStateActionHash

		// generate state,action hash
		sStateActionHash = r.previousState.(*ADPState).getStateActionHash(r.previousAction.Action)

		// increment q_state counter
		r.n_sa[sStateActionHash]++

		if r.n_sPrime_sa[sStateActionHash] == nil {
			r.n_sPrime_sa[sStateActionHash] = make(map[ReflexStateHash]int)
		}

		// increment transition counter for q_state successor transition
		r.n_sPrime_sa[sStateActionHash][sPrimeHash]++

		// recalculate model probabilities for all q_state successor transitions
		for successorHash,count := range r.n_sPrime_sa[sStateActionHash] {
			if r.transition_model[sStateActionHash] == nil {
				r.transition_model[sStateActionHash] = make(map[ReflexStateHash]float64)
			}
			r.transition_model[sStateActionHash][successorHash] = float64(count) / float64(r.n_sa[sStateActionHash])
		}

		/*
		defer func(){
			for s,m := range r.transition_model{
				fmt.Printf("transitions for %s:\t",s)
				for _,prob := range m {
					fmt.Printf("%.2f, ",prob)
				}
				fmt.Println()
			}
		}()
		*/
	}

	// Policy Evaluation
	r.utilities = *policyEvaluation(&r.utilities,&r.transition_model,&r.rewards,r.discount_factor,r.policy,&r.state_hash_lookup)

	if sPrime.(*ADPState).Terminal() {
		r.previousState = nil
		r.previousAction.Action = nil
	} else {
		r.previousState = sPrime
		if r.previousAction == nil {
			r.previousAction = initAction()
		}
		r.previousAction = &ReflexAction{Action:r.policy(sPrime)}
	}
	return
}

func policyEvaluation(u *UtilityFunction, p *Model, r *RewardFunction, gamma float64, policyFunction Policy, dict *HashLookupTable)(*UtilityFunction){
	//start := time.Now()
	u_new := UtilityFunction{}

	for stateHash,_ := range *u {
		// retrieve state pointer from hash value
		s := (*dict)[stateHash]

		var sum float64
		var a *system.Action

		// fetch action according to policy
		a = policyFunction(s)

		// get transition map for state,action pair
		probs := (*p)[s.(*ADPState).getStateActionHash(a)]

		// loop over all neighbours and update values
		for neighbourHash,transitionProb := range probs {
			sum += transitionProb * float64((*u)[neighbourHash])
		}
		u_new[stateHash] = float64((*r)[stateHash]) + gamma * sum

	}
	//elapsed := time.Since(start)
	//fmt.Printf("Calculation took %.2f seconds\n",elapsed.Seconds())
	return &u_new
}

func (r *ReflexAgent) InitProcessPerceptData(p *system.Percept, vals ...int)(s *ReflexState){
	// initialize state
	s = r.generateReflexState(p)

	// set actual state system environment
	for i,v := range vals {
		switch i {
		case 0:
			s.SetBurnerVal(v)
		case 1:
			s.SetWPumpVal(v)
		}
	}

	// init utility and reward if new state explored

	//r.addToSequenceAndGenerateRecord(s)

	r.previousState = s
	return
}

/*
func (r *ReflexAgent) ProcessNewState(s *ReflexState)(){
	if r.previousState == nil {
		r.previousState = s
		r.previousAction = s.action
		r.seqBeginningState = s
		s.wEnergyRequirement = 0
	} else {
		if r.seqBeginningState.action != nil {
			if s.action.GetBurnerState() != r.previousState.action.GetBurnerState() || s.action.GetWPumpState() != r.previousState.action.GetWPumpState() {
				r.seqBeginningState = s
			} else {
				if !(r.previousAction.GetBurnerState() || r.previousAction.GetWPumpState()) {
					e := base_delta_transitions[fmt.Sprintf("BT:%d x DE:%d",r.previousState.Percept.BoilerMidTemp.GetValue(),r.previousState.boilerDeltaSince)][s.wEnergyRequirement]
					var t int
					for _,count := range base_delta_transitions[fmt.Sprintf("BT:%d x DE:%d",r.previousState.Percept.BoilerMidTemp.GetValue(),r.previousState.boilerDeltaSince)] {
						t += count
					}
					ae := delta_transitions[fmt.Sprintf("BT:%d x DE:%d",r.previousState.Percept.BoilerMidTemp.GetValue(),r.previousState.boilerDeltaSince)][s.wEnergyRequirement]
					var at int
					for _,count := range delta_transitions[fmt.Sprintf("BT:%d x DE:%d",r.previousState.Percept.BoilerMidTemp.GetValue(),r.previousState.boilerDeltaSince)] {
						at += count
					}
					if t == 0 {
						t=1
						e=0
					}
					fmt.Printf("%v\t%.2f (night)\t%.2f (all)\t[%d>%d]",r.previousState.Percept.CurrentTime,100.0*float64(e)/float64(t),100.0*float64(ae)/float64(at),r.previousState.wEnergyRequirement,s.wEnergyRequirement)

					npb := base_potential_transitions[r.previousState.wEnergyPotential][s.wEnergyPotential]
					var npt int
					for _,count := range base_potential_transitions[r.previousState.wEnergyPotential] {
						npt += count
					}
					pab := potential_transitions[r.previousState.wEnergyPotential][s.wEnergyPotential]
					var pat int
					for _,count := range potential_transitions[r.previousState.wEnergyPotential] {
						pat += count
					}
					if t > 0 {
						fmt.Printf("\t%.2f (night)\t%.2f (all)\t[%d>%d]\n",100.0*float64(npb)/float64(npt),100.0*float64(pab)/float64(pat),r.previousState.wEnergyPotential,s.wEnergyPotential)
					}
				}
			}
		}
		r.previousState = s
		r.previousAction = s.action
	}
}
*/
func GetDeltaCount()(){
	fmt.Printf("#Delta-States: %d\n#Load-State: %d\n#Requ-States: %d\n",total_count,len(states),0)
	var c int
	for k,v := range states {
		if v > 20 {
			fmt.Printf("%s : %d\n",k,v)
			c++
		}
	}
	fmt.Println(c)
}

func (s *ReflexState) calcWEnergyRequirement(ancestor AiState, t time.Time)(){
	var key, key1 int
	//duration := p.CurrentTime.Sub(t)
	//key = int(duration.Minutes()/10)
	key = ancestor.(*ReflexState).boilerDeltaSince + 500
	//key = int(s.wEnergyPotential/100.00)
	if key > 999 {
		key = 999
		fmt.Printf("Duration does not fit into array %v\n",s.Percept.CurrentTime)
	}
	key1 = s.boilerDeltaSince + 500
	//key1=int(s.wEnergyPotential/100.00)
	if key1 > 999 {
		key1 = 999
		fmt.Printf("Potential does not fit into array (too high) %v\n",s.Percept.CurrentTime)
	} else if key1 < 0 {
		key1 = 0
		fmt.Printf("Potential does not fit into array (too low) %v\n",s.Percept.CurrentTime)
	}
	Potential_Count[key][key1]++
	s.wEnergyRequirement = 0
}

func (r *ReflexAgent) PrintData()() {
	var line_empty bool
	for i := 0; i < len(Potential_Count); i++ {
		line_empty = true
		for j := 0; j < len(Potential_Count[i]); j++ {
			if Potential_Count[i][j] > 0 {
				if line_empty {
					fmt.Printf("[KEY: %d]\t", i-500)
				}
				fmt.Printf("(%d,%d),", j-500,Potential_Count[i][j])
				line_empty = false
			}
		}
		if !line_empty {
			fmt.Println()
		}
	}
	return
}

func (r *ReflexAgent) PrintSeq()(){
	fmt.Printf("Seqs: %d\n",len(sequences))
	fmt.Printf("Potential Values seen: %d\n",len(base_potential_transitions))
	for k,l := range base_delta_transitions {
		fmt.Printf("Counts for event %d\t",k)
		for e,count := range l {
			fmt.Printf("%d:%d,",e,count)
		}
		fmt.Println()
	}
	fmt.Printf("Delta Values seen: %d\n",len(base_delta_transitions))
}

func (agent *ReflexAgent) initStateHistory(p *system.Percept)(){
	wState,_ := buffer_pump.GetState()
	hState,hFreq := radiator_pump.GetState()
	bState := burner.GetValue()
	a:=&system.Action{
		//RollOut:DefaultRollOut
	}
	a.SetWPumpState(wState)
	a.SetBurnerState(bState)
	a.SetHPumpState(hState)
	a.SetHPumpThrottle(hFreq)
	action := &ReflexAction{a}
	state := &ReflexState{id:STATE_w0_h0, w_active:false, h_active:false, next_w_active:false, next_h_active:false, Percept:p, action:action, circulationValue:0.0}
	agent.seqBeginningState = state
	seqHistory.PushFront(state)
	return
}

func (agent *ReflexAgent) GetCurrentHStartingTemp(percept *system.Percept)(int){
	return 65 - h_deviation
}

func (agent *ReflexAgent) GetCurrentWStartingTemp(percept *system.Percept)(int){
	return 0
}

// Calculates the circulation value as running average over intervals of definded length. The
// @param the actual state for which the circulation value is calculated and set
func (agent *ReflexAgent) calcCirculationValueRunningAverage(actualState *ReflexState)(){
	gamma := 0.925
	intervall_length := 3.0
	default_circulation_value := 0.0

	predecessor_state := seqHistory.Front().Value.(*ReflexState)
	time_diff := actualState.Percept.CurrentTime.Sub(predecessor_state.Percept.CurrentTime).Seconds()

	exp := math.Pow(2.0,-2.5)
	base := math.Pow(2.0,-exp)

	if predecessor_state == nil {
		actualState.circulationValue = default_circulation_value
	} else {
		var ratio, func_value, throttle, history float64
		if predecessor_state.action == nil {
			_,throttle = radiator_pump.GetState()
		} else {
			throttle = predecessor_state.action.GetHPumpThrottle()
		}
		history = predecessor_state.circulationValue
		for time_diff := actualState.Percept.CurrentTime.Sub(predecessor_state.Percept.CurrentTime).Seconds(); time_diff >= intervall_length; time_diff -= intervall_length{
			func_value = history * radiator_pump.GetMaxFreq() - throttle
			if func_value < 0.0 {
				func_value *= -1.0
			}
			ratio = math.Pow(base,func_value)
			actualState.circulationValue = (ratio * throttle / radiator_pump.GetMaxFreq() + gamma * history) / (ratio+gamma)
			history = actualState.circulationValue
		}
		actualState.circulationValue = (time_diff / intervall_length * ratio * throttle / radiator_pump.GetMaxFreq() + gamma * history) / (time_diff/intervall_length*ratio+gamma)
	}
}

// Calculates a weighted rate of reverse temperature change in heating circuit. The calculation approach considers the
// last reverse_heating_delta_change_count temperature change occurences and uses the duration to calculate a mean value
// for each interval.
// @param current state of the system which is not included in the system's state history
// @return float64 actual rate of reverse temperature change per second, normalized by number of considered intervals
func (agent *ReflexAgent) calcHReverseDelta(actualState *ReflexState)(delta float64){
	// Iterate through list and print its contents.
	delta = 0.0
	normalizer := 0.0
	gamma := 0.7
	i := 0

	var time_diff,weight float64
	var val_diff int

	v_new := actualState.Percept.HReverseRunTemp.GetValue()
	t_new := actualState.Percept.CurrentTime
	for e := seqHistory.Front(); e != nil || i == reverse_heating_delta_change_count; e = e.Next() {
		if val_diff = v_new - e.Value.(*ReflexState).Percept.HReverseRunTemp.GetValue(); val_diff != 0 {
			// wait until percept is found that differs by temp value
			weight = math.Pow(gamma,float64(i))
			time_diff = t_new.Sub(e.Value.(*ReflexState).Percept.CurrentTime).Seconds()
			if time_diff != 0 {
				delta += weight * (float64(val_diff) / time_diff)
			}
			normalizer += weight
			i+=1
			// update interval base
			v_new = e.Value.(*ReflexState).Percept.HReverseRunTemp.GetValue()
			t_new = e.Value.(*ReflexState).Percept.CurrentTime
		}
	}
	if i > 0 {
		delta *= (1.0 / normalizer)
	}
	delta *= float64(i/reverse_heating_delta_change_count)
	return
}

// Calculates and sets the current temperature change in millidegree per minute for the actual state
// @param the actual state for which the value is calculated and set
func (agent *ReflexAgent) calcHReverseDeltaRunningAverage(actualState *ReflexState)(){
	gamma := 0.85
	time_diff_minimum := 5.0
	default_delta_value := 0.0

	predecessor_state := seqHistory.Front().Value.(*ReflexState)
	var time_diff, temp_diff float64

	if predecessor_state == nil {
		actualState.hReverseDelta = default_delta_value
		return
	} else {
		time_diff = actualState.Percept.CurrentTime.Sub(predecessor_state.Percept.CurrentTime).Seconds()
	}

	if actualState.Percept.HReverseRunTemp.IsValid() && predecessor_state.Percept.HReverseRunTemp.IsValid() {
		temp_diff = float64(actualState.Percept.HReverseRunTemp.GetValue() - predecessor_state.Percept.HReverseRunTemp.GetValue()) * 60.0 / time_diff
	} else {
		temp_diff = predecessor_state.hReverseDelta
	}

	if time_diff <= time_diff_minimum {
		actualState.hReverseDelta = default_delta_value + gamma * predecessor_state.hReverseDelta
	} else {
		actualState.hReverseDelta = temp_diff + gamma * predecessor_state.hReverseDelta
	}

	return
}

func (agent *ReflexAgent) calcBoilerDeltaRunningAverage(actualState *ReflexState)(){
	gamma := 0.75
	time_diff_minimum := 5.0
	default_boiler_delta_value := 0.0

	predecessor_state := seqHistory.Front().Value.(*ReflexState)
	var time_diff, temp_diff float64

	if predecessor_state == nil {
		actualState.boilerDelta = default_boiler_delta_value
		return
	} else {
		time_diff = actualState.Percept.CurrentTime.Sub(predecessor_state.Percept.CurrentTime).Seconds()
	}

	if actualState.Percept.BoilerMidTemp.IsValid() && predecessor_state.Percept.BoilerMidTemp.IsValid() {
		temp_diff = float64(actualState.Percept.BoilerMidTemp.GetValue() - predecessor_state.Percept.HReverseRunTemp.GetValue()) * 60.0 / time_diff
	} else {
		temp_diff = predecessor_state.boilerDelta
	}

	if time_diff <= time_diff_minimum || !actualState.Percept.BoilerMidTemp.IsValid() {
		actualState.boilerDelta = default_boiler_delta_value + gamma * predecessor_state.boilerDelta
	} else {
		actualState.boilerDelta =  temp_diff + gamma * predecessor_state.boilerDelta
	}

	return
}

func (agent *ReflexAgent) calcHForeRunTempDeltaRunningAverage(actualState *ReflexState)(){
	gamma := 0.85
	time_diff_minimum := 5.0
	default_delta_value := 0.0

	predecessor_state := seqHistory.Front().Value.(*ReflexState)
	var time_diff, temp_diff float64

	if predecessor_state == nil {
		actualState.hForeRunTempDelta = default_delta_value
		return
	} else {
		time_diff = actualState.Percept.CurrentTime.Sub(predecessor_state.Percept.CurrentTime).Seconds()
	}

	if actualState.Percept.HForeRunTemp.IsValid() && predecessor_state.Percept.HForeRunTemp.IsValid() {
		temp_diff = float64(actualState.Percept.HForeRunTemp.GetValue() - predecessor_state.Percept.HForeRunTemp.GetValue()) * 60.0 / time_diff
	} else {
		temp_diff = predecessor_state.hForeRunTempDelta
	}

	if time_diff <= time_diff_minimum {
		actualState.hForeRunTempDelta = default_delta_value + gamma * predecessor_state.hForeRunTempDelta
	} else {
		actualState.hForeRunTempDelta = temp_diff + gamma * predecessor_state.hForeRunTempDelta
	}

	return
}

func (agent *ReflexAgent) calcKettleDeltaRunningAverage(actualState *ReflexState)(){
	gamma := 0.75
	time_diff_minimum := 3.0
	default_delta_value := 0.0

	predecessor_state := seqHistory.Front().Value.(*ReflexState)
	var time_diff, temp_diff float64

	if predecessor_state == nil {
		actualState.kettleDelta = default_delta_value
		return
	} else {
		time_diff = actualState.Percept.CurrentTime.Sub(predecessor_state.Percept.CurrentTime).Seconds()
	}

	if actualState.Percept.KettleTemp.IsValid() && predecessor_state.Percept.KettleTemp.IsValid() {
		temp_diff = float64(actualState.Percept.KettleTemp.GetValue() - predecessor_state.Percept.KettleTemp.GetValue()) * 60.0 / time_diff
	} else {
		temp_diff = predecessor_state.kettleDelta
	}

	if time_diff <= time_diff_minimum {
		actualState.kettleDelta = default_delta_value + gamma * predecessor_state.kettleDelta
	} else {
		actualState.kettleDelta = temp_diff + gamma * predecessor_state.kettleDelta
	}

	return
}

func (agent *ReflexAgent) calcHForeReverseDiffDeltaRunningAverage(actualState *ReflexState)(){
	gamma := 0.6
	time_diff_minimum := 5.0
	default_boiler_delta_value := 0.0

	predecessor_state := seqHistory.Front().Value.(*ReflexState)
	var time_diff, temp_delta float64
	var actualStateDelta, lastStateDelta int

	if predecessor_state == nil {
		actualState.hForeReverseDiffDelta = default_boiler_delta_value
		return
	} else {
		time_diff = actualState.Percept.CurrentTime.Sub(predecessor_state.Percept.CurrentTime).Seconds()
	}

	if actualState.Percept.BoilerMidTemp.IsValid() && predecessor_state.Percept.BoilerMidTemp.IsValid() {
		actualStateDelta = actualState.Percept.HForeRunTemp.GetValue() - actualState.Percept.HForeRunTemp.GetValue()
		lastStateDelta = predecessor_state.Percept.HForeRunTemp.GetValue() - predecessor_state.Percept.HForeRunTemp.GetValue()
		temp_delta = float64(actualStateDelta - lastStateDelta) * 60.0 / time_diff
	} else {
		temp_delta = predecessor_state.hForeReverseDiffDelta
	}

	if time_diff <= time_diff_minimum || !actualState.Percept.BoilerMidTemp.IsValid() {
		actualState.hForeReverseDiffDelta = default_boiler_delta_value + gamma * predecessor_state.hForeReverseDiffDelta
	} else {
		actualState.hForeReverseDiffDelta =  temp_delta + gamma * predecessor_state.hForeReverseDiffDelta
	}

	return
}

func (agent *ReflexAgent) burnerDeactivatedInSecondsEstimate(actualState *ReflexState)(float64) {
	switch actualState.id {
	case STATE_w1_h0:
		if actualState.Percept.HForeRunTemp.GetValue() >= agent.GetCurrentHStartingTemp(actualState.Percept){
			return 1.0
		}
	}
	return 0.0
}

func (agent *ReflexAgent) takeStateTransition(actualState *ReflexState)(transition bool){
	switch actualState.id {
	case STATE_w1_h0:
		// water only
		switch agent.seqBeginningState.(*ReflexState).action.GetBurnerState() {
		case true: // burner is already running
			// first check transition to true,true
			if actualState.Percept.HForeRunTemp.GetValue() < agent.GetCurrentHStartingTemp(actualState.Percept) {
				actualState.next_h_active = true
				transition = true
			}
		// do not consider burner off case
		}

		if actualState.Percept.BoilerMidTemp.GetValue() >= w_target_bound {
			actualState.next_w_active = false
			transition = true
		}
		return
	case STATE_w0_h1:
		// heating only
		switch agent.seqBeginningState.(*ReflexState).action.GetBurnerState() {
		case true:
			if agent.seqBeginningState.(*ReflexState).Percept.BoilerMidTemp.GetValue() < w_lower_bound {
				actualState.next_w_active = true
				transition = true
			} else if actualState.Percept.HForeRunTemp.GetValue() >= agent.GetCurrentHStartingTemp(actualState.Percept) + h_deviation {
				actualState.next_h_active = false
				transition = true
			}
		}
		//s.action = &action
		return
	case STATE_w1_h1:
		// both on
		if actualState.Percept.HForeRunTemp.GetValue() >= agent.GetCurrentHStartingTemp(actualState.Percept) + h_deviation {
			actualState.next_h_active = false
			transition = true
		}
		if actualState.Percept.BoilerMidTemp.GetValue() >= w_target_bound {
			actualState.next_w_active = false
			transition = true
		}
		//s.action = &action
		return
	default:
		// both off
		if actualState.Percept.BoilerMidTemp.GetValue() < w_lower_bound {
			actualState.next_w_active = true
			transition = actualState.next_w_active
		}

		if actualState.Percept.HForeRunTemp.GetValue() < agent.GetCurrentHStartingTemp(actualState.Percept) {
			actualState.next_h_active = true
			transition = true
		}

		on,_ := buffer_pump.GetState()
		switch on {
		case true:
			if 2 * actualState.Percept.BoilerMidTemp.GetValue() - actualState.Percept.WReverseRunTemp.GetValue() < actualState.Percept.KettleTemp.GetValue() {
				// only toggle w_pump on if the gap between actual boiler temp and reverse temp is not too high
				transition = true
			}
		default:
			if actualState.Percept.KettleTemp.GetValue() - 2000 <= actualState.Percept.BoilerMidTemp.GetValue() {
				transition = true
			}
		}

		_,freq := radiator_pump.GetState()
		if actualState.circulationValue - freq < 0.005 && actualState.circulationValue - freq > -0.005 {
			transition = true
		}
		return
	}
}

func (agent *ReflexAgent) calculateReflexAction(percept *system.Percept, reward system.Reward)(action *system.Action,log bool) {

	next_action := &ReflexAction{action}

	//next_action.RollOut()
	//action = Action{}


	// make sure state history contains at least one state
	if seqHistory.Len() <= 0 {
		agent.initStateHistory(percept)
	}

	// initialize state
	s := ReflexState{
		w_active:seqHistory.Front().Value.(*ReflexState).next_w_active,
		h_active:seqHistory.Front().Value.(*ReflexState).next_h_active,
		next_w_active:seqHistory.Front().Value.(*ReflexState).next_w_active,
		next_h_active:seqHistory.Front().Value.(*ReflexState).next_h_active,
		Percept:percept}

	if s.w_active && s.h_active {
		s.id = STATE_w1_h1
	} else if s.w_active && !s.h_active {
		s.id = STATE_w1_h0
	} else if !s.w_active && s.h_active {
		s.id = STATE_w0_h1
	}

	var transition bool

	// add current state to history
	defer func(bool){
		s.action = next_action
		agent.seqBeginningState = &s
		if seqHistory.Len() < 512 {
			seqHistory.PushFront(&s)
		} else  {
			seqHistory.Remove(seqHistory.Back())
			seqHistory.PushFront(&s)
		}
		if seqHistory.Len() > 0 {
			fmt.Printf("%v\n", seqHistory.Front().Value.(*ReflexState).hReverseDelta)
		}
	}(transition)

	// update state
	if seqHistory.Len() > 0 {
		// assign reverse heating delta value to state &s
		agent.calcHReverseDeltaRunningAverage(&s)

		// assign circulatin value to state &s
		agent.calcCirculationValueRunningAverage(&s)
	}

	transition = agent.takeStateTransition(&s)

	if transition {
		fmt.Printf("Not good here\n",nil)
		// calculate new action
		//action.GetHPumpThrottle() = agent.calcHPumpThrottle(&percept,agent.seqBeginningState.action.GetHPumpThrottle(),action.GetBurnerState(),&s)
	} else {
		action.SetBurnerState(agent.seqBeginningState.(*ReflexState).action.GetBurnerState())
		action.SetHPumpThrottle(agent.seqBeginningState.(*ReflexState).action.GetHPumpThrottle())
		action.SetWPumpState(agent.seqBeginningState.(*ReflexState).action.GetWPumpState())
		action.SetHPumpState(agent.seqBeginningState.(*ReflexState).action.GetHPumpState())
		//action.RollOut = DefaultRollOut
	}
	return
}

func (agent *ReflexAgent) setBurnerAction(state *ReflexState)(burner_state,transition bool){
	/*
	current_action := state.action
	percept := state.percept
	burner_state = current_action.GetBurnerState()

	// state_id contains id of the current sequence's beginning state
	switch state.state_id {
	case STATE_w1_h0:
		// water only
		switch action.GetBurnerState() {
		case true:
			// first check transition to true,true
			if percept.HForeRunTemp.GetValue() < agent.GetCurrentHStartingTemp(&percept) - h_deviation {
				state.next_h_active = true
				log = true
			} else if percept.BoilerMidTemp.GetValue() >= w_target_bound {
				action.GetBurnerState() = false
				state.next_w_active = false
				log = true
			}
		}
		//s.action = &action
		return
	case STATE_w0_h1:
		// heating only
		switch action.GetBurnerState() {
		case true:
			if percept.BoilerMidTemp.GetValue() < w_lower_bound {
				s.next_w_active = true
				log = true
			} else if percept.HForeRunTemp.GetValue() >= agent.GetCurrentHStartingTemp(&percept) + h_deviation {
				action.GetBurnerState() = false
				s.next_h_active = false
				log = true
			}
		}
		//s.action = &action
		return
	case STATE_w1_h1:
		// both on
		if percept.HForeRunTemp.GetValue() >= agent.GetCurrentHStartingTemp(&percept) + h_deviation {
			s.next_h_active = false
			log = true
		}
		if percept.BoilerMidTemp.GetValue() >= w_target_bound {
			s.next_w_active = false
			log = true
		}
		//s.action = &action
		return
	default:
		// both off

		if percept.BoilerMidTemp.GetValue() < agent.GetCurrentWStartingTemp(percept) {
			transition = true
			burner_state = true
			state.next_w_active = true
		}

		if percept.HForeRunTemp.GetValue() < agent.GetCurrentHStartingTemp(percept) {
			transition = true
			burner_state = true
			state.next_h_active = true
		}

		return
	}
	*/
	return
}

// @return estimation in minutes when next w start is required
func (agent *ReflexAgent) setWStartIndicator(state *ReflexState)(){
	percept := state.Percept
	minutes_since_last_load := state.Percept.CurrentTime.Sub(agent.lastWEnergyLoad).Minutes()
	minutes_left := int((agent.GetCurrentWStartingTemp(percept)-state.Percept.BoilerMidTemp.GetValue()) / int(state.boilerDelta))
	if minutes_left < 0 {
		minutes_left *= -1
	}
	switch {
	case minutes_since_last_load < 15:
		state.wStartIndicator = float64(minutes_left) * 1.5
		return
	case minutes_since_last_load < 30:
		state.wStartIndicator = float64(minutes_left) * 1.2
		return
	case minutes_since_last_load < 50:
		state.wStartIndicator = float64(minutes_left)
		return
	case minutes_since_last_load < 80:
		state.wStartIndicator = float64(minutes_left) * 0.9
		return
	default:
		state.wStartIndicator = float64(minutes_left) * 0.8
		return
	}
}

func (agent *ReflexAgent) setHStartIndicator(state *ReflexState)(){
/*
	percept := state.percept
	// divide case w_pump = true and w_pump = false

	// energy potential = kettle temperature x kettle delta

	energy_requirement := state.hForeReverseDiffDelta * state.circulationValue
	energy_potential := float64(percept.KettleTemp.GetValue() - percept.HReverseRunTemp.GetValue())

	energy_potential / sta

	state.hReverseDelta * state.circulationValue
*/
	return
}

func (agent *ReflexAgent) GetWPumpThrottle(*system.Action)(float64) {
	return 32.0
}
func (agent *ReflexAgent) GetHPumpThrottle(*system.Action)(float64) {
	return 31.0
}
func (agent *ReflexAgent) GetBurnerState(*system.Action)(bool) {
	return true
}
func initAction()(*ReflexAction){
	return &ReflexAction{&system.Action{
		//RollOut:DefaultRollOut
		}}
}
