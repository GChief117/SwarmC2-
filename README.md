# SWARM C2 — Command & Control

Made a real-time aircraft surveillance and tactical analysis platform with live OpenSky Network data, AI-powered ariel assessment, and immersive pilot POV mode, in 2 regions (UK and SoCal)

<img width="1654" height="1116" alt="Screenshot 2026-02-09 at 2 25 35 PM" src="https://github.com/user-attachments/assets/3bac9780-8a96-4749-9d39-2fea1058f557" />


<img width="1671" height="1122" alt="Screenshot 2026-02-09 at 2 36 46 PM" src="https://github.com/user-attachments/assets/f78a6120-76ef-439f-b75d-9df32df1f350" />



## Features

- **Live Aircraft Tracking** — Real-time data from OpenSky Network via WebSocket
- **3 Strategic Regions** —  Strait, Southern California, United Kingdom
- **SENTINEL AI** — Opus 4 mini tactical ariel analysis with pattern recognition
- **Pilot POV Mode** — Synthetic vision cockpit view with 3D terrain and HUD overlay
- **3D Globe** — Three.js WebGL globe with aircraft shape rendering
- **Aircraft Intelligence** — Photo, registration, operator, and route data via hexdb.io + planespotters.net

---

## F Prime Integration — Autonomous Space Drone Operations

SwarmC2 extends its ground control capabilities to support **NASA F Prime**-based autonomous space drones, as described in the ESS paper: *"Autonomous Space UAVs: Embedded Systems Architecture for Intergalactic Drones with NASA F Prime and Cloud-Native Ground Control Systems"* (Nelson, 2026).

### DRONE OPS Mode

The frontend includes a **DRONE OPS** mode that provides real-time telemetry, FSM visualization, and remote configuration for a simulated fleet of 3 autonomous drones.

### F Prime Components (FPP Definitions)

The `fprime/SpaceDrone/` directory contains 8 F Prime component definitions modeling the complete flight software architecture:

| Component | File | Purpose |
|-----------|------|---------|
| **SpaceDroneTypes** | `Types/SpaceDroneTypes.fpp` | Core type definitions: FSMState/FSMInput enums, DronePosition/EnergyBudget structs, port definitions |
| **MissionControl** | `MissionControl/MissionControl.fpp` | Central FSM controller (BOOT/CRUISE/EVADE/HOLD/SAFE_MODE), ground commands, state transitions |
| **GNC** | `GNC/GNC.fpp` | Guidance, Navigation & Control: position estimation, waypoint tracking, flight control |
| **PowerManagement** | `PowerManagement/PowerManagement.fpp` | Energy invariance enforcement (Theorem 1): E(C') ≤ B ≤ (Ps+Pb)·T |
| **Communications** | `Communications/Communications.fpp` | CCSDS Space Packet Protocol (133.0-B-2): framing, APID routing, CRC-16 |
| **ObjectDetection** | `ObjectDetection/ObjectDetection.fpp` | Proximity detection and collision avoidance via LiDAR |
| **HealthMonitor** | `HealthMonitor/HealthMonitor.fpp` | Subsystem watchdog with configurable timeout |
| **SpaceDroneTopology** | `Top/SpaceDroneTopology.fpp` | Top-level topology wiring all components via ports |

### Finite State Machine (Appendix B)

The drone's autonomy logic is implemented as a deterministic FSM:

```
M = (S, I, f, s₀, A)
  S  = {BOOT, CRUISE, EVADE, HOLD, SAFE_MODE}
  I  = {proximity_alert, proximity_clear, power_drop, formation_signal}
  s₀ = BOOT
  A  = {SAFE_MODE}
```

**Transition Table** (5 states × 4 inputs = 20 defined pairs):

|           | proximity_alert | proximity_clear | power_drop | formation_signal |
|-----------|----------------|-----------------|------------|-----------------|
| BOOT      | SAFE_MODE      | CRUISE          | SAFE_MODE  | CRUISE          |
| CRUISE    | EVADE          | CRUISE          | SAFE_MODE  | CRUISE          |
| EVADE     | EVADE          | HOLD            | SAFE_MODE  | HOLD            |
| HOLD      | EVADE          | HOLD            | SAFE_MODE  | CRUISE          |
| SAFE_MODE | SAFE_MODE      | SAFE_MODE       | SAFE_MODE  | SAFE_MODE       |

**Theorem 2 (FSM Completeness):** Every state-input pair has a defined next state — no deadlocks, undefined outputs, or unrecoverable states can occur.

### Energy Invariance (Appendix A)

Remote configuration is gated by a formal energy proof:

```
E(C') ≤ B ≤ (Ps + Pb) · T

Where:
  Ps = 12W (solar), Pb = 8W (battery)
  T  = 0.0025s (400 Hz control loop)
  (Ps+Pb)·T = 0.050 J
  B  = 0.045 J (10% safety margin)

Baseline C₀: ε₁=0.018J (attitude) + ε₂=0.008J (encoding) + ε₃=0.006J (health)
  E(C₀) = 0.032 J
```

**Theorem 1 (Energy Invariance):** A candidate configuration C' is accepted only when E(C') ≤ B. The drone never consumes more energy per cycle than its power system delivers.

### Control Pipeline & Scheduling (Section 6)

The RTOS schedules tasks using Rate Monotonic Scheduling:

| Task | Rate | Priority | WCET |
|------|------|----------|------|
| Attitude Control | 400 Hz | 1 (highest) | 500 µs |
| Guidance Loop | 100 Hz | 2 | 1200 µs |
| Navigation Update | 10 Hz | 3 | 5000 µs |
| Health Monitoring | 10 Hz | 4 | 2000 µs |
| Object Detection | 10 Hz | 5 | 8000 µs |
| Telemetry Downlink | 1 Hz | 6 (lowest) | 3000 µs |

### Sensor Interfaces (Section 5, Tables 6-7)

| Sensor | Part | Protocol | Signal |
|--------|------|----------|--------|
| 6DoF IMU | ISM330DHCX | SPI/I²C | Digital |
| Atmospheric | BME280 | I²C/SPI | Digital |
| GPS/GNSS | NEO-M9N | I²C/UART | Digital |
| LiDAR | LIDAR-Lite v3 | I²C/PWM | Digital |
| Battery Current | INA169 | Analog→ADC | Analog |
| Solar Current | ACS723 | Analog→ADC | Analog |

### Communication Links (Section 5.3, Table 8)

| Link | Type | Bandwidth | Protocol |
|------|------|-----------|----------|
| S-Band Ground | RF | 2 Mbps | CCSDS (AES-256) |
| UHF Backup | RF | 9.6 kbps | Beacon |
| Inter-Drone Mesh | SpaceWire | 200 Mbps | SpaceWire |

### CCSDS Packet Handler

The Go backend includes a full CCSDS Space Packet Protocol (133.0-B-2) implementation in `backend/ccsds/ccsds.go`:
- 6-byte primary header with APID-based routing
- CRC-16/CCITT-FALSE verification
- Encode/decode for telemetry and command packets

### Drone API Endpoints

```
GET  /api/drones            — list drones + summary state
GET  /api/drones/telemetry   — full telemetry (?drone_id=)
GET  /api/drones/events      — recent events for a drone
GET  /api/drones/fsm         — FSM state + transition table
POST /api/drones/config      — push config (validates Theorem 1 & 2 first)
POST /api/drones/validate    — dry-run validation only
WS   /ws/drones              — real-time drone telemetry stream (1 Hz)
```

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Frontend (React + Vite)                        │
│  MapLibre GL · Three.js · WebSocket Client      │
│  Builds to static files → served by Go backend  │
├─────────────────────────────────────────────────┤
│  Backend (Go)                                   │
│  WebSocket server · OpenSky poller · Claude API │
│  F Prime GDS bridge · CCSDS handler             │
│  Drone simulator · Validation gates             │
│  Serves frontend static files in production     │
├─────────────────────────────────────────────────┤
│  F Prime FPP Definitions                        │
│  8 components · FSM · Energy model · CCSDS      │
│  Sensor interfaces · Watchdog · Scheduling      │
└─────────────────────────────────────────────────┘
```

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- API keys (see `.env.example`)

## Local Development

```bash
# 1. Clone
git clone https://github.com/GChief117/SwarmC2-.git && cd SwarmC2-

# 2. Copy env and add your keys
cp .env.example .env

# 3. Install frontend
cd frontend && npm install && cd ..

# 4. Generate Go dependencies
cd backend && go mod tidy && cd ..

# 5. Start backend (terminal 1)
cd backend && go run main.go

# 6. Start frontend dev server (terminal 2)
cd frontend && npm run dev
# → http://localhost:5173
```

## API Keys

| Key | Purpose | Source |
|-----|---------|--------|
| `ANTHROPIC_API_KEY` | SENTINEL AI analysis | [console.anthropic.com](https://console.anthropic.com) |
| `VITE_MAPTILER_KEY` | Satellite tiles + terrain | [cloud.maptiler.com](https://cloud.maptiler.com/account/keys) |

## Project Structure

```
SwarmC2-/
├── backend/
│   ├── main.go                # Go: WebSocket, aircraft sim, Claude API, drone routes
│   ├── ccsds/
│   │   └── ccsds.go           # CCSDS Space Packet Protocol encoder/decoder
│   └── fprime/
│       ├── bridge.go          # FSM, energy model, drone state types, fleet manager
│       ├── simulator.go       # Mock telemetry: 3 drones, FSM transitions, sensors
│       └── validation.go      # Theorem 1 (energy) + Theorem 2 (FSM) gates
├── frontend/src/
│   ├── App.jsx                # WebSocket state, DRONE OPS / AIRCRAFT modes
│   ├── index.css              # All styles
│   └── components/
│       ├── FlightMap.jsx          # MapLibre 2D map
│       ├── Globe3D.jsx            # Three.js 3D globe with drone rendering
│       ├── AIAnalysisPanel.jsx    # SENTINEL AI tactical panel
│       ├── AircraftList.jsx       # Tracked aircraft list
│       ├── TelemetryPanel.jsx     # Aircraft telemetry instruments
│       ├── DronePanel.jsx         # Drone ops sidebar (tabs)
│       ├── DroneFSMDisplay.jsx    # FSM diagram + transition table
│       ├── DroneTelemetryGrid.jsx # Metrics, sensors, scheduling, energy invariance
│       ├── DroneEventLog.jsx      # Event log with severity coloring
│       ├── DroneConfigPanel.jsx   # Remote config with validation gates
│       └── Clock.jsx              # Header clock
├── fprime/SpaceDrone/
│   ├── Types/SpaceDroneTypes.fpp
│   ├── Top/SpaceDroneTopology.fpp
│   ├── MissionControl/MissionControl.fpp
│   ├── GNC/GNC.fpp
│   ├── PowerManagement/PowerManagement.fpp
│   ├── Communications/Communications.fpp
│   ├── ObjectDetection/ObjectDetection.fpp
│   └── HealthMonitor/HealthMonitor.fpp
├── Dockerfile
├── .gitignore
└── README.md
```

## License

MIT
