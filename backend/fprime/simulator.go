// Package fprime simulator provides mock telemetry generation.
// Simulates 3 drones in formation flight with ESS paper-accurate behavior:
//   - FSM with 4 inputs (proximity_alert, proximity_clear, power_drop, formation_signal)
//   - Control pipeline at 400Hz/100Hz/10Hz scheduling rates
//   - Per-cycle energy model E(C) = Σεᵢ(C) with invariance enforcement
//   - Dual-band communication (S-band ground + UHF mesh + SpaceWire inter-drone)
//   - Watchdog timer with configurable timeout
//   - Sensor interface simulation (SPI/I2C/UART)
package fprime

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// SimConfig controls simulator behavior.
type SimConfig struct {
	TickInterval time.Duration
	NumDrones    int
}

// DefaultSimConfig returns standard simulator settings.
func DefaultSimConfig() SimConfig {
	return SimConfig{
		TickInterval: 1 * time.Second,
		NumDrones:    3,
	}
}

// Simulator generates mock drone telemetry.
type Simulator struct {
	fleet    *Fleet
	config   SimConfig
	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
	tickNum  int
	rng      *rand.Rand
	simState map[string]*droneSimState
}

type droneSimState struct {
	baseLatitude  float64
	baseLongitude float64
	baseAltitude  float64
	orbitRadius   float64
	orbitSpeed    float64
	angle         float64
	bootTick      int
	watchdogKick  int // tick when watchdog was last kicked
}

// NewSimulator creates a new telemetry simulator.
func NewSimulator(fleet *Fleet, config SimConfig) *Simulator {
	s := &Simulator{
		fleet:    fleet,
		config:   config,
		stopCh:   make(chan struct{}),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		simState: make(map[string]*droneSimState),
	}
	s.initDrones()
	return s
}

func (s *Simulator) initDrones() {
	// Formation center: over SoCal test range
	centerLat := 34.05
	centerLon := -118.25

	drones := []struct {
		id       string
		callsign string
		offset   float64
	}{
		{"DRONE-001", "ALPHA-1", 0},
		{"DRONE-002", "BETA-2", 120},
		{"DRONE-003", "GAMMA-3", 240},
	}

	for _, d := range drones {
		// Deep copy sensor and comm link slices
		sensors := make([]SensorInterface, len(DefaultSensors))
		copy(sensors, DefaultSensors)
		commLinks := make([]CommLink, len(DefaultCommLinks))
		copy(commLinks, DefaultCommLinks)
		activeTasks := make([]TaskEnergyCost, len(BaselineConfiguration))
		copy(activeTasks, BaselineConfiguration)
		pipeline := make([]ControlPipelineStage, len(ControlPipeline))
		copy(pipeline, ControlPipeline)

		state := &DroneState{
			DroneID:     d.id,
			Callsign:    d.callsign,
			FSMState:    StateBoot,
			MooreOutput: GetMooreOutput(StateBoot),
			Position: DronePosition{
				Latitude:  centerLat,
				Longitude: centerLon,
				Altitude:  100.0,
			},
			Velocity:    DroneVelocity{},
			Heading:     0,
			ThreatLevel: "NONE",
			LinkState:   "NOMINAL",
			RSSI:        -45.0,
			Energy: EnergyBudget{
				BatteryPercent:     95.0 + s.rng.Float64()*5.0,
				SolarInputWatts:    DefaultEnergyModel.Ps + s.rng.Float64()*3.0,
				PowerDrawWatts:     25.0,
				EstimatedEndurance: 120.0,
				BudgetLimit:        500.0,
				CurrentExpenditure: 0.0,
				PerCycleEnergy:     ComputePerCycleEnergy(activeTasks),
				BudgetPerCycle:     DefaultEnergyModel.B,
				MaxPerCycle:        DefaultEnergyModel.MaxPerCycle,
				ActiveTasks:        activeTasks,
			},
			Timestamp: time.Now(),
			Sensors:   sensors,
			CommLinks: commLinks,
			Watchdog: WatchdogState{
				TimeoutSec:     10,
				LastKickTime:   time.Now().Unix(),
				TicksSinceKick: 0,
				Triggered:      false,
			},
			Pipeline: pipeline,
			SchedulingHz: map[string]float64{
				"AttitudeControl":   400,
				"GuidanceLoop":      100,
				"NavigationUpdate":  10,
				"HealthMonitoring":  10,
				"ObjectDetection":   10,
				"TelemetryDownlink": 1,
			},
		}
		s.fleet.UpdateDrone(state)

		s.simState[d.id] = &droneSimState{
			baseLatitude:  centerLat,
			baseLongitude: centerLon,
			baseAltitude:  200.0 + float64(len(s.simState))*50.0,
			orbitRadius:   0.01 + float64(len(s.simState))*0.005,
			orbitSpeed:    0.02,
			angle:         d.offset * math.Pi / 180.0,
			bootTick:      3 + s.rng.Intn(3),
			watchdogKick:  0,
		}
	}
}

// Start begins the simulation loop.
func (s *Simulator) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.run()
}

// Stop halts the simulation.
func (s *Simulator) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

func (s *Simulator) run() {
	ticker := time.NewTicker(s.config.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Simulator) tick() {
	s.tickNum++

	for _, drone := range s.fleet.GetAllDrones() {
		ss := s.simState[drone.DroneID]
		if ss == nil {
			continue
		}
		d := drone // copy to avoid loop variable pointer reuse
		s.updateDrone(&d, ss)
		s.fleet.UpdateDrone(&d)
	}
}

func (s *Simulator) updateDrone(d *DroneState, ss *droneSimState) {
	d.Timestamp = time.Now()

	// FSM transitions using paper's 4 inputs
	s.updateFSM(d, ss)

	// Update Moore output based on current state
	d.MooreOutput = GetMooreOutput(d.FSMState)

	// Position update (orbit pattern when cruising)
	s.updatePosition(d, ss)

	// Energy simulation with per-cycle model
	s.updateEnergy(d, ss)

	// Sensor status simulation
	s.updateSensors(d)

	// Communication link simulation (S-band + UHF + SpaceWire)
	s.updateCommLinks(d)

	// Watchdog timer (Section 6.3)
	s.updateWatchdog(d, ss)

	// Periodic events
	s.generateEvents(d, ss)

	// Link quality from comm links
	s.updateLinkState(d)
}

func (s *Simulator) updateFSM(d *DroneState, ss *droneSimState) {
	oldState := d.FSMState

	switch d.FSMState {
	case StateBoot:
		// Boot completes → proximity_clear triggers transition to CRUISE
		if s.tickNum >= ss.bootTick {
			d.FSMState, _ = FSMTransition(d.FSMState, InputProximityClear)
		}

	case StateCruise:
		// Periodic proximity events (~every 30 ticks, 40% chance)
		if s.tickNum%30 == 0 && s.rng.Float64() < 0.4 {
			mealyOut := GetMealyOutput(d.FSMState, InputProximityAlert)
			d.FSMState, _ = FSMTransition(d.FSMState, InputProximityAlert)
			d.ThreatLevel = "HIGH"
			if mealyOut != "" {
				s.fleet.AddEvent(DroneEvent{
					Timestamp: time.Now(),
					DroneID:   d.DroneID,
					Severity:  "warning",
					Category:  "fsm",
					Message:   fmt.Sprintf("Mealy output: %s", mealyOut),
				})
			}
		}
		// Power drop check
		if d.Energy.BatteryPercent < 15.0 {
			d.FSMState, _ = FSMTransition(d.FSMState, InputPowerDrop)
		}

	case StateEvade:
		// Evasion lasts ~8 ticks, then proximity_clear → HOLD (per paper's table)
		if s.tickNum%8 == 0 {
			d.FSMState, _ = FSMTransition(d.FSMState, InputProximityClear)
			d.ThreatLevel = "NONE"
		}
		// Power drop overrides evasion
		if d.Energy.BatteryPercent < 10.0 {
			d.FSMState, _ = FSMTransition(d.FSMState, InputPowerDrop)
		}

	case StateHold:
		// Formation signal resumes cruise (per paper's table: HOLD + formation_signal → CRUISE)
		if s.tickNum%15 == 0 {
			mealyOut := GetMealyOutput(d.FSMState, InputFormationSignal)
			d.FSMState, _ = FSMTransition(d.FSMState, InputFormationSignal)
			if mealyOut != "" {
				s.fleet.AddEvent(DroneEvent{
					Timestamp: time.Now(),
					DroneID:   d.DroneID,
					Severity:  "info",
					Category:  "fsm",
					Message:   fmt.Sprintf("Mealy output: %s", mealyOut),
				})
			}
		}
		// Power drop
		if d.Energy.BatteryPercent < 10.0 {
			d.FSMState, _ = FSMTransition(d.FSMState, InputPowerDrop)
		}

	case StateSafeMode:
		// SAFE_MODE is absorbing — all inputs map to SAFE_MODE (per paper)
		// In simulation, slowly recover battery and eventually restart
		if d.Energy.BatteryPercent > 50.0 && s.tickNum%60 == 0 {
			// Manual reset scenario — re-initialize to BOOT
			d.FSMState = StateBoot
			ss.bootTick = s.tickNum + 3
			s.fleet.AddEvent(DroneEvent{
				Timestamp: time.Now(),
				DroneID:   d.DroneID,
				Severity:  "info",
				Category:  "fsm",
				Message:   "Manual reset: SAFE_MODE → BOOT (operator intervention)",
			})
		}
	}

	if d.FSMState != oldState {
		s.fleet.AddEvent(DroneEvent{
			Timestamp: time.Now(),
			DroneID:   d.DroneID,
			Severity:  fsmEventSeverity(d.FSMState),
			Category:  "fsm",
			Message:   fmt.Sprintf("FSM: %s → %s (Moore: %s)", oldState, d.FSMState, GetMooreOutput(d.FSMState)),
		})
	}
}

func (s *Simulator) updatePosition(d *DroneState, ss *droneSimState) {
	if d.FSMState == StateBoot || d.FSMState == StateSafeMode {
		return // stationary
	}

	ss.angle += ss.orbitSpeed
	if ss.angle > 2*math.Pi {
		ss.angle -= 2 * math.Pi
	}

	d.Position.Latitude = ss.baseLatitude + ss.orbitRadius*math.Cos(ss.angle)
	d.Position.Longitude = ss.baseLongitude + ss.orbitRadius*math.Sin(ss.angle)
	d.Position.Altitude = ss.baseAltitude + 10.0*math.Sin(ss.angle*2)

	// Velocity from position derivatives
	speed := ss.orbitSpeed * ss.orbitRadius * 111000 // rough m/s
	d.Velocity.Vx = speed * math.Cos(ss.angle+math.Pi/2)
	d.Velocity.Vy = speed * math.Sin(ss.angle+math.Pi/2)
	d.Velocity.Vz = 0.5 * math.Sin(ss.angle*2)

	// Heading from velocity
	d.Heading = math.Mod(math.Atan2(d.Velocity.Vy, d.Velocity.Vx)*180/math.Pi+360, 360)

	// Evasion adds lateral offset
	if d.FSMState == StateEvade {
		d.Position.Latitude += 0.002 * math.Sin(float64(s.tickNum))
		d.Position.Altitude += 20.0
	}
}

func (s *Simulator) updateEnergy(d *DroneState, ss *droneSimState) {
	// Power draw varies by state (mapped to paper's model)
	switch d.FSMState {
	case StateBoot:
		d.Energy.PowerDrawWatts = 15.0
	case StateCruise:
		d.Energy.PowerDrawWatts = 25.0 + s.rng.Float64()*5.0
	case StateEvade:
		d.Energy.PowerDrawWatts = 40.0 + s.rng.Float64()*10.0
	case StateHold:
		d.Energy.PowerDrawWatts = 18.0
	case StateSafeMode:
		d.Energy.PowerDrawWatts = 8.0
	}

	// Solar fluctuation (Ps ≈ 12W nominal per paper)
	d.Energy.SolarInputWatts = DefaultEnergyModel.Ps + 5.0*math.Sin(float64(s.tickNum)*0.1) + s.rng.Float64()*2.0
	if d.Energy.SolarInputWatts < 0 {
		d.Energy.SolarInputWatts = 0
	}

	// Per-cycle energy: E(C) = Σεᵢ(C) — recompute each tick
	d.Energy.PerCycleEnergy = ComputePerCycleEnergy(d.Energy.ActiveTasks)
	d.Energy.BudgetPerCycle = DefaultEnergyModel.B
	d.Energy.MaxPerCycle = DefaultEnergyModel.MaxPerCycle

	// Net power flow
	netDraw := d.Energy.PowerDrawWatts - d.Energy.SolarInputWatts
	energyDelta := netDraw / 3600.0 // Wh per second

	d.Energy.CurrentExpenditure += math.Abs(energyDelta)
	d.Energy.BatteryPercent -= energyDelta * 0.1
	if d.Energy.BatteryPercent > 100 {
		d.Energy.BatteryPercent = 100
	}
	if d.Energy.BatteryPercent < 0 {
		d.Energy.BatteryPercent = 0
	}

	// Endurance estimate
	if netDraw > 0 {
		remainingWh := d.Energy.BatteryPercent / 100.0 * d.Energy.BudgetLimit
		d.Energy.EstimatedEndurance = (remainingWh / netDraw) * 60.0
	} else {
		d.Energy.EstimatedEndurance = 999.0 // charging
	}

	// Energy invariance warning event
	if d.Energy.PerCycleEnergy > d.Energy.BudgetPerCycle*0.9 && s.tickNum%10 == 0 {
		s.fleet.AddEvent(DroneEvent{
			Timestamp: time.Now(),
			DroneID:   d.DroneID,
			Severity:  "warning",
			Category:  "power",
			Message:   fmt.Sprintf("Energy margin low: E(C)=%.4f J approaching B=%.4f J", d.Energy.PerCycleEnergy, d.Energy.BudgetPerCycle),
		})
	}
}

func (s *Simulator) updateSensors(d *DroneState) {
	// Occasional sensor degradation
	for i := range d.Sensors {
		if s.rng.Float64() < 0.005 { // 0.5% chance per tick
			d.Sensors[i].Status = "DEGRADED"
		} else if d.Sensors[i].Status == "DEGRADED" && s.rng.Float64() < 0.2 {
			d.Sensors[i].Status = "NOMINAL"
		}
	}
}

func (s *Simulator) updateCommLinks(d *DroneState) {
	for i := range d.CommLinks {
		switch d.CommLinks[i].Type {
		case "S-Band":
			// S-band ground link — primary CCSDS channel
			d.CommLinks[i].RSSI = -45.0 + s.rng.Float64()*15.0 - 7.5
			if s.rng.Float64() < 0.02 {
				d.CommLinks[i].State = "DEGRADED"
				d.CommLinks[i].RSSI = -75.0 + s.rng.Float64()*10.0
			} else if d.CommLinks[i].State == "DEGRADED" && s.rng.Float64() < 0.3 {
				d.CommLinks[i].State = "NOMINAL"
			}
		case "UHF":
			// UHF backup — beacon mode
			d.CommLinks[i].RSSI = -55.0 + s.rng.Float64()*10.0 - 5.0
			if d.CommLinks[0].State == "DEGRADED" {
				d.CommLinks[i].State = "ACTIVE" // UHF activates when S-band degrades
			} else {
				d.CommLinks[i].State = "STANDBY"
			}
		case "SpaceWire":
			// Inter-drone mesh — always active in formation
			if d.FSMState == StateCruise || d.FSMState == StateHold {
				d.CommLinks[i].State = "ACTIVE"
			} else {
				d.CommLinks[i].State = "NOMINAL"
			}
		}
	}
}

func (s *Simulator) updateWatchdog(d *DroneState, ss *droneSimState) {
	// Kick watchdog every tick (simulating healthy operation)
	d.Watchdog.TicksSinceKick++

	// In normal operation, software kicks the watchdog regularly
	if d.FSMState != StateSafeMode {
		d.Watchdog.LastKickTime = time.Now().Unix()
		d.Watchdog.TicksSinceKick = 0
		d.Watchdog.Triggered = false
		ss.watchdogKick = s.tickNum
	}

	// Simulate rare watchdog timeout (only in SAFE_MODE if stuck too long)
	if d.Watchdog.TicksSinceKick > d.Watchdog.TimeoutSec && !d.Watchdog.Triggered {
		d.Watchdog.Triggered = true
		s.fleet.AddEvent(DroneEvent{
			Timestamp: time.Now(),
			DroneID:   d.DroneID,
			Severity:  "critical",
			Category:  "watchdog",
			Message:   fmt.Sprintf("Watchdog timeout after %d ticks — hardware reset initiated", d.Watchdog.TicksSinceKick),
		})
	}
}

func (s *Simulator) generateEvents(d *DroneState, ss *droneSimState) {
	// Periodic comms quality report
	if s.tickNum%20 == 0 {
		sBandState := "UNKNOWN"
		for _, cl := range d.CommLinks {
			if cl.Type == "S-Band" {
				sBandState = cl.State
			}
		}
		s.fleet.AddEvent(DroneEvent{
			Timestamp: time.Now(),
			DroneID:   d.DroneID,
			Severity:  "info",
			Category:  "comms",
			Message:   fmt.Sprintf("S-Band: %s | UHF: standby | Mesh: active", sBandState),
		})
	}

	// Scheduling report (control pipeline rates)
	if s.tickNum%30 == 0 {
		s.fleet.AddEvent(DroneEvent{
			Timestamp: time.Now(),
			DroneID:   d.DroneID,
			Severity:  "info",
			Category:  "scheduling",
			Message:   "Pipeline: 400Hz attitude | 100Hz guidance | 10Hz nav/health | 1Hz downlink",
		})
	}

	// Health check events
	if s.tickNum%25 == 0 {
		degradedSensors := 0
		for _, sen := range d.Sensors {
			if sen.Status == "DEGRADED" {
				degradedSensors++
			}
		}
		msg := "All subsystems nominal"
		sev := "info"
		if degradedSensors > 0 {
			msg = fmt.Sprintf("%d sensor(s) degraded — monitoring", degradedSensors)
			sev = "warning"
		}
		s.fleet.AddEvent(DroneEvent{
			Timestamp: time.Now(),
			DroneID:   d.DroneID,
			Severity:  sev,
			Category:  "health",
			Message:   msg,
		})
	}

	// Energy invariance check event
	if s.tickNum%20 == 0 {
		eC := d.Energy.PerCycleEnergy
		b := d.Energy.BudgetPerCycle
		status := "PASS"
		sev := "info"
		if eC > b {
			status = "FAIL"
			sev = "critical"
		} else if eC > b*0.9 {
			status = "WARN"
			sev = "warning"
		}
		s.fleet.AddEvent(DroneEvent{
			Timestamp: time.Now(),
			DroneID:   d.DroneID,
			Severity:  sev,
			Category:  "power",
			Message:   fmt.Sprintf("Thm1 [%s]: E(C)=%.4fJ ≤ B=%.4fJ ≤ %.4fJ", status, eC, b, d.Energy.MaxPerCycle),
		})
	}
}

func (s *Simulator) updateLinkState(d *DroneState) {
	// Aggregate link state from S-band primary
	for _, cl := range d.CommLinks {
		if cl.Type == "S-Band" {
			d.LinkState = cl.State
			d.RSSI = cl.RSSI
			return
		}
	}
}

func fsmEventSeverity(state string) string {
	switch state {
	case StateSafeMode:
		return "critical"
	case StateEvade:
		return "warning"
	default:
		return "info"
	}
}
