@ Object Detection Component
@ Monitors proximity sensors and generates collision avoidance alerts.
@ Feeds threat data to MissionControl for FSM EVADE transitions.

module SpaceDrone {

  @ Proximity detection and collision avoidance alert generator
  active component ObjectDetection {

    # ------------------------------------------------------------------
    # Commands
    # ------------------------------------------------------------------

    @ Set safety radius for proximity alerts
    async command SET_SAFETY_RADIUS(
      radius: F64  @< Safety radius in meters
    ) opcode 0x7001

    @ Set minimum separation distance for formation flight
    async command SET_FORMATION_DISTANCE(
      distance: F64  @< Minimum inter-drone separation, meters
    ) opcode 0x7002

    @ Enable/disable collision avoidance
    async command SET_AVOIDANCE_ENABLED(
      enabled: bool
    ) opcode 0x7003

    # ------------------------------------------------------------------
    # Ports
    # ------------------------------------------------------------------

    @ Outputs proximity alerts to MissionControl
    output port proximityOut: SpaceDrone.ProximityAlert

    @ Reports health to HealthMonitor
    output port healthOut: SpaceDrone.HealthReport

    @ Receives current position for range calculations
    async input port telemetryIn: SpaceDrone.TelemetryPort

    # ------------------------------------------------------------------
    # Telemetry
    # ------------------------------------------------------------------

    @ Number of currently tracked objects
    telemetry TrackedObjects: U32 id 0x7101

    @ Nearest object distance (meters)
    telemetry NearestObjectRange: F64 id 0x7102

    @ Nearest object bearing (degrees)
    telemetry NearestObjectBearing: F64 id 0x7103

    @ Current threat level assessment
    telemetry CurrentThreatLevel: SpaceDrone.ThreatLevel id 0x7104

    @ Closing speed of nearest threat (m/s)
    telemetry NearestClosingSpeed: F64 id 0x7105

    @ Total proximity alerts generated since boot
    telemetry AlertCount: U32 id 0x7106

    # ------------------------------------------------------------------
    # Events
    # ------------------------------------------------------------------

    @ New object detected within safety radius
    event ObjectDetected(
      range: F64
      bearing: F64
      closingSpeed: F64
    ) severity warning high \
      format "Object detected: range={}m, bearing={}°, closing={}m/s"

    @ Object cleared safety radius
    event ObjectCleared(
      range: F64
    ) severity activity high \
      format "Object cleared at {}m"

    @ Collision avoidance alert escalated
    event ThreatEscalated(
      oldLevel: SpaceDrone.ThreatLevel
      newLevel: SpaceDrone.ThreatLevel
    ) severity warning high \
      format "Threat escalated: {} -> {}"

    @ Formation separation warning
    event FormationWarning(
      droneId: string size 32
      separation: F64
    ) severity warning low \
      format "Formation drone {} too close: {}m"

    # ------------------------------------------------------------------
    # Parameters
    # ------------------------------------------------------------------

    @ Safety radius for proximity alerts (meters)
    param SAFETY_RADIUS: F64 default 100.0 id 0x7201

    @ Minimum formation separation (meters)
    param MIN_FORMATION_SEP: F64 default 50.0 id 0x7202

    @ Sensor update rate (Hz)
    param SENSOR_RATE: U32 default 20 id 0x7203
  }
}
