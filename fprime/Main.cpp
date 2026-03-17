// ======================================================================
// SpaceDrone F Prime Deployment — Main Entry Point
//
// HADES Autonomous Space Drone Flight Software
// Reference: Nelson, G. "Autonomous Space UAVs" — Oxford ESS, 2026.
//
// This binary runs the F Prime runtime with the SpaceDrone topology:
//   MissionControl → FSM (BOOT/CRUISE/EVADE/HOLD/SAFE_MODE)
//   GNC → Guidance, Navigation & Control (400Hz attitude loop)
//   PowerManagement → Energy Invariance enforcement (Theorem 1)
//   Communications → CCSDS Space Packet Protocol (S-band/UHF)
//   ObjectDetection → LiDAR proximity detection
//   HealthMonitor → Watchdog timer supervision
//
// The F Prime GDS connects via TCP to provide ground control,
// which SwarmC2 bridges to a cloud-native web interface.
// ======================================================================

#include <cstdlib>
#include <SpaceDrone/Top/SpaceDroneTopologyAc.hpp>

// Instantiate the topology
SpaceDrone::TopologyState state;

// Component instances (auto-generated from FPP topology)
void constructApp() {
    SpaceDrone::setup(state);
}

void exitApp() {
    SpaceDrone::teardown(state);
}

int main(int argc, char* argv[]) {
    constructApp();

    // Run the F Prime rate group scheduler
    // In deployment: this blocks, running the control pipeline forever
    // 400Hz attitude | 100Hz guidance | 10Hz nav/health | 1Hz downlink
    SpaceDrone::run(state);

    exitApp();
    return EXIT_SUCCESS;
}
