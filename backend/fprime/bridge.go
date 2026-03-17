// Package fprime provides the F Prime GDS bridge for SwarmC2.
// It defines drone state types, manages a fleet of simulated drones,
// and exposes an interface that mirrors a real F Prime Ground Data System.
//
// The FSM, energy model, and control pipeline are derived from:
//   Nelson, G. "Autonomous Space UAVs: Embedded Systems Architecture
//   for Intergalactic Drones with NASA F Prime and Cloud-Native Ground
//   Control Systems." Oxford ESS Assignment, 2026.
package fprime

import (
	"sync"
	"time"
)

// ============================================================
// FSM Definition — matches paper Appendix B exactly
// M = (S, I, f, s₀, A)
//   S  = {BOOT, CRUISE, EVADE, HOLD, SAFE_MODE}
//   I  = {proximity_alert, proximity_clear, power_drop, formation_signal}
//   s₀ = BOOT
//   A  = {SAFE_MODE}
// ============================================================

// FSM states (S)
const (
	StateBoot     = "BOOT"
	StateCruise   = "CRUISE"
	StateEvade    = "EVADE"
	StateHold     = "HOLD"
	StateSafeMode = "SAFE_MODE"
)

// FSM inputs (I) — 4 inputs per paper Appendix B
const (
	InputProximityAlert  = "proximity_alert"
	InputProximityClear  = "proximity_clear"
	InputPowerDrop       = "power_drop"
	InputFormationSignal = "formation_signal"
)

// AllStates lists every FSM state for completeness verification.
var AllStates = []string{StateBoot, StateCruise, StateEvade, StateHold, StateSafeMode}

// AllInputs lists every FSM input for completeness verification.
var AllInputs = []string{
	InputProximityAlert, InputProximityClear, InputPowerDrop, InputFormationSignal,
}

// TransitionTable defines the complete FSM: f(state, input) → nextState.
// Per Theorem 2 (FSM Completeness), every (state, input) pair must have a
// defined transition. 5 states × 4 inputs = 20 pairs, all defined.
//
// Table from paper Appendix B:
//
//	              proximity_alert  proximity_clear  power_drop  formation_signal
//	BOOT          SAFE_MODE        CRUISE           SAFE_MODE   CRUISE
//	CRUISE        EVADE            CRUISE           SAFE_MODE   CRUISE
//	EVADE         EVADE            HOLD             SAFE_MODE   HOLD
//	HOLD          EVADE            HOLD             SAFE_MODE   CRUISE
//	SAFE_MODE     SAFE_MODE        SAFE_MODE        SAFE_MODE   SAFE_MODE
var TransitionTable = map[string]map[string]string{
	StateBoot: {
		InputProximityAlert:  StateSafeMode,
		InputProximityClear:  StateCruise,
		InputPowerDrop:       StateSafeMode,
		InputFormationSignal: StateCruise,
	},
	StateCruise: {
		InputProximityAlert:  StateEvade,
		InputProximityClear:  StateCruise,
		InputPowerDrop:       StateSafeMode,
		InputFormationSignal: StateCruise,
	},
	StateEvade: {
		InputProximityAlert:  StateEvade,
		InputProximityClear:  StateHold,
		InputPowerDrop:       StateSafeMode,
		InputFormationSignal: StateHold,
	},
	StateHold: {
		InputProximityAlert:  StateEvade,
		InputProximityClear:  StateHold,
		InputPowerDrop:       StateSafeMode,
		InputFormationSignal: StateCruise,
	},
	StateSafeMode: {
		InputProximityAlert:  StateSafeMode,
		InputProximityClear:  StateSafeMode,
		InputPowerDrop:       StateSafeMode,
		InputFormationSignal: StateSafeMode,
	},
}

// MooreOutputs define the output determined solely by state (Section 4.2).
// Moore machine: output = g(state)
var MooreOutputs = map[string]string{
	StateBoot:     "INITIALIZING",
	StateCruise:   "GREEN_BEACON",
	StateEvade:    "RED_BEACON",
	StateHold:     "YELLOW_BEACON",
	StateSafeMode: "EMERGENCY_BEACON",
}

// MealyOutputs define outputs determined by (state, input) transitions (Section 4.2).
// Mealy machine: output = h(state, input)
var MealyOutputs = map[string]map[string]string{
	StateCruise: {
		InputProximityAlert: "ENGAGE_EVASION",
	},
	StateEvade: {
		InputProximityClear:  "RESUME_HOLD",
		InputFormationSignal: "REJOIN_FORMATION",
	},
	StateHold: {
		InputFormationSignal: "RESUME_CRUISE",
	},
}

// ============================================================
// Control Pipeline (Section 4, Figure in paper)
// sensor read → state estimation → control computation → actuator command → ground control
// ============================================================

// ControlPipelineStage represents one stage of the embedded control loop.
type ControlPipelineStage struct {
	Name      string  `json:"name"`
	RateHz    float64 `json:"rateHz"`
	WCET_us   float64 `json:"wcet_us"`   // Worst-Case Execution Time in microseconds
	Priority  int     `json:"priority"`   // RMS priority (lower = higher priority)
}

// ControlPipeline defines the drone's real-time task schedule (Section 6).
// Rates from paper: 400Hz attitude, 100Hz guidance, 10Hz navigation/health.
var ControlPipeline = []ControlPipelineStage{
	{"AttitudeControl", 400, 500, 1},        // 400 Hz — highest priority
	{"GuidanceLoop", 100, 1200, 2},          // 100 Hz
	{"NavigationUpdate", 10, 5000, 3},       // 10 Hz
	{"HealthMonitoring", 10, 2000, 4},       // 10 Hz
	{"ObjectDetection", 10, 8000, 5},        // 10 Hz — range check at 5m
	{"TelemetryDownlink", 1, 3000, 6},       // 1 Hz — ground control link
}

// ============================================================
// Sensor Interfaces (Section 5, Tables 6-7)
// ============================================================

// SensorInterface represents a hardware sensor bus type.
type SensorInterface struct {
	Component string `json:"component"`
	Part      string `json:"part"`
	Protocol  string `json:"protocol"`  // SPI, I2C, UART, Analog→ADC
	Signal    string `json:"signal"`    // Digital, Analog, RF
	RateHz    int    `json:"rateHz"`
	Status    string `json:"status"`
}

// DefaultSensors defines the drone's sensor suite per Tables 6-7.
var DefaultSensors = []SensorInterface{
	{"6DoF IMU", "ISM330DHCX", "SPI/I2C", "Digital", 400, "NOMINAL"},
	{"Atmospheric", "BME280", "I2C/SPI", "Digital", 10, "NOMINAL"},
	{"GPS/GNSS", "NEO-M9N", "I2C/UART", "Digital", 10, "NOMINAL"},
	{"LiDAR Rangefinder", "LIDAR-Lite v3", "I2C/PWM", "Digital", 100, "NOMINAL"},
	{"Battery Current", "INA169", "Analog→ADC", "Analog", 10, "NOMINAL"},
	{"Solar Current", "ACS723", "Analog→ADC", "Analog", 10, "NOMINAL"},
}

// ============================================================
// Communication Links (Section 5.3, Table 8)
// S-band ground link + UHF mesh + SpaceWire inter-drone
// ============================================================

// CommLink represents a communication channel.
type CommLink struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`       // S-band, UHF, SpaceWire
	Bandwidth string  `json:"bandwidth"`
	Protocol  string  `json:"protocol"`   // CCSDS, beacon, SpaceWire
	State     string  `json:"state"`
	RSSI      float64 `json:"rssi"`
}

// DefaultCommLinks defines the drone's communication interfaces per Table 8.
var DefaultCommLinks = []CommLink{
	{"S-Band Ground", "S-Band", "2 Mbps", "CCSDS", "NOMINAL", -45.0},
	{"UHF Backup", "UHF", "9.6 kbps", "Beacon", "NOMINAL", -55.0},
	{"Inter-Drone Mesh", "SpaceWire", "200 Mbps", "SpaceWire", "NOMINAL", 0},
}

// ============================================================
// Watchdog Timer (Section 6.3)
// ============================================================

// WatchdogState tracks the software watchdog.
type WatchdogState struct {
	TimeoutSec     int   `json:"timeoutSec"`
	LastKickTime   int64 `json:"lastKickTime"`   // Unix timestamp
	TicksSinceKick int   `json:"ticksSinceKick"`
	Triggered      bool  `json:"triggered"`
}

// ============================================================
// Energy Model (Section 8.1, Appendix A)
// Per-cycle energy: E(C) = Σεᵢ(C)
// Budget: B ≤ (Ps + Pb) · T
// ============================================================

// EnergyModelParams holds the paper's energy invariance parameters.
type EnergyModelParams struct {
	Ps           float64 `json:"ps"`           // Solar power generation (W) — 12W
	Pb           float64 `json:"pb"`           // Battery discharge capacity (W) — 8W
	T            float64 `json:"t"`            // Control cycle period (s) — 0.0025s (400Hz)
	MaxPerCycle  float64 `json:"maxPerCycle"`  // (Ps+Pb)·T = 0.050 J
	B            float64 `json:"b"`            // Budget with 10% reserve = 0.045 J
	SafetyMargin float64 `json:"safetyMargin"` // 0.10 (10%)
}

// DefaultEnergyModel returns the paper's reference energy parameters (Appendix A).
var DefaultEnergyModel = EnergyModelParams{
	Ps:           12.0,
	Pb:           8.0,
	T:            0.0025,
	MaxPerCycle:  0.050,       // (12+8) × 0.0025
	B:            0.045,       // 0.9 × 0.050
	SafetyMargin: 0.10,
}

// TaskEnergyCost represents a single active task's worst-case energy per cycle.
type TaskEnergyCost struct {
	Name     string  `json:"name"`
	Epsilon  float64 `json:"epsilon"`   // εᵢ in joules per cycle
	Active   bool    `json:"active"`
}

// BaselineConfiguration C₀ from paper Appendix A:
// attitude control (ε₁=0.018J), ground control encoding (ε₂=0.008J), health monitoring (ε₃=0.006J)
var BaselineConfiguration = []TaskEnergyCost{
	{"AttitudeControl", 0.018, true},
	{"GroundControlEncoding", 0.008, true},
	{"HealthMonitoring", 0.006, true},
}

// ============================================================
// Drone State Types
// ============================================================

// DronePosition represents a 3D geographic position.
type DronePosition struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// DroneVelocity represents velocity in NED frame.
type DroneVelocity struct {
	Vx float64 `json:"vx"`
	Vy float64 `json:"vy"`
	Vz float64 `json:"vz"`
}

// EnergyBudget tracks power state and invariance.
type EnergyBudget struct {
	BatteryPercent     float64          `json:"batteryPercent"`
	SolarInputWatts    float64          `json:"solarInputWatts"`
	PowerDrawWatts     float64          `json:"powerDrawWatts"`
	EstimatedEndurance float64          `json:"estimatedEndurance"`
	BudgetLimit        float64          `json:"budgetLimit"`        // B in Wh for display
	CurrentExpenditure float64          `json:"currentExpenditure"` // Running total Wh
	PerCycleEnergy     float64          `json:"perCycleEnergy"`     // E(C) in joules
	BudgetPerCycle     float64          `json:"budgetPerCycle"`     // B in joules (0.045)
	MaxPerCycle        float64          `json:"maxPerCycle"`        // (Ps+Pb)·T in joules (0.050)
	ActiveTasks        []TaskEnergyCost `json:"activeTasks"`
}

// DroneEvent represents a timestamped event from a drone.
type DroneEvent struct {
	Timestamp time.Time `json:"timestamp"`
	DroneID   string    `json:"droneId"`
	Severity  string    `json:"severity"`  // "info", "warning", "critical"
	Category  string    `json:"category"`  // "fsm", "power", "proximity", "comms", "health", "watchdog", "scheduling"
	Message   string    `json:"message"`
}

// DroneState holds the complete state of a single drone.
type DroneState struct {
	DroneID       string            `json:"droneId"`
	Callsign      string            `json:"callsign"`
	FSMState      string            `json:"fsmState"`
	MooreOutput   string            `json:"mooreOutput"`
	Position      DronePosition     `json:"position"`
	Velocity      DroneVelocity     `json:"velocity"`
	Heading       float64           `json:"heading"`
	Energy        EnergyBudget      `json:"energy"`
	ThreatLevel   string            `json:"threatLevel"`
	LinkState     string            `json:"linkState"`
	RSSI          float64           `json:"rssi"`
	Timestamp     time.Time         `json:"timestamp"`
	Sensors       []SensorInterface `json:"sensors"`
	CommLinks     []CommLink        `json:"commLinks"`
	Watchdog      WatchdogState     `json:"watchdog"`
	Pipeline      []ControlPipelineStage `json:"pipeline"`
	SchedulingHz  map[string]float64     `json:"schedulingHz"`
}

// DroneConfig represents a remote configuration push.
type DroneConfig struct {
	DroneID              string           `json:"droneId"`
	EnergyBudgetLimit    float64          `json:"energyBudgetLimit"`
	SafetyRadius         float64          `json:"safetyRadius"`
	MaxSpeed             float64          `json:"maxSpeed"`
	MaxAltitude          float64          `json:"maxAltitude"`
	CriticalBatteryPct   float64          `json:"criticalBatteryPct"`
	WatchdogTimeoutSec   int              `json:"watchdogTimeoutSec"`
	DownlinkRateHz       int              `json:"downlinkRateHz"`
	ProposedTasks        []TaskEnergyCost `json:"proposedTasks,omitempty"`
}

// ============================================================
// Fleet Manager
// ============================================================

// Fleet manages the state of all drones.
type Fleet struct {
	mu     sync.RWMutex
	drones map[string]*DroneState
	events []DroneEvent
}

// NewFleet creates a new fleet manager.
func NewFleet() *Fleet {
	return &Fleet{
		drones: make(map[string]*DroneState),
		events: make([]DroneEvent, 0, 1000),
	}
}

// UpdateDrone sets the state for a drone (thread-safe).
func (f *Fleet) UpdateDrone(state *DroneState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drones[state.DroneID] = state
}

// GetDrone retrieves a single drone's state.
func (f *Fleet) GetDrone(id string) *DroneState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if d, ok := f.drones[id]; ok {
		cp := *d
		return &cp
	}
	return nil
}

// GetAllDrones returns a snapshot of all drone states.
func (f *Fleet) GetAllDrones() []DroneState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make([]DroneState, 0, len(f.drones))
	for _, d := range f.drones {
		result = append(result, *d)
	}
	return result
}

// AddEvent records a drone event (capped at 1000 events).
func (f *Fleet) AddEvent(evt DroneEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, evt)
	if len(f.events) > 1000 {
		f.events = f.events[len(f.events)-500:]
	}
}

// GetEvents returns recent events for a drone (up to limit).
func (f *Fleet) GetEvents(droneID string, limit int) []DroneEvent {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var result []DroneEvent
	for i := len(f.events) - 1; i >= 0 && len(result) < limit; i-- {
		if f.events[i].DroneID == droneID || droneID == "" {
			result = append(result, f.events[i])
		}
	}
	return result
}

// FSMTransition applies an input to a drone's FSM and returns (newState, transitioned).
func FSMTransition(currentState, input string) (string, bool) {
	stateTransitions, ok := TransitionTable[currentState]
	if !ok {
		return currentState, false
	}
	next, ok := stateTransitions[input]
	if !ok {
		return currentState, false
	}
	return next, next != currentState
}

// GetMooreOutput returns the Moore machine output for a given state.
func GetMooreOutput(state string) string {
	if output, ok := MooreOutputs[state]; ok {
		return output
	}
	return "UNKNOWN"
}

// GetMealyOutput returns the Mealy machine output for a (state, input) pair, if any.
func GetMealyOutput(state, input string) string {
	if transitions, ok := MealyOutputs[state]; ok {
		if output, found := transitions[input]; found {
			return output
		}
	}
	return ""
}

// ComputePerCycleEnergy calculates E(C) = Σεᵢ(C) for active tasks.
func ComputePerCycleEnergy(tasks []TaskEnergyCost) float64 {
	total := 0.0
	for _, t := range tasks {
		if t.Active {
			total += t.Epsilon
		}
	}
	return total
}
