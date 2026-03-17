@ Mission Control Component
@ Implements the core Finite State Machine for autonomous drone operations.
@ Reference: Nelson, G. "Autonomous Space UAVs" — Oxford ESS Assignment, 2026.
@
@ FSM: M = (S, I, f, s₀, A)
@   S  = {BOOT, CRUISE, EVADE, HOLD, SAFE_MODE}
@   I  = {proximity_alert, proximity_clear, power_drop, formation_signal}
@   s₀ = BOOT
@   A  = {SAFE_MODE}
@
@ Transition Table (Appendix B):
@              prox_alert    prox_clear    power_drop    form_signal
@  BOOT        SAFE_MODE     CRUISE        SAFE_MODE     CRUISE
@  CRUISE      EVADE         CRUISE        SAFE_MODE     CRUISE
@  EVADE       EVADE         HOLD          SAFE_MODE     HOLD
@  HOLD        EVADE         HOLD          SAFE_MODE     CRUISE
@  SAFE_MODE   SAFE_MODE     SAFE_MODE     SAFE_MODE     SAFE_MODE
@
@ Moore Outputs: g(state)
@   BOOT → INITIALIZING, CRUISE → GREEN_BEACON, EVADE → RED_BEACON,
@   HOLD → YELLOW_BEACON, SAFE_MODE → EMERGENCY_BEACON
@
@ Theorem 2 (FSM Completeness):
@   5 states × 4 inputs = 20 pairs, all defined. No deadlock possible.

module SpaceDrone {

  @ Central FSM controller managing drone operational states
  active component MissionControl {

    # ------------------------------------------------------------------
    # F Prime standard ports
    # ------------------------------------------------------------------
    command recv port CmdDisp
    command reg port CmdReg
    command resp port CmdStatus
    event port Log
    text event port LogText
    time get port Time
    telemetry port Tlm
    param get port ParamGet
    param set port ParamSet

    # ------------------------------------------------------------------
    # Command interface (ground-to-drone via CCSDS S-band uplink)
    # ------------------------------------------------------------------

    @ Send a new waypoint to the navigation system
    async command SEND_WAYPOINT(
      latitude: F64,   @< Target latitude
      longitude: F64,  @< Target longitude
      altitude: F64    @< Target altitude MSL
    ) opcode 0x1001

    @ Command the drone to hold position
    async command HOLD_POSITION opcode 0x1002

    @ Resume navigation from hold state (sends formation_signal)
    async command RESUME_NAVIGATION opcode 0x1003

    @ Force transition to SAFE_MODE (ground override → power_drop)
    async command EMERGENCY_SAFE_MODE opcode 0x1004

    @ Reset drone FSM to BOOT state (requires SAFE_MODE)
    async command RESET_FSM opcode 0x1005

    @ Update energy budget limit B (Theorem 1 gate applied before activation)
    async command SET_ENERGY_BUDGET(
      budgetLimit: F64  @< New maximum energy budget
    ) opcode 0x1006

    # ------------------------------------------------------------------
    # Port connections
    # ------------------------------------------------------------------

    @ Receives proximity alerts from ObjectDetection
    @ Maps to FSM inputs: proximity_alert / proximity_clear
    async input port proximityIn: SpaceDrone.ProximityAlert

    @ Receives energy updates from PowerManagement
    @ Maps to FSM input: power_drop
    async input port energyIn: SpaceDrone.EnergyUpdate

    @ Receives health reports from HealthMonitor
    async input port healthIn: SpaceDrone.HealthReport

    @ Outputs FSM state changes to all subscribers
    output port fsmStateOut: SpaceDrone.FSMStateChange

    @ Outputs telemetry to Communications for CCSDS downlink
    output port telemetryOut: SpaceDrone.TelemetryPort

    # ------------------------------------------------------------------
    # Telemetry channels
    # ------------------------------------------------------------------

    @ Current FSM state
    telemetry CurrentState: SpaceDrone.FSMState id 0x2001

    @ Moore output for current state
    telemetry MooreOutput: string size 32 id 0x2002

    @ Time in current state (seconds)
    telemetry StateDuration: U32 id 0x2003

    @ Total FSM transitions since boot
    telemetry TransitionCount: U32 id 0x2004

    @ Last input that caused a transition
    telemetry LastInput: SpaceDrone.FSMInput id 0x2005

    @ Whether the FSM transition table is complete (Theorem 2)
    telemetry FSMComplete: bool id 0x2006

    @ Number of defined (state, input) pairs (should be 20)
    telemetry DefinedPairs: U32 id 0x2007

    # ------------------------------------------------------------------
    # Events
    # ------------------------------------------------------------------

    @ FSM state transition occurred
    event StateTransition(
      fromState: SpaceDrone.FSMState,
      toState: SpaceDrone.FSMState,
      trigger: SpaceDrone.FSMInput
    ) severity activity high \
      format "FSM transition: {} --[{}]--> {}"

    @ Moore output changed
    event MooreOutputChanged(
      fsmState: SpaceDrone.FSMState,
      mooreOut: string size 32
    ) severity activity low \
      format "Moore output: {} → {}"

    @ Mealy output generated during transition
    event MealyOutput(
      fromState: SpaceDrone.FSMState,
      trigger: SpaceDrone.FSMInput,
      mealyOut: string size 32
    ) severity activity low \
      format "Mealy output: ({}, {}) → {}"

    @ Drone entered SAFE_MODE (absorbing state per paper)
    event SafeModeEntered(
      reason: string size 64
    ) severity warning high \
      format "SAFE_MODE entered: {}"

    @ Boot sequence completed
    event BootComplete severity activity high \
      format "Boot sequence complete, transitioning to CRUISE"

    @ Invalid transition attempted (should never fire if Theorem 2 holds)
    event InvalidTransition(
      fsmState: SpaceDrone.FSMState,
      trigger: SpaceDrone.FSMInput
    ) severity warning high \
      format "Undefined transition attempted: state={}, input={}"

    @ Ground override received
    event GroundOverride severity warning high \
      format "Ground override command received"

    # ------------------------------------------------------------------
    # Parameters (configurable at runtime via CCSDS command uplink)
    # ------------------------------------------------------------------

    @ Minimum battery percentage before triggering power_drop input
    param ENERGY_CRITICAL_THRESHOLD: F64 default 15.0 id 0x3001

    @ Battery percentage to consider energy restored
    param ENERGY_RESTORE_THRESHOLD: F64 default 30.0 id 0x3002

    @ Maximum time in EVADE before forcing SAFE_MODE (seconds)
    param MAX_EVADE_DURATION: U32 default 120 id 0x3003
  }
}
