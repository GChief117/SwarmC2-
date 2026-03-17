@ SpaceDrone Top-Level Topology
@ Defines the component wiring for the complete space drone flight software.
@ This topology maps directly to the SwarmC2 system architecture:
@
@   ┌──────────────┐     ┌─────┐     ┌──────────────┐
@   │ObjectDetection│────▶│ FSM │────▶│Communications│
@   └──────────────┘     │(MC) │     └──────────────┘
@                        └──┬──┘            ▲
@   ┌──────────────┐       │          ┌─────┴────────┐
@   │PowerManagement│──────┘          │     GNC      │
@   └──────────────┘                  └──────────────┘
@                        ┌──────────────┐
@                        │HealthMonitor │ ◀── all subsystems
@                        └──────────────┘
@
@ All inter-component communication uses typed F Prime ports.
@ The topology enforces that:
@   1. MissionControl (FSM) is the central decision authority
@   2. All telemetry flows through Communications for CCSDS framing
@   3. HealthMonitor receives reports from every subsystem
@   4. ObjectDetection feeds proximity alerts to MissionControl

module SpaceDrone {

  @ Top-level topology for SpaceDrone flight software
  topology SpaceDroneTopology {

    # ------------------------------------------------------------------
    # Component instances
    # ------------------------------------------------------------------

    instance missionControl: SpaceDrone.MissionControl base id 0x1000
    instance gnc: SpaceDrone.GNC base id 0x4000
    instance powerManagement: SpaceDrone.PowerManagement base id 0x5000
    instance communications: SpaceDrone.Communications base id 0x6000
    instance objectDetection: SpaceDrone.ObjectDetection base id 0x7000
    instance healthMonitor: SpaceDrone.HealthMonitor base id 0x8000

    # ------------------------------------------------------------------
    # Port connections — data flow
    # ------------------------------------------------------------------

    @ ObjectDetection → MissionControl: proximity alerts drive FSM EVADE
    connections ProximityAlerts {
      objectDetection.proximityOut -> missionControl.proximityIn
    }

    @ PowerManagement → MissionControl: energy state for invariance checks
    connections EnergyFlow {
      powerManagement.energyOut -> missionControl.energyIn
    }

    @ MissionControl → Communications: telemetry for CCSDS downlink
    connections TelemetryDownlink {
      missionControl.telemetryOut -> communications.telemetryIn
    }

    @ MissionControl → GNC: FSM state changes adjust guidance mode
    connections FSMToGNC {
      missionControl.fsmStateOut -> gnc.fsmStateIn
    }

    @ MissionControl → PowerManagement: FSM state changes adjust power profile
    connections FSMToPower {
      missionControl.fsmStateOut -> powerManagement.fsmStateIn
    }

    @ GNC → ObjectDetection: position for range calculations
    connections PositionFeed {
      gnc.telemetryOut -> objectDetection.telemetryIn
    }

    # ------------------------------------------------------------------
    # Health monitoring connections — all subsystems report health
    # ------------------------------------------------------------------

    connections HealthReporting {
      gnc.healthOut -> healthMonitor.healthIn
      powerManagement.healthOut -> healthMonitor.healthIn
      communications.healthOut -> healthMonitor.healthIn
      objectDetection.healthOut -> healthMonitor.healthIn
    }

    @ HealthMonitor → MissionControl: watchdog events trigger SAFE_MODE
    connections HealthToFSM {
      healthMonitor.fsmStateOut -> missionControl.fsmStateIn  # Note: uses health port
    }
  }
}
