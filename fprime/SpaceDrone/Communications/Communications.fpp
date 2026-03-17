@ Communications Component
@ Handles CCSDS Space Packet Protocol framing for drone-to-ground telemetry
@ and ground-to-drone command uplink.
@ Implements CCSDS 133.0-B-2 with APID-based routing and CRC-16 verification.

module SpaceDrone {

  @ CCSDS communication handler and link state manager
  active component Communications {

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

    @ Set communication frequency
    async command SET_FREQUENCY(
      freqMHz: F64  @< Transmit frequency in MHz
    ) opcode 0x6001

    @ Set transmit power
    async command SET_TX_POWER(
      powerDbm: F64  @< Transmit power in dBm
    ) opcode 0x6002

    @ Force link reacquisition
    async command REACQUIRE_LINK opcode 0x6003

    @ Set APID for this drone's telemetry stream
    async command SET_APID(
      apid: U16  @< Application Process Identifier [0..2047]
    ) opcode 0x6004

    @ Enable/disable telemetry downlink
    async command SET_DOWNLINK_ENABLED(
      enabled: bool
    ) opcode 0x6005

    # ------------------------------------------------------------------
    # Ports
    # ------------------------------------------------------------------

    @ Receives telemetry from MissionControl for downlink
    async input port telemetryIn: SpaceDrone.TelemetryPort

    @ Reports health to HealthMonitor
    output port healthOut: SpaceDrone.HealthReport

    # ------------------------------------------------------------------
    # Telemetry
    # ------------------------------------------------------------------

    @ Current link state
    telemetry LinkState: SpaceDrone.LinkState id 0x6101

    @ Received signal strength indicator (dBm)
    telemetry RSSI: F64 id 0x6102

    @ Packet loss rate (%) over last 60s window
    telemetry PacketLossRate: F64 id 0x6103

    @ Total packets transmitted
    telemetry PacketsTx: U64 id 0x6104

    @ Total packets received
    telemetry PacketsRx: U64 id 0x6105

    @ Current CCSDS sequence count
    telemetry SequenceCount: U16 id 0x6106

    @ Round-trip latency estimate (ms)
    telemetry Latency: F64 id 0x6107

    @ Current APID
    telemetry APID: U16 id 0x6108

    @ Transmit frequency (MHz)
    telemetry Frequency: F64 id 0x6109

    # ------------------------------------------------------------------
    # Events
    # ------------------------------------------------------------------

    @ Link state changed
    event LinkStateChange(
      oldState: SpaceDrone.LinkState,
      newState: SpaceDrone.LinkState
    ) severity activity high \
      format "Link state: {} -> {}"

    @ Link lost
    event LinkLost severity warning high \
      format "Communication link LOST"

    @ Link recovered
    event LinkRecovered(
      rssi: F64
    ) severity activity high \
      format "Link recovered, RSSI={} dBm"

    @ CRC error detected on received packet
    event CRCError(
      sequenceCount: U16
    ) severity warning low \
      format "CRC error on packet seq={}"

    @ APID mismatch on received command
    event APIDMismatch(
      expected: U16,
      received: U16
    ) severity warning low \
      format "APID mismatch: expected={}, received={}"

    # ------------------------------------------------------------------
    # Parameters
    # ------------------------------------------------------------------

    @ Link timeout before declaring LOST (seconds)
    param LINK_TIMEOUT: U32 default 30 id 0x6201

    @ Telemetry downlink rate (Hz)
    param DOWNLINK_RATE: U32 default 1 id 0x6202

    @ Default APID for this drone
    param DEFAULT_APID: U16 default 42 id 0x6203
  }
}
