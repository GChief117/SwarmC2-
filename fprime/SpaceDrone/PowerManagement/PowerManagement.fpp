@ Power Management Component
@ Monitors energy state and enforces the Energy Invariance property.
@ Reference: Theorem 1 (Energy Invariance)
@ Energy expenditure never exceeds the budget limit.
@
@ The component continuously tracks:
@   - Battery state of charge
@   - Solar panel input
@   - Total power draw across subsystems
@   - Running energy expenditure
@ And raises ENERGY_CRITICAL when expenditure approaches the budget limit.

module SpaceDrone {

  @ Power management and energy invariance enforcement
  active component PowerManagement {

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
    # Commands
    # ------------------------------------------------------------------

    @ Set the energy budget limit B
    async command SET_BUDGET(
      budget: F64  @< Maximum allowed cumulative energy (Wh)
    ) opcode 0x5001

    @ Reset energy expenditure counter (after recharge)
    async command RESET_EXPENDITURE opcode 0x5002

    @ Enable/disable solar charging
    async command SET_SOLAR_ENABLED(
      enabled: bool
    ) opcode 0x5003

    @ Set low-power mode thresholds
    async command SET_THRESHOLDS(
      criticalPercent: F64,  @< Battery % for CRITICAL alert
      warningPercent: F64    @< Battery % for WARNING alert
    ) opcode 0x5004

    # ------------------------------------------------------------------
    # Ports
    # ------------------------------------------------------------------

    @ Outputs energy budget updates to MissionControl
    output port energyOut: SpaceDrone.EnergyUpdate

    @ Reports health status to HealthMonitor
    output port healthOut: SpaceDrone.HealthReport

    @ Receives FSM state to adjust power profile
    async input port fsmStateIn: SpaceDrone.FSMStateChange

    # ------------------------------------------------------------------
    # Telemetry
    # ------------------------------------------------------------------

    @ Battery state of charge [0..100] %
    telemetry BatteryPercent: F64 id 0x5101

    @ Solar panel output (Watts)
    telemetry SolarInput: F64 id 0x5102

    @ Total system power draw (Watts)
    telemetry PowerDraw: F64 id 0x5103

    @ Estimated remaining endurance (minutes)
    telemetry Endurance: F64 id 0x5104

    @ Cumulative energy expenditure in Wh
    telemetry EnergyExpenditure: F64 id 0x5105

    @ Energy budget limit B in Wh
    telemetry EnergyBudget: F64 id 0x5106

    @ Energy margin: B - E(C') in Wh
    telemetry EnergyMargin: F64 id 0x5107

    @ Battery temperature (°C)
    telemetry BatteryTemp: F64 id 0x5108

    @ Whether energy invariance holds
    telemetry InvarianceHolds: bool id 0x5109

    # ------------------------------------------------------------------
    # Events
    # ------------------------------------------------------------------

    @ Energy expenditure approaching budget limit
    event EnergyWarning(
      expenditure: F64,
      budget: F64,
      marginPercent: F64
    ) severity warning high \
      format "Energy warning: E={} Wh / B={} Wh ({}% margin remaining)"

    @ Energy invariance violation
    event EnergyInvarianceViolation(
      expenditure: F64,
      budget: F64
    ) severity warning high \
      format "INVARIANCE VIOLATION: E(C')={} > B={}"

    @ Battery critically low
    event BatteryCritical(
      percent: F64
    ) severity warning high \
      format "Battery critical: {}%"

    @ Solar charging state changed
    event SolarStateChange(
      inputWatts: F64
    ) severity activity low \
      format "Solar input changed to {} W"

    @ Entered low-power mode
    event LowPowerMode severity warning high \
      format "Entering low-power mode to preserve energy budget"

    # ------------------------------------------------------------------
    # Parameters
    # ------------------------------------------------------------------

    @ Critical battery threshold (%)
    param CRITICAL_THRESHOLD: F64 default 15.0 id 0x5201

    @ Warning battery threshold (%)
    param WARNING_THRESHOLD: F64 default 30.0 id 0x5202

    @ Default energy budget B (Wh)
    param DEFAULT_BUDGET: F64 default 500.0 id 0x5203

    @ Energy margin warning threshold (% of B remaining)
    param MARGIN_WARNING_PERCENT: F64 default 10.0 id 0x5204
  }
}
