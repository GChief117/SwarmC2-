@ SpaceDrone Top-Level Topology
@ Defines the component wiring for the complete space drone flight software.

module SpaceDrone {

  topology SpaceDroneTopology {

    # ------------------------------------------------------------------
    # Import standard F Prime subtopologies
    # ------------------------------------------------------------------
    import CdhCore.Subtopology

    # ------------------------------------------------------------------
    # Instances used in the topology
    # ------------------------------------------------------------------

    instance posixTime
    instance rateGroupDriver
    instance rateGroup1
    instance linuxTimer
    instance missionControl
    instance gnc
    instance powerManagement
    instance communications
    instance objectDetection
    instance healthMonitor

    # ------------------------------------------------------------------
    # Standard F Prime pattern graph specifiers
    # ------------------------------------------------------------------

    command connections instance CdhCore.cmdDisp
    event connections instance CdhCore.events
    telemetry connections instance CdhCore.tlmSend
    text event connections instance CdhCore.textLogger
    time connections instance posixTime
    param connections instance CdhCore.cmdDisp

    # ------------------------------------------------------------------
    # Rate group connections
    # ------------------------------------------------------------------

    connections RateGroups {
      linuxTimer.CycleOut -> rateGroupDriver.CycleIn
      rateGroupDriver.CycleOut[0] -> rateGroup1.CycleIn
      rateGroup1.RateGroupMemberOut[0] -> CdhCore.tlmSend.Run
      rateGroup1.RateGroupMemberOut[1] -> CdhCore.cmdDisp.run
    }

    # ------------------------------------------------------------------
    # SpaceDrone port connections — data flow
    # ------------------------------------------------------------------

    @ ObjectDetection -> MissionControl: proximity alerts drive FSM EVADE
    connections ProximityAlerts {
      objectDetection.proximityOut -> missionControl.proximityIn
    }

    @ PowerManagement -> MissionControl: energy state for invariance checks
    connections EnergyFlow {
      powerManagement.energyOut -> missionControl.energyIn
    }

    @ MissionControl -> Communications: telemetry for CCSDS downlink
    connections TelemetryDownlink {
      missionControl.telemetryOut -> communications.telemetryIn
    }

    @ MissionControl -> GNC: FSM state changes adjust guidance mode
    connections FSMToGNC {
      missionControl.fsmStateOut -> gnc.fsmStateIn
    }

    @ MissionControl -> PowerManagement: FSM state changes adjust power profile
    connections FSMToPower {
      missionControl.fsmStateOut -> powerManagement.fsmStateIn
    }

    @ GNC -> ObjectDetection: position for range calculations
    connections PositionFeed {
      gnc.telemetryOut -> objectDetection.telemetryIn
    }

    # ------------------------------------------------------------------
    # Health monitoring — all subsystems report to HealthMonitor
    # ------------------------------------------------------------------

    connections HealthReporting {
      gnc.healthOut -> healthMonitor.healthIn
      powerManagement.healthOut -> healthMonitor.healthIn
      communications.healthOut -> healthMonitor.healthIn
      objectDetection.healthOut -> healthMonitor.healthIn
    }

    @ HealthMonitor -> MissionControl: watchdog events trigger SAFE_MODE
    connections HealthToFSM {
      healthMonitor.fsmStateOut -> missionControl.healthIn
    }

  }
}
