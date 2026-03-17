package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"swarm-c2/fprime"
)

// OpenAI Integration
const TACTICAL_SYSTEM_PROMPT = `You are SENTINEL, an advanced tactical AI advisor for the SWARM C2 (Command & Control) system. Your role is to analyze real-time aircraft tracking data and provide threat assessments, pattern recognition, and tactical recommendations.

## CORE RESPONSIBILITIES

1. **THREAT ASSESSMENT**: Evaluate danger levels based on aircraft positions, trajectories, and behaviors
2. **PATTERN RECOGNITION**: Identify formations, coordinated movements, intercept trajectories, and anomalies
3. **SITUATIONAL AWARENESS**: Provide context on geopolitical implications of observed activity
4. **TACTICAL RECOMMENDATIONS**: Suggest appropriate responses and monitoring priorities

## THREAT LEVEL CLASSIFICATION

- **CRITICAL** (Red): Immediate threat, hostile intent confirmed, requires immediate action
  - Active intercept trajectory toward protected assets
  - Weapons-hot indicators (specific squawk codes)
  - Violation of restricted airspace with aggressive maneuvering
  
- **HIGH** (Orange): Elevated concern, potential hostile activity
  - Military aircraft in unusual positions near borders
  - Formation flying indicating coordinated operation
  - Rapid altitude/heading changes suggesting combat maneuvering
  - Aircraft approaching ADIZ (Air Defense Identification Zone)
  
- **MEDIUM** (Yellow): Monitored activity, requires attention
  - Unidentified aircraft in sensitive areas
  - Unusual flight patterns deviating from commercial routes
  - Military transport aircraft in contested regions
  - Increased traffic density in strategic areas
  
- **LOW** (Blue): Normal military/government activity
  - Routine patrol patterns
  - Known training exercises
  - Standard military transport operations
  
- **NOMINAL** (Green): Normal civilian/commercial activity
  - Commercial airliners on established routes
  - General aviation in unrestricted airspace

## GEOGRAPHIC CONTEXT

**Taiwan Strait Region:**
- Taiwan ADIZ boundaries and median line significance
- PRC military activity patterns (Eastern Theater Command)
- Key strategic points: Pratas Islands, Kinmen, Matsu
- Commercial shipping lane considerations

**Southern California:**
- Military Operating Areas (MOAs) and restricted zones
- Edwards AFB, Point Mugu, China Lake test ranges
- Commercial traffic corridors (LAX, SAN approaches)

**United Kingdom:**
- RAF and NATO QRA (Quick Reaction Alert) operations
- North Sea patrol patterns and offshore energy infrastructure
- London FIR and Scottish FIR boundaries
- Military areas: Salisbury Plain, Welsh MOD ranges, North Sea danger areas
- Watch for: Russian long-range aviation probing UK ADIZ, P-8 maritime patrol

## ANALYSIS OUTPUT FORMAT

Provide analysis in this JSON structure:
{
  "timestamp": "ISO-8601",
  "region": "region_name",
  "overall_threat_level": "CRITICAL|HIGH|MEDIUM|LOW|NOMINAL",
  "threat_score": 0-100,
  "summary": "Brief 1-2 sentence situation summary",
  "key_observations": [
    {
      "type": "FORMATION|INTERCEPT|ANOMALY|PATROL|VIOLATION|TRANSIT",
      "description": "What was observed",
      "aircraft_involved": ["callsign1", "callsign2"],
      "threat_contribution": "HIGH|MEDIUM|LOW"
    }
  ],
  "aircraft_of_interest": [
    {
      "callsign": "ABC123",
      "icao24": "hex_code",
      "threat_level": "CRITICAL|HIGH|MEDIUM|LOW|NOMINAL",
      "reason": "Why this aircraft is notable",
      "recommended_action": "TRACK|MONITOR|INTERCEPT|IGNORE"
    }
  ],
  "tactical_recommendations": [
    {
      "priority": 1-5,
      "action": "Recommended action",
      "rationale": "Why this action"
    }
  ],
  "pattern_analysis": {
    "formations_detected": 0,
    "unusual_behaviors": 0,
    "potential_threats": 0,
    "commercial_density": "LOW|NORMAL|HIGH"
  },
  "next_update_priority": "IMMEDIATE|HIGH|NORMAL|LOW"
}

## EDGE CASES AND SPECIAL HANDLING

1. **Emergency Squawks**: 7500 (hijack), 7600 (comm failure), 7700 (emergency) - Always flag as priority
2. **Military vs Civilian Ambiguity**: When uncertain, analyze trajectory and behavior patterns
3. **Data Gaps**: Note when aircraft disappear from tracking (potential jamming or low-altitude flight)
4. **Coordinated Activity**: Multiple aircraft with synchronized heading/altitude changes
5. **Shadow Tracking**: Aircraft following same route as another with offset timing
6. **Holding Patterns**: Could indicate reconnaissance or waiting for clearance
7. **Unusual Speed/Altitude**: Jets at low altitude or slow aircraft at high altitude

## RESPONSE GUIDELINES

- Be decisive but indicate confidence levels
- Prioritize actionable intelligence
- Consider false positive rates - avoid alert fatigue
- Note data limitations and assumptions
- Provide context for non-expert operators
- Maintain operational security awareness in recommendations`

// Anthropic API structures
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []AnthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature"`
}

type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicResponse struct {
	Content []AnthropicContentBlock `json:"content"`
	Error   *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// TacticalAnalysis represents AI analysis results
type TacticalAnalysis struct {
	Timestamp             string                   `json:"timestamp"`
	Region                string                   `json:"region"`
	OverallThreatLevel    string                   `json:"overall_threat_level"`
	ThreatScore           int                      `json:"threat_score"`
	Summary               string                   `json:"summary"`
	KeyObservations       []map[string]interface{} `json:"key_observations"`
	AircraftOfInterest    []map[string]interface{} `json:"aircraft_of_interest"`
	TacticalRecommendations []map[string]interface{} `json:"tactical_recommendations"`
	PatternAnalysis       map[string]interface{}   `json:"pattern_analysis"`
	NextUpdatePriority    string                   `json:"next_update_priority"`
	Raw                   string                   `json:"raw,omitempty"`
}

var (
	analysisCache     = make(map[string]*TacticalAnalysis)
	analysisCacheMutex sync.RWMutex
)

// Aircraft represents a single aircraft state from OpenSky
type Aircraft struct {
	ICAO24         string   `json:"icao24"`
	Callsign       string   `json:"callsign"`
	OriginCountry  string   `json:"originCountry"`
	TimePosition   *int64   `json:"timePosition"`
	LastContact    int64    `json:"lastContact"`
	Longitude      *float64 `json:"longitude"`
	Latitude       *float64 `json:"latitude"`
	BaroAltitude   *float64 `json:"baroAltitude"`
	OnGround       bool     `json:"onGround"`
	Velocity       *float64 `json:"velocity"`
	TrueTrack      *float64 `json:"trueTrack"`
	VerticalRate   *float64 `json:"verticalRate"`
	Sensors        []int    `json:"sensors"`
	GeoAltitude    *float64 `json:"geoAltitude"`
	Squawk         *string  `json:"squawk"`
	SPI            bool     `json:"spi"`
	PositionSource int      `json:"positionSource"`
	Category       int      `json:"category"`
}

// AirspaceData represents processed data sent to clients
type AirspaceData struct {
	Timestamp int64      `json:"timestamp"`
	Aircraft  []Aircraft `json:"aircraft"`
	Region    string     `json:"region"`
	Count     int        `json:"count"`
}

// Region defines a geographic bounding box
type Region struct {
	Name   string  `json:"name"`
	MinLat float64 `json:"minLat"`
	MaxLat float64 `json:"maxLat"`
	MinLon float64 `json:"minLon"`
	MaxLon float64 `json:"maxLon"`
}

// Predefined regions
var regions = map[string]Region{
	"socal": {
		Name:   "Southern California",
		MinLat: 32.5,
		MaxLat: 34.5,
		MinLon: -120.0,
		MaxLon: -117.0,
	},
	"europe": {
		Name:   "United Kingdom",
		MinLat: 49.9,
		MaxLat: 60.9,
		MinLon: -8.2,
		MaxLon: 1.8,
	},
}

// Simulated flight route
type SimRoute struct {
	Callsign      string
	OriginCountry string
	DepLat, DepLon float64
	ArrLat, ArrLon float64
	CycleSec      float64 // how many seconds for a full route cycle
	PhaseOffset   float64 // offset in seconds so flights don't bunch up
}

// Predefined routes for each region
var simRoutes = map[string][]SimRoute{
	"socal": {
		{"UAL1522", "United States", 33.94, -118.41, 37.62, -122.38, 2400, 0},      // LAX→SFO
		{"SWA437",  "United States", 33.94, -118.41, 36.08, -115.15, 1800, 200},     // LAX→LAS
		{"DAL892",  "United States", 33.94, -118.41, 33.44, -112.01, 2100, 400},     // LAX→PHX
		{"AAL118",  "United States", 33.94, -118.41, 32.90, -97.04, 5400, 600},      // LAX→DFW
		{"UAL489",  "United States", 33.94, -118.41, 39.86, -104.67, 4200, 800},     // LAX→DEN
		{"JBU624",  "United States", 32.73, -117.19, 37.62, -122.38, 2700, 1000},    // SAN→SFO
		{"SWA1203", "United States", 32.73, -117.19, 36.08, -115.15, 1800, 1200},    // SAN→LAS
		{"AAL2145", "United States", 34.06, -117.60, 41.97, -87.91, 7200, 1400},     // ONT→ORD
		{"SWA318",  "United States", 34.20, -118.36, 36.08, -115.15, 1800, 1600},    // BUR→LAS
		{"DAL1847", "United States", 33.94, -118.41, 47.45, -122.31, 5400, 1800},    // LAX→SEA
		{"UAL2210", "United States", 33.94, -118.41, 41.97, -87.91, 7800, 2000},     // LAX→ORD
		{"AAL734",  "United States", 33.94, -118.41, 40.64, -73.78, 10800, 2200},    // LAX→JFK
		{"SWA992",  "United States", 33.94, -118.41, 33.64, -84.43, 8400, 2400},     // LAX→ATL
		{"UAL157",  "United States", 37.62, -122.38, 33.94, -118.41, 2400, 2600},    // SFO→LAX
		{"SWA814",  "United States", 36.08, -115.15, 33.94, -118.41, 1800, 2800},    // LAS→LAX
		{"DAL445",  "United States", 33.44, -112.01, 33.94, -118.41, 2100, 3000},    // PHX→LAX
		{"AAL670",  "United States", 32.90, -97.04, 33.94, -118.41, 5400, 3200},     // DFW→LAX
		{"SWA2308", "United States", 32.73, -117.19, 33.44, -112.01, 1500, 3400},    // SAN→PHX
		{"HAL11",   "United States", 33.94, -118.41, 21.32, -157.92, 10800, 3600},   // LAX→HNL
		{"UAL796",  "United States", 39.86, -104.67, 33.94, -118.41, 4200, 3800},    // DEN→LAX
		{"SWA1654", "United States", 36.08, -115.15, 32.73, -117.19, 1800, 4000},    // LAS→SAN
		{"AAL1890", "United States", 33.94, -118.41, 25.80, -80.29, 9600, 4200},     // LAX→MIA
		{"DAL2034", "United States", 33.64, -84.43, 33.94, -118.41, 8400, 4400},     // ATL→LAX
		{"JBU127",  "United States", 40.64, -73.78, 33.94, -118.41, 10800, 4600},    // JFK→LAX
		{"SKW5412", "United States", 33.94, -118.41, 34.06, -117.60, 600, 4800},     // LAX→ONT shuttle
	},
	"europe": {
		{"BAW115",  "United Kingdom", 51.47, -0.45, 40.64, -73.78, 14400, 0},        // LHR→JFK
		{"BAW303",  "United Kingdom", 51.47, -0.45, 49.01, 2.55, 2400, 300},         // LHR→CDG
		{"EZY8901", "United Kingdom", 51.15, -0.18, 41.30, 2.08, 4800, 600},         // LGW→BCN
		{"RYR217",  "United Kingdom", 51.89, 0.24, 53.43, -6.25, 2400, 900},         // STN→DUB
		{"EZY6023", "United Kingdom", 53.35, -2.28, 55.95, -3.36, 1800, 1200},       // MAN→EDI
		{"BAW1446", "United Kingdom", 51.47, -0.45, 52.31, 4.77, 2400, 1500},        // LHR→AMS
		{"VIR401",  "United Kingdom", 51.47, -0.45, 33.94, -118.41, 18000, 1800},    // LHR→LAX
		{"EZY435",  "United Kingdom", 51.47, -0.45, 55.95, -3.36, 2700, 2100},       // LHR→EDI
		{"BAW225",  "United Kingdom", 51.47, -0.45, 25.25, 55.36, 12600, 2400},      // LHR→DXB
		{"RYR812",  "United Kingdom", 51.89, 0.24, 41.80, 12.24, 5400, 2700},        // STN→FCO
		{"LOG301",  "United Kingdom", 55.95, -3.36, 51.47, -0.45, 2700, 3000},       // EDI→LHR
		{"EZY6210", "United Kingdom", 51.15, -0.18, 52.31, 4.77, 2400, 3300},        // LGW→AMS
		{"BAW883",  "United Kingdom", 51.47, -0.45, 50.04, 8.56, 3600, 3600},        // LHR→FRA
		{"EZY321",  "United Kingdom", 53.35, -2.28, 38.78, -9.14, 6000, 3900},       // MAN→LIS
		{"RYR506",  "United Kingdom", 51.89, 0.24, 40.50, -3.57, 5400, 4200},        // STN→MAD
		{"BAW762",  "United Kingdom", 51.47, -0.45, 47.46, 8.55, 3600, 4500},        // LHR→ZRH
		{"TOM2314", "United Kingdom", 53.35, -2.28, 41.30, 2.08, 4800, 4800},        // MAN→BCN
		{"AFR1081", "France",         49.01, 2.55, 51.47, -0.45, 2400, 5100},        // CDG→LHR
		{"KLM1024", "Netherlands",    52.31, 4.77, 51.47, -0.45, 2400, 5400},        // AMS→LHR
		{"EIN208",  "Ireland",        53.43, -6.25, 51.47, -0.45, 2700, 5700},       // DUB→LHR
		{"SAS502",  "Norway",         60.19, 11.10, 51.47, -0.45, 4800, 6000},       // OSL→LHR
		{"BAW2721", "United Kingdom", 51.47, -0.45, 55.62, 12.65, 4200, 6300},       // LHR→CPH
		{"DLH902",  "Germany",        50.04, 8.56, 51.47, -0.45, 3600, 6600},        // FRA→LHR
		{"EZY104",  "United Kingdom", 51.38, -2.72, 49.01, 2.55, 3000, 6900},        // BRS→CDG
		{"RYR9144", "United Kingdom", 55.04, -1.69, 41.30, 2.08, 5400, 7200},        // NCL→BCN
	},
}

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for demo
		},
	}
	clients      = make(map[*websocket.Conn]string) // conn -> region
	clientsMutex sync.RWMutex
	airspaceCache = make(map[string]*AirspaceData)
	cacheMutex   sync.RWMutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start simulated aircraft traffic for both regions
	go simulateAircraftTraffic("socal", 2*time.Second)
	go simulateAircraftTraffic("europe", 2*time.Second)

	// Start background AI analysis
	go runTacticalAnalysis("socal", 30*time.Second)
	go runTacticalAnalysis("europe", 30*time.Second)

	// Start drone simulator
	droneFleet = fprime.NewFleet()
	droneSim = fprime.NewSimulator(droneFleet, fprime.DefaultSimConfig())
	droneSim.Start()
	log.Println("🚁 Drone simulator started (3 drones in formation)")

	mux := http.NewServeMux()

	// WebSocket endpoints
	mux.HandleFunc("/ws", handleWebSocket)
	mux.HandleFunc("/ws/drones", handleDroneWebSocket)

	// REST endpoints
	mux.HandleFunc("/api/aircraft", handleGetAircraft)
	mux.HandleFunc("/api/regions", handleGetRegions)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/analysis", handleGetAnalysis)
	mux.HandleFunc("/api/analyze", handleRunAnalysis)

	// Drone API endpoints
	mux.HandleFunc("/api/drones", handleGetDrones)
	mux.HandleFunc("/api/drones/telemetry", handleGetDroneTelemetry)
	mux.HandleFunc("/api/drones/events", handleGetDroneEvents)
	mux.HandleFunc("/api/drones/fsm", handleGetDroneFSM)
	mux.HandleFunc("/api/drones/config", handleDroneConfig)
	mux.HandleFunc("/api/drones/validate", handleDroneValidate)

	// Serve static files from frontend build (for production)
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(mux)

	log.Printf("Swarm C2 Backend starting on port %s", port)
	log.Printf("WebSocket: ws://localhost:%s/ws", port)
	log.Printf("Drone WS: ws://localhost:%s/ws/drones", port)
	log.Printf("REST API: http://localhost:%s/api/aircraft?region=socal", port)
	log.Printf("Drone API: http://localhost:%s/api/drones", port)
	log.Printf("AI Analysis: http://localhost:%s/api/analysis?region=socal", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

// runTacticalAnalysis periodically analyzes aircraft data
func runTacticalAnalysis(regionName string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial analysis after first data fetch
	time.Sleep(15 * time.Second)
	performAnalysis(regionName)

	for range ticker.C {
		performAnalysis(regionName)
	}
}

func performAnalysis(regionName string) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Printf("[%s] ANTHROPIC_API_KEY not set, skipping analysis", regionName)
		return
	}

	// Get cached aircraft data
	cacheMutex.RLock()
	data, exists := airspaceCache[regionName]
	cacheMutex.RUnlock()

	if !exists || len(data.Aircraft) == 0 {
		log.Printf("[%s] No aircraft data for analysis", regionName)
		return
	}

	analysis, err := callAnthropicAnalysis(apiKey, regionName, data.Aircraft)
	if err != nil {
		log.Printf("[%s] AI analysis error: %v", regionName, err)
		return
	}

	// Cache the analysis
	analysisCacheMutex.Lock()
	analysisCache[regionName] = analysis
	analysisCacheMutex.Unlock()

	log.Printf("[%s] AI Analysis complete: %s (Score: %d)", regionName, analysis.OverallThreatLevel, analysis.ThreatScore)

	// Broadcast analysis to WebSocket clients
	broadcastAnalysisToClients(regionName, analysis)
}

func callAnthropicAnalysis(apiKey string, region string, aircraft []Aircraft) (*TacticalAnalysis, error) {
	// Prepare aircraft data summary for the prompt
	aircraftJSON, _ := json.MarshalIndent(aircraft, "", "  ")

	userPrompt := fmt.Sprintf(`Analyze the following real-time aircraft tracking data for the %s region.

Current timestamp: %s
Total aircraft tracked: %d

Aircraft Data:
%s

Provide your tactical analysis in the specified JSON format.`,
		region,
		time.Now().UTC().Format(time.RFC3339),
		len(aircraft),
		string(aircraftJSON),
	)

	reqBody := AnthropicRequest{
		Model:       "claude-sonnet-4-20250514",
		MaxTokens:   2000,
		System:      TACTICAL_SYSTEM_PROMPT,
		Messages: []AnthropicMessage{
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if anthropicResp.Error != nil {
		return nil, fmt.Errorf("Anthropic error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no response content")
	}

	// Parse the JSON response from the AI
	content := anthropicResp.Content[0].Text
	
	// Try to extract JSON from the response (may be wrapped in markdown)
	jsonStart := 0
	jsonEnd := len(content)
	if idx := findJSONStart(content); idx >= 0 {
		jsonStart = idx
	}
	if idx := findJSONEnd(content[jsonStart:]); idx >= 0 {
		jsonEnd = jsonStart + idx + 1
	}
	
	jsonContent := content[jsonStart:jsonEnd]

	var analysis TacticalAnalysis
	if err := json.Unmarshal([]byte(jsonContent), &analysis); err != nil {
		// If parsing fails, return a basic analysis with the raw content
		return &TacticalAnalysis{
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
			Region:             region,
			OverallThreatLevel: "UNKNOWN",
			ThreatScore:        0,
			Summary:            "Analysis parsing failed - raw response available",
			Raw:                content,
		}, nil
	}

	analysis.Timestamp = time.Now().UTC().Format(time.RFC3339)
	analysis.Region = region

	return &analysis, nil
}

func findJSONStart(s string) int {
	for i, c := range s {
		if c == '{' {
			return i
		}
	}
	return -1
}

func findJSONEnd(s string) int {
	depth := 0
	for i, c := range s {
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func broadcastAnalysisToClients(region string, analysis *TacticalAnalysis) {
	message := map[string]interface{}{
		"type":     "analysis",
		"region":   region,
		"analysis": analysis,
	}

	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	for conn, clientRegion := range clients {
		if clientRegion == region {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("Write analysis to client failed: %v", err)
			}
		}
	}
}

func handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "socal"
	}

	analysisCacheMutex.RLock()
	analysis, exists := analysisCache[region]
	analysisCacheMutex.RUnlock()

	if !exists {
		// Return empty analysis if none cached
		analysis = &TacticalAnalysis{
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
			Region:             region,
			OverallThreatLevel: "NOMINAL",
			ThreatScore:        0,
			Summary:            "Awaiting initial analysis...",
			NextUpdatePriority: "NORMAL",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

func handleRunAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	region := r.URL.Query().Get("region")
	if region == "" {
		region = "socal"
	}

	// Run analysis synchronously
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		http.Error(w, "ANTHROPIC_API_KEY not configured", http.StatusServiceUnavailable)
		return
	}

	cacheMutex.RLock()
	data, exists := airspaceCache[region]
	cacheMutex.RUnlock()

	if !exists || len(data.Aircraft) == 0 {
		http.Error(w, "No aircraft data available", http.StatusServiceUnavailable)
		return
	}

	analysis, err := callAnthropicAnalysis(apiKey, region, data.Aircraft)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update cache
	analysisCacheMutex.Lock()
	analysisCache[region] = analysis
	analysisCacheMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// simulateAircraftTraffic generates and broadcasts simulated flight positions
func simulateAircraftTraffic(regionName string, interval time.Duration) {
	routes, ok := simRoutes[regionName]
	if !ok {
		return
	}

	log.Printf("[%s] Aircraft simulator started (%d routes)", regionName, len(routes))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		nowUnix := now.Unix()
		t := float64(nowUnix)

		var aircraft []Aircraft

		for i, route := range routes {
			// Each flight cycles along its route with its own period and phase
			progress := math.Mod((t+route.PhaseOffset), route.CycleSec) / route.CycleSec
			// Bounce: go out 0→1, then return 1→0
			if progress > 0.5 {
				progress = 1.0 - (progress-0.5)*2
			} else {
				progress = progress * 2
			}
			// Clamp to in-flight range
			progress = 0.05 + progress*0.9

			lat, lon := greatCircleInterpolate(
				route.DepLat, route.DepLon,
				route.ArrLat, route.ArrLon,
				progress,
			)
			bearing := greatCircleBearing(lat, lon, route.ArrLat, route.ArrLon)
			alt := estimateAltitude(progress)
			speed := estimateSpeed(progress)

			icao24 := fmt.Sprintf("%06x", (i*7919+42)%0xFFFFFF)
			vertRate := 0.0
			if progress < 0.15 {
				vertRate = 15.0
			} else if progress > 0.85 {
				vertRate = -12.0
			}

			ac := Aircraft{
				ICAO24:        icao24,
				Callsign:      route.Callsign,
				OriginCountry: route.OriginCountry,
				TimePosition:  &nowUnix,
				LastContact:   nowUnix,
				Longitude:     &lon,
				Latitude:      &lat,
				BaroAltitude:  &alt,
				OnGround:      false,
				Velocity:      &speed,
				TrueTrack:     &bearing,
				VerticalRate:  &vertRate,
				GeoAltitude:   &alt,
			}
			aircraft = append(aircraft, ac)
		}

		data := &AirspaceData{
			Timestamp: nowUnix,
			Aircraft:  aircraft,
			Region:    regionName,
			Count:     len(aircraft),
		}

		cacheMutex.Lock()
		airspaceCache[regionName] = data
		cacheMutex.Unlock()

		broadcastToClients(regionName, data)
	}
}

// greatCircleInterpolate returns lat/lon at fraction t along great circle from A to B
func greatCircleInterpolate(lat1, lon1, lat2, lon2, t float64) (float64, float64) {
	lat1R := lat1 * math.Pi / 180
	lon1R := lon1 * math.Pi / 180
	lat2R := lat2 * math.Pi / 180
	lon2R := lon2 * math.Pi / 180

	d := math.Acos(math.Sin(lat1R)*math.Sin(lat2R) + math.Cos(lat1R)*math.Cos(lat2R)*math.Cos(lon2R-lon1R))

	if d < 0.0001 {
		return lat1 + t*(lat2-lat1), lon1 + t*(lon2-lon1)
	}

	a := math.Sin((1-t)*d) / math.Sin(d)
	b := math.Sin(t*d) / math.Sin(d)

	x := a*math.Cos(lat1R)*math.Cos(lon1R) + b*math.Cos(lat2R)*math.Cos(lon2R)
	y := a*math.Cos(lat1R)*math.Sin(lon1R) + b*math.Cos(lat2R)*math.Sin(lon2R)
	z := a*math.Sin(lat1R) + b*math.Sin(lat2R)

	lat := math.Atan2(z, math.Sqrt(x*x+y*y)) * 180 / math.Pi
	lon := math.Atan2(y, x) * 180 / math.Pi

	return lat, lon
}

// greatCircleBearing returns the bearing from (lat1,lon1) to (lat2,lon2) in degrees
func greatCircleBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1R := lat1 * math.Pi / 180
	lat2R := lat2 * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(dLon) * math.Cos(lat2R)
	x := math.Cos(lat1R)*math.Sin(lat2R) - math.Sin(lat1R)*math.Cos(lat2R)*math.Cos(dLon)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}

// estimateAltitude returns altitude in meters based on flight progress
func estimateAltitude(progress float64) float64 {
	cruiseAlt := 10668.0 // ~35000 ft in meters
	if progress < 0.15 {
		return cruiseAlt * (progress / 0.15)
	} else if progress > 0.85 {
		return cruiseAlt * ((1 - progress) / 0.15)
	}
	return cruiseAlt
}

// estimateSpeed returns speed in m/s based on flight progress
func estimateSpeed(progress float64) float64 {
	cruiseSpeed := 231.5 // ~450 kts in m/s
	if progress < 0.1 {
		return cruiseSpeed * 0.6
	} else if progress > 0.9 {
		return cruiseSpeed * 0.5
	}
	return cruiseSpeed
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Default to Taiwan region
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "socal"
	}

	clientsMutex.Lock()
	clients[conn] = region
	clientsMutex.Unlock()

	log.Printf("Client connected, subscribed to: %s", region)

	// Send initial cached data if available
	cacheMutex.RLock()
	if data, exists := airspaceCache[region]; exists {
		conn.WriteJSON(data)
	}
	cacheMutex.RUnlock()

	// Handle incoming messages (for region switching)
	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
		conn.Close()
		log.Println("Client disconnected")
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Handle region switch requests
		var request struct {
			Action string `json:"action"`
			Region string `json:"region"`
		}
		if json.Unmarshal(msg, &request) == nil && request.Action == "subscribe" {
			clientsMutex.Lock()
			clients[conn] = request.Region
			clientsMutex.Unlock()

			// Send cached data for new region
			cacheMutex.RLock()
			if data, exists := airspaceCache[request.Region]; exists {
				conn.WriteJSON(data)
			}
			cacheMutex.RUnlock()

			log.Printf("Client switched to region: %s", request.Region)
		}
	}
}

func broadcastToClients(region string, data *AirspaceData) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	for conn, clientRegion := range clients {
		if clientRegion == region {
			if err := conn.WriteJSON(data); err != nil {
				log.Printf("Write to client failed: %v", err)
			}
		}
	}
}

func handleGetAircraft(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "socal"
	}

	cacheMutex.RLock()
	data, exists := airspaceCache[region]
	cacheMutex.RUnlock()

	if !exists {
		data = &AirspaceData{
			Timestamp: time.Now().Unix(),
			Aircraft:  []Aircraft{},
			Region:    region,
			Count:     0,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleGetRegions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(regions)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"regions":   len(regions),
	})
}

// ========================= DRONE OPS =========================

var (
	droneFleet      *fprime.Fleet
	droneSim        *fprime.Simulator
	droneClients      = make(map[*websocket.Conn]bool)
	droneClientsMutex sync.RWMutex
)

func init() {
	// Start broadcasting drone telemetry to WS clients
	go broadcastDroneTelemetry()
}

func broadcastDroneTelemetry() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if droneFleet == nil {
			continue
		}

		drones := droneFleet.GetAllDrones()
		events := droneFleet.GetEvents("", 10)

		msg := map[string]interface{}{
			"type":   "drone_telemetry",
			"drones": drones,
			"events": events,
			"timestamp": time.Now().Unix(),
		}

		droneClientsMutex.RLock()
		for conn := range droneClients {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("Drone WS write error: %v", err)
			}
		}
		droneClientsMutex.RUnlock()
	}
}

func handleDroneWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Drone WebSocket upgrade failed: %v", err)
		return
	}

	droneClientsMutex.Lock()
	droneClients[conn] = true
	droneClientsMutex.Unlock()

	log.Println("Drone WS client connected")

	// Send initial state
	if droneFleet != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":   "drone_telemetry",
			"drones": droneFleet.GetAllDrones(),
			"events": droneFleet.GetEvents("", 50),
			"timestamp": time.Now().Unix(),
		})
	}

	defer func() {
		droneClientsMutex.Lock()
		delete(droneClients, conn)
		droneClientsMutex.Unlock()
		conn.Close()
		log.Println("Drone WS client disconnected")
	}()

	// Keep connection alive, read messages (unused for now)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func handleGetDrones(w http.ResponseWriter, r *http.Request) {
	if droneFleet == nil {
		http.Error(w, "Drone fleet not initialized", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(droneFleet.GetAllDrones())
}

func handleGetDroneTelemetry(w http.ResponseWriter, r *http.Request) {
	if droneFleet == nil {
		http.Error(w, "Drone fleet not initialized", http.StatusServiceUnavailable)
		return
	}
	droneID := r.URL.Query().Get("drone_id")
	if droneID != "" {
		drone := droneFleet.GetDrone(droneID)
		if drone == nil {
			http.Error(w, "Drone not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(drone)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(droneFleet.GetAllDrones())
}

func handleGetDroneEvents(w http.ResponseWriter, r *http.Request) {
	if droneFleet == nil {
		http.Error(w, "Drone fleet not initialized", http.StatusServiceUnavailable)
		return
	}
	droneID := r.URL.Query().Get("drone_id")
	limit := 50
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(droneFleet.GetEvents(droneID, limit))
}

func handleGetDroneFSM(w http.ResponseWriter, r *http.Request) {
	droneID := r.URL.Query().Get("drone_id")
	result := map[string]interface{}{
		"states":          fprime.AllStates,
		"inputs":          fprime.AllInputs,
		"transitionTable": fprime.TransitionTable,
	}
	if droneFleet != nil && droneID != "" {
		drone := droneFleet.GetDrone(droneID)
		if drone != nil {
			result["currentState"] = drone.FSMState
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleDroneConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config fprime.DroneConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate first
	results := fprime.ValidateConfig(config, droneFleet)
	allPass := true
	for _, r := range results {
		if !r.Pass {
			allPass = false
			break
		}
	}

	if !allPass {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Validation failed",
			"validation": results,
		})
		return
	}

	// Apply config (in simulation, just update budget limit)
	drone := droneFleet.GetDrone(config.DroneID)
	if drone != nil {
		drone.Energy.BudgetLimit = config.EnergyBudgetLimit
		droneFleet.UpdateDrone(drone)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "deployed",
		"validation": results,
	})
}

func handleDroneValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config fprime.DroneConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	results := fprime.ValidateConfig(config, droneFleet)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
