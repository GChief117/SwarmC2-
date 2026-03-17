@ Health Monitor Component
@ Watchdog for all subsystems. Receives periodic health reports and
@ triggers WATCHDOG_TIMEOUT if any subsystem goes silent.

module SpaceDrone {

  @ Subsystem health watchdog and aggregated status reporter
  active component HealthMonitor {

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

    @ Set watchdog timeout for a subsystem
    async command SET_WATCHDOG_TIMEOUT(
      timeoutSec: U32  @< Seconds before declaring subsystem offline
    ) opcode 0x8001

    @ Request health summary report
    async command REQUEST_HEALTH_SUMMARY opcode 0x8002

    @ Reset all health counters
    async command RESET_COUNTERS opcode 0x8003

    # ------------------------------------------------------------------
    # Ports
    # ------------------------------------------------------------------

    @ Receives health reports from all subsystems
    async input port healthIn: SpaceDrone.HealthReport

    @ Sends critical health events to MissionControl
    output port fsmStateOut: SpaceDrone.FSMStateChange

    # ------------------------------------------------------------------
    # Telemetry
    # ------------------------------------------------------------------

    @ Overall system health (worst subsystem status)
    telemetry SystemHealth: SpaceDrone.HealthStatus id 0x8101

    @ Number of healthy subsystems
    telemetry HealthyCount: U8 id 0x8102

    @ Number of warning subsystems
    telemetry WarningCount: U8 id 0x8103

    @ Number of critical subsystems
    telemetry CriticalCount: U8 id 0x8104

    @ Number of offline subsystems
    telemetry OfflineCount: U8 id 0x8105

    @ Uptime in seconds
    telemetry UptimeSeconds: U64 id 0x8106

    @ Total watchdog timeouts since boot
    telemetry WatchdogTimeouts: U32 id 0x8107

    # ------------------------------------------------------------------
    # Events
    # ------------------------------------------------------------------

    @ Subsystem health changed
    event SubsystemHealthChange(
      subsystem: string size 32,
      oldStatus: SpaceDrone.HealthStatus,
      newStatus: SpaceDrone.HealthStatus
    ) severity activity high \
      format "Subsystem {} health: {} -> {}"

    @ Watchdog timeout — subsystem not reporting
    event WatchdogTimeout(
      subsystem: string size 32
    ) severity warning high \
      format "Watchdog timeout: {} not responding"

    @ System health summary
    event HealthSummary(
      healthyCount: U8,
      warningCount: U8,
      criticalCount: U8,
      offlineCount: U8
    ) severity activity low \
      format "Health: {}H {}W {}C {}O"

    # ------------------------------------------------------------------
    # Parameters
    # ------------------------------------------------------------------

    @ Default watchdog timeout (seconds)
    param WATCHDOG_TIMEOUT: U32 default 10 id 0x8201

    @ Health report interval (seconds)
    param REPORT_INTERVAL: U32 default 5 id 0x8202
  }
}
