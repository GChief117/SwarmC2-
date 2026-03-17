@ Guidance, Navigation, and Control (GNC) Component
@ Manages position estimation, waypoint tracking, and flight control outputs.
@ Provides core navigation telemetry to MissionControl and Communications.

module SpaceDrone {

  @ GNC subsystem: sensor fusion, waypoint guidance, attitude control
  active component GNC {

    # ------------------------------------------------------------------
    # Commands
    # ------------------------------------------------------------------

    @ Load a waypoint into the guidance queue
    async command LOAD_WAYPOINT(
      latitude: F64
      longitude: F64
      altitude: F64
      speed: F64       @< Target ground speed, m/s
    ) opcode 0x4001

    @ Clear all queued waypoints and hold current position
    async command CLEAR_WAYPOINTS opcode 0x4002

    @ Set maximum allowed speed
    async command SET_MAX_SPEED(
      maxSpeed: F64  @< m/s
    ) opcode 0x4003

    @ Set maximum allowed altitude
    async command SET_MAX_ALTITUDE(
      maxAltitude: F64  @< meters MSL
    ) opcode 0x4004

    @ Execute evasion maneuver in given direction
    async command EXECUTE_EVASION(
      bearing: F64   @< Evasion heading, degrees
      distance: F64  @< Minimum displacement, meters
    ) opcode 0x4005

    # ------------------------------------------------------------------
    # Ports
    # ------------------------------------------------------------------

    @ Receives FSM state changes to adjust guidance mode
    async input port fsmStateIn: SpaceDrone.FSMStateChange

    @ Outputs full telemetry including position/velocity
    output port telemetryOut: SpaceDrone.TelemetryPort

    @ Reports health to HealthMonitor
    output port healthOut: SpaceDrone.HealthReport

    # ------------------------------------------------------------------
    # Telemetry
    # ------------------------------------------------------------------

    @ Current latitude (degrees)
    telemetry Latitude: F64 id 0x4101

    @ Current longitude (degrees)
    telemetry Longitude: F64 id 0x4102

    @ Current altitude MSL (meters)
    telemetry Altitude: F64 id 0x4103

    @ Ground speed (m/s)
    telemetry GroundSpeed: F64 id 0x4104

    @ Vertical speed (m/s, positive = ascending)
    telemetry VerticalSpeed: F64 id 0x4105

    @ Magnetic heading (degrees)
    telemetry Heading: F64 id 0x4106

    @ Distance to next waypoint (meters)
    telemetry WaypointDistance: F64 id 0x4107

    @ Number of remaining waypoints
    telemetry WaypointsRemaining: U32 id 0x4108

    @ GPS fix quality (0=none, 1=2D, 2=3D, 3=DGPS)
    telemetry GPSFixQuality: U8 id 0x4109

    @ Number of GPS satellites tracked
    telemetry GPSSatellites: U8 id 0x410A

    # ------------------------------------------------------------------
    # Events
    # ------------------------------------------------------------------

    @ Waypoint reached
    event WaypointReached(
      waypointIndex: U32
      latitude: F64
      longitude: F64
    ) severity activity high \
      format "Waypoint {} reached at ({}, {})"

    @ All waypoints completed
    event MissionComplete severity activity high \
      format "All waypoints reached, entering HOLD"

    @ Evasion maneuver initiated
    event EvasionStarted(
      bearing: F64
      distance: F64
    ) severity warning high \
      format "Evasion maneuver: heading {} for {}m"

    @ GPS fix lost
    event GPSFixLost severity warning high \
      format "GPS fix lost, using dead reckoning"

    # ------------------------------------------------------------------
    # Parameters
    # ------------------------------------------------------------------

    @ Waypoint arrival radius (meters)
    param WAYPOINT_RADIUS: F64 default 10.0 id 0x4201

    @ Maximum ground speed (m/s)
    param MAX_SPEED: F64 default 25.0 id 0x4202

    @ Maximum altitude MSL (meters)
    param MAX_ALTITUDE: F64 default 400.0 id 0x4203

    @ Navigation update rate (Hz)
    param NAV_UPDATE_RATE: U32 default 10 id 0x4204
  }
}
