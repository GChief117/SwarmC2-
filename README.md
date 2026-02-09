# SWARM C2 — Command & Control

Made a real-time aircraft surveillance and tactical analysis platform with live OpenSky Network data, AI-powered threat assessment, and immersive pilot POV mode, in 3 reigions (UK, Taiwan, and SoCal)

<img width="1654" height="1116" alt="Screenshot 2026-02-09 at 2 25 35 PM" src="https://github.com/user-attachments/assets/3bac9780-8a96-4749-9d39-2fea1058f557" />


<img width="1671" height="1122" alt="Screenshot 2026-02-09 at 2 36 46 PM" src="https://github.com/user-attachments/assets/f78a6120-76ef-439f-b75d-9df32df1f350" />


<img width="1675" height="1122" alt="Screenshot 2026-02-09 at 2 37 37 PM" src="https://github.com/user-attachments/assets/d024e903-45b3-4e61-af07-bbb22eff48db" />



## Features

- **Live Aircraft Tracking** — Real-time data from OpenSky Network via WebSocket
- **3 Strategic Regions** — Taiwan Strait, Southern California, United Kingdom
- **SENTINEL AI** — GPT-4.1 mini tactical threat analysis with pattern recognition
- **Pilot POV Mode** — Synthetic vision cockpit view with 3D terrain and HUD overlay
- **3D Globe** — Three.js WebGL globe with aircraft shape rendering
- **Aircraft Intelligence** — Photo, registration, operator, and route data via hexdb.io + planespotters.net

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Frontend (React + Vite)                        │
│  MapLibre GL · Three.js · WebSocket Client      │
│  Builds to static files → served by Go backend  │
├─────────────────────────────────────────────────┤
│  Backend (Go)                                   │
│  WebSocket server · OpenSky poller · OpenAI API │
│  Serves frontend static files in production     │
└─────────────────────────────────────────────────┘
```

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)
- API keys (see `.env.example`)

## Local Development

```bash
# 1. Clone
git clone https://github.com/YOUR_USER/swarm-c2.git && cd swarm-c2

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
| `OPENAI_API_KEY` | SENTINEL AI analysis | [platform.openai.com](https://platform.openai.com/api-keys) |
| `VITE_MAPTILER_KEY` | Satellite tiles + terrain | [cloud.maptiler.com](https://cloud.maptiler.com/account/keys) |
| `OPENSKY_CLIENT_ID` | Aircraft data (OAuth2) | [opensky-network.org](https://opensky-network.org) |
| `OPENSKY_CLIENT_SECRET` | Aircraft data (OAuth2) | Same |

## Deploy to Railway

1. Push to GitHub
2. [railway.app](https://railway.app) → **New Project** → **Deploy from GitHub**
3. Select `swarm-c2` repo — Railway auto-detects `railway.json`
4. Add variables in the **Variables** tab:
   ```
   OPENAI_API_KEY=sk-proj-...
   VITE_MAPTILER_KEY=...
   OPENSKY_CLIENT_ID=...
   OPENSKY_CLIENT_SECRET=...
   PORT=8080
   ```
5. Deploy triggers automatically

### Railway CLI Alternative

```bash
npm i -g @railway/cli && railway login
railway init
railway variables set OPENAI_API_KEY=... VITE_MAPTILER_KEY=... OPENSKY_CLIENT_ID=... OPENSKY_CLIENT_SECRET=...
railway up
```

## Project Structure

```
swarm-c2/
├── backend/
│   └── main.go            # Go: WebSocket, OpenSky poller, OpenAI, static server
├── frontend/src/
│   ├── App.jsx             # WebSocket state, region management
│   ├── index.css           # All styles
│   └── components/
│       ├── FlightMap.jsx       # MapLibre map, markers, POV mode
│       ├── Globe3D.jsx         # Three.js 3D globe
│       ├── AircraftPopup.jsx   # Aircraft detail popup
│       ├── AIAnalysisPanel.jsx # SENTINEL AI panel
│       ├── AircraftList.jsx    # Tracked aircraft list
│       ├── TelemetryPanel.jsx  # Bottom instruments
│       └── Clock.jsx           # Header clock
├── railway.json        # Railway build/deploy config
├── nixpacks.toml       # Nixpacks build phases
└── .env.example        # Environment variable template
```

## Build Pipeline

Nixpacks (Railway) runs: `npm install` → `vite build` → copy `dist/` → `backend/static/` → `go build` → single binary serves everything on one port.

## License

MIT
