@ Space Drone Type Definitions
@ Defines core enumerations, structures, and ports used across all SpaceDrone components.
@ Reference: Nelson, G. "Autonomous Space UAVs" — Oxford ESS Assignment, 2026.
@
@ FSM Definition (Appendix B):
@   M = (S, I, f, s₀, A) where
@   S  = {BOOT, CRUISE, EVADE, HOLD, SAFE_MODE}
@   I  = {proximity_alert, proximity_clear, power_drop, formation_signal}
@   s₀ = BOOT
@   A  = {SAFE_MODE}

module SpaceDrone {

  @ Finite State Machine states for autonomous drone control
  @ Maps to Definition 1 (FSM Model) — 5 states
  enum FSMState {
    BOOT        @< Initial power-on state; subsystem initialization
    CRUISE      @< Nominal flight operations; executing waypoint plan
    EVADE       @< Collision avoidance maneuver active
    HOLD        @< Station-keeping at current position
    SAFE_MODE   @< Minimum-power emergency state; awaiting ground command
  }

  @ Input events that drive FSM transitions
  @ Per paper Appendix B: 4 inputs yielding 5×4 = 20 defined pairs
  enum FSMInput {
    PROXIMITY_ALERT     @< Object detected within safety radius (LiDAR/radar)
    PROXIMITY_CLEAR     @< Threat no longer within safety radius
    POWER_DROP          @< Battery below safe threshold or solar eclipse event
    FORMATION_SIGNAL    @< Swarm coordination signal from inter-drone mesh
  }

  @ Threat classification for detected proximate objects
  enum ThreatLevel {
    NONE      @< No detected threats
    LOW       @< Object detected, no collision risk
    MEDIUM    @< Object on converging trajectory, monitoring
    HIGH      @< Collision possible within 60s, preparing evasion
    CRITICAL  @< Collision imminent, evasion executing
  }

  @ Communication link quality states
  enum LinkState {
    NOMINAL     @< Full duplex, low latency
    DEGRADED    @< Partial packet loss, increased latency
    LOST        @< No uplink/downlink for > timeout period
    RECOVERING  @< Link reacquisition in progress
  }

  @ Subsystem health status
  enum HealthStatus {
    HEALTHY     @< Operating within nominal parameters
    WARNING     @< Degraded but functional
    CRITICAL    @< Failure imminent or partial failure
    OFFLINE     @< Subsystem not responding
  }

  @ Sensor interface protocol types (Section 5, Tables 6-7)
  enum SensorProtocol {
    SPI         @< Serial Peripheral Interface (IMU, atmospheric)
    I2C         @< Inter-Integrated Circuit (temp, pressure, LiDAR)
    UART        @< Universal Asynchronous (GPS, comms)
    ANALOG_ADC  @< Analog sensor via ADC (current, voltage)
  }

  @ Communication band types (Section 5.3, Table 8)
  enum CommBand {
    S_BAND      @< 2 Mbps ground link, CCSDS-compliant, AES-256
    UHF         @< 9.6 kbps backup, beacon mode
    SPACEWIRE   @< High-speed inter-drone mesh link
  }

  @ 3D position in geographic coordinates
  struct DronePosition {
    latitude: F64   @< Degrees, WGS84
    longitude: F64  @< Degrees, WGS84
    altitude: F64   @< Meters above mean sea level
  }

  @ Velocity vector in NED (North-East-Down) frame
  struct DroneVelocity {
    vx: F64  @< North velocity, m/s
    vy: F64  @< East velocity, m/s
    vz: F64  @< Down velocity, m/s
  }

  @ Energy budget tracking for invariance enforcement
  @ Reference: Theorem 1 (Energy Invariance)
  @ E(C') ≤ B ≤ (Ps + Pb) · T
  @ Ps=12W, Pb=8W, T=0.0025s, B=0.045J
  struct EnergyBudget {
    batteryPercent: F64       @< Current state of charge [0..100]
    solarInputWatts: F64      @< Real-time solar panel output (Ps ≈ 12W)
    powerDrawWatts: F64       @< Current total power consumption
    estimatedEndurance: F64   @< Minutes of flight remaining at current draw
    budgetLimit: F64          @< Maximum allowed energy expenditure B (Wh for ops)
    currentExpenditure: F64   @< Running total energy spent E(C') (Wh for ops)
    perCycleEnergy: F64       @< E(C) = Σεᵢ(C) in joules per control cycle
    budgetPerCycle: F64       @< B in joules (0.045 J per Appendix A)
    maxPerCycle: F64          @< (Ps+Pb)·T in joules (0.050 J)
  }

  @ Full telemetry frame aggregating all drone state
  struct DroneTelemetry {
    droneId: string size 32          @< Unique drone identifier
    timestamp: U64                   @< Unix epoch milliseconds
    fsmState: FSMState               @< Current FSM state
    mooreOutput: string size 32      @< Moore machine output g(state)
    position: DronePosition          @< Current 3D position
    velocity: DroneVelocity          @< Current velocity vector
    heading: F64                     @< Magnetic heading, degrees [0..360)
    energy: EnergyBudget             @< Full energy state
    threatLevel: ThreatLevel         @< Current threat assessment
    linkState: LinkState             @< S-Band comms link quality
    rssi: F64                        @< Received signal strength, dBm
  }

  @ FSM transition record for audit trail
  struct FSMTransition {
    fromState: FSMState   @< State before transition
    input: FSMInput       @< Input that triggered transition
    toState: FSMState     @< State after transition
    mooreOutput: string size 32   @< Moore output of new state
    mealyOutput: string size 32   @< Mealy output of transition (if any)
    timestamp: U64        @< When transition occurred
  }

  @ Port for proximity alert notifications between ObjectDetection and MissionControl
  port ProximityAlert(
    threatLevel: ThreatLevel  @< Severity of detected threat
    bearing: F64              @< Bearing to threat, degrees
    range: F64                @< Distance to threat, meters
    closingSpeed: F64         @< Rate of range decrease, m/s
  )

  @ Port for telemetry data distribution
  port TelemetryPort(
    ref telemetry: DroneTelemetry  @< Complete telemetry frame
  )

  @ Port for FSM state change notifications
  port FSMStateChange(
    oldState: FSMState  @< Previous state
    newState: FSMState  @< New state
    input: FSMInput     @< Triggering input
  )

  @ Port for energy budget updates
  port EnergyUpdate(
    ref budget: EnergyBudget  @< Current energy state
  )

  @ Port for health status reporting
  port HealthReport(
    subsystem: string size 32  @< Name of reporting subsystem
    status: HealthStatus       @< Current health
    message: string size 128   @< Diagnostic message
  )
}
