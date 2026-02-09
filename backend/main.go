package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"
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

// OpenAI API structures
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
	Error   *struct {
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

// OpenSkyResponse represents the API response from OpenSky Network
type OpenSkyResponse struct {
	Time   int64           `json:"time"`
	States [][]interface{} `json:"states"`
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
	"taiwan": {
		Name:   "Taiwan Strait",
		MinLat: 21.5,
		MaxLat: 26.0,
		MinLon: 117.0,
		MaxLon: 123.0,
	},
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
	"global": {
		Name:   "Global",
		MinLat: -90.0,
		MaxLat: 90.0,
		MinLon: -180.0,
		MaxLon: 180.0,
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

	// Global rate limiter — ensures only 1 OpenSky request at a time
	// with minimum gap between requests
	openSkyMutex    sync.Mutex
	lastOpenSkyCall time.Time

	// OAuth2 token management for OpenSky
	oauthToken      string
	oauthTokenExpiry time.Time
	oauthTokenMutex  sync.Mutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	openSkyClientID := os.Getenv("OPENSKY_CLIENT_ID")
	openSkyUser := os.Getenv("OPENSKY_USERNAME")
	openSkyPass := os.Getenv("OPENSKY_PASSWORD")
	
	pollInterval := 15 * time.Second
	if openSkyClientID != "" {
		pollInterval = 10 * time.Second
		log.Printf("✅ OpenSky OAuth2 mode (client: %s...) — poll every %v", openSkyClientID[:min(12, len(openSkyClientID))], pollInterval)
	} else if openSkyUser != "" && openSkyPass != "" {
		pollInterval = 10 * time.Second
		log.Printf("✅ OpenSky Basic Auth as: %s — poll every %v", openSkyUser, pollInterval)
	} else {
		log.Println("⚠️  OpenSky anonymous mode — limited to 400 credits/day")
		log.Println("   Add OPENSKY_CLIENT_ID + OPENSKY_CLIENT_SECRET to .env")
	}

	// Start background polling — stagger by half the interval
	go pollOpenSky("taiwan", pollInterval)
	go func() {
		time.Sleep(pollInterval / 2) // offset to avoid simultaneous requests
		pollOpenSky("socal", pollInterval)
	}()

	// Start background AI analysis
	go runTacticalAnalysis("taiwan", 30*time.Second)
	go runTacticalAnalysis("socal", 30*time.Second)

	mux := http.NewServeMux()
	
	// WebSocket endpoint
	mux.HandleFunc("/ws", handleWebSocket)
	
	// REST endpoints
	mux.HandleFunc("/api/aircraft", handleGetAircraft)
	mux.HandleFunc("/api/regions", handleGetRegions)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/analysis", handleGetAnalysis)
	mux.HandleFunc("/api/analyze", handleRunAnalysis)

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
	log.Printf("REST API: http://localhost:%s/api/aircraft?region=taiwan", port)
	log.Printf("AI Analysis: http://localhost:%s/api/analysis?region=taiwan", port)
	
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
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Printf("[%s] OPENAI_API_KEY not set, skipping analysis", regionName)
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

	analysis, err := callOpenAIAnalysis(apiKey, regionName, data.Aircraft)
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

func callOpenAIAnalysis(apiKey string, region string, aircraft []Aircraft) (*TacticalAnalysis, error) {
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

	reqBody := OpenAIRequest{
		Model: "gpt-4o",
		Messages: []OpenAIMessage{
			{Role: "system", Content: TACTICAL_SYSTEM_PROMPT},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   2000,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("OpenAI error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices")
	}

	// Parse the JSON response from the AI
	content := openAIResp.Choices[0].Message.Content
	
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
		region = "taiwan"
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
		region = "taiwan"
	}

	// Run analysis synchronously
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		http.Error(w, "OPENAI_API_KEY not configured", http.StatusServiceUnavailable)
		return
	}

	cacheMutex.RLock()
	data, exists := airspaceCache[region]
	cacheMutex.RUnlock()

	if !exists || len(data.Aircraft) == 0 {
		http.Error(w, "No aircraft data available", http.StatusServiceUnavailable)
		return
	}

	analysis, err := callOpenAIAnalysis(apiKey, region, data.Aircraft)
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

func pollOpenSky(regionName string, interval time.Duration) {
	region, exists := regions[regionName]
	if !exists {
		log.Printf("Unknown region: %s", regionName)
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial fetch
	fetchAndBroadcast(regionName, region)

	for range ticker.C {
		fetchAndBroadcast(regionName, region)
	}
}

func fetchAndBroadcast(regionName string, region Region) {
	aircraft, err := fetchOpenSkyData(region)
	if err != nil {
		log.Printf("Error fetching OpenSky data for %s: %v", regionName, err)
		return
	}

	data := &AirspaceData{
		Timestamp: time.Now().Unix(),
		Aircraft:  aircraft,
		Region:    regionName,
		Count:     len(aircraft),
	}

	// Update cache
	cacheMutex.Lock()
	airspaceCache[regionName] = data
	cacheMutex.Unlock()

	// Broadcast to subscribed clients
	broadcastToClients(regionName, data)

	log.Printf("[%s] Fetched %d aircraft", regionName, len(aircraft))
}

func getOpenSkyToken() (string, error) {
	clientID := os.Getenv("OPENSKY_CLIENT_ID")
	clientSecret := os.Getenv("OPENSKY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("no OAuth2 credentials")
	}

	oauthTokenMutex.Lock()
	defer oauthTokenMutex.Unlock()

	// Return cached token if still valid (60s buffer)
	if oauthToken != "" && time.Now().Before(oauthTokenExpiry.Add(-60*time.Second)) {
		return oauthToken, nil
	}

	tokenURL := "https://auth.opensky-network.org/auth/realms/opensky-network/protocol/openid-connect/token"
	body := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", bytes.NewBufferString(body))
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("token parse failed: %w", err)
	}

	oauthToken = tokenResp.AccessToken
	oauthTokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	log.Printf("🔑 OpenSky OAuth2 token acquired (expires in %ds)", tokenResp.ExpiresIn)
	return oauthToken, nil
}

func fetchOpenSkyData(region Region) ([]Aircraft, error) {
	// Global rate limiter: enforce minimum gap between OpenSky API calls
	openSkyMutex.Lock()
	hasAuth := os.Getenv("OPENSKY_CLIENT_ID") != "" || os.Getenv("OPENSKY_USERNAME") != ""
	minGap := 6 * time.Second
	if hasAuth {
		minGap = 3 * time.Second
	}
	elapsed := time.Since(lastOpenSkyCall)
	if elapsed < minGap {
		wait := minGap - elapsed
		log.Printf("⏳ Rate limiter: waiting %v before next OpenSky call", wait.Round(time.Millisecond))
		time.Sleep(wait)
	}
	lastOpenSkyCall = time.Now()
	openSkyMutex.Unlock()

	url := fmt.Sprintf(
		"https://opensky-network.org/api/states/all?lamin=%.2f&lomin=%.2f&lamax=%.2f&lomax=%.2f",
		region.MinLat, region.MinLon, region.MaxLat, region.MaxLon,
	)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("request build failed: %w", err)
	}

	// Auth priority: OAuth2 > Basic Auth > Anonymous
	if os.Getenv("OPENSKY_CLIENT_ID") != "" {
		token, err := getOpenSkyToken()
		if err != nil {
			log.Printf("⚠️  OAuth2 token error: %v (falling back to anonymous)", err)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	} else if os.Getenv("OPENSKY_USERNAME") != "" && os.Getenv("OPENSKY_PASSWORD") != "" {
		req.SetBasicAuth(os.Getenv("OPENSKY_USERNAME"), os.Getenv("OPENSKY_PASSWORD"))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("OpenSky rate limited (429) — will retry next cycle")
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Clear cached OAuth token on 401
		oauthTokenMutex.Lock()
		oauthToken = ""
		oauthTokenMutex.Unlock()
		return nil, fmt.Errorf("OpenSky auth failed (401) — check credentials")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var openSkyResp OpenSkyResponse
	if err := json.Unmarshal(respBody, &openSkyResp); err != nil {
		return nil, fmt.Errorf("JSON parse failed: %w", err)
	}

	return parseAircraftStates(openSkyResp.States), nil
}

func parseAircraftStates(states [][]interface{}) []Aircraft {
	aircraft := make([]Aircraft, 0, len(states))

	for _, state := range states {
		if len(state) < 17 {
			continue
		}

		ac := Aircraft{
			ICAO24:         getString(state[0]),
			Callsign:       getString(state[1]),
			OriginCountry:  getString(state[2]),
			TimePosition:   getInt64Ptr(state[3]),
			LastContact:    getInt64(state[4]),
			Longitude:      getFloat64Ptr(state[5]),
			Latitude:       getFloat64Ptr(state[6]),
			BaroAltitude:   getFloat64Ptr(state[7]),
			OnGround:       getBool(state[8]),
			Velocity:       getFloat64Ptr(state[9]),
			TrueTrack:      getFloat64Ptr(state[10]),
			VerticalRate:   getFloat64Ptr(state[11]),
			GeoAltitude:    getFloat64Ptr(state[13]),
			Squawk:         getStringPtr(state[14]),
			SPI:            getBool(state[15]),
			PositionSource: getInt(state[16]),
		}

		// Only include aircraft with valid positions
		if ac.Latitude != nil && ac.Longitude != nil {
			aircraft = append(aircraft, ac)
		}
	}

	return aircraft
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
		region = "taiwan"
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
		region = "taiwan"
	}

	cacheMutex.RLock()
	data, exists := airspaceCache[region]
	cacheMutex.RUnlock()

	if !exists {
		// Fetch fresh if not cached
		regionDef, ok := regions[region]
		if !ok {
			http.Error(w, "Unknown region", http.StatusBadRequest)
			return
		}
		aircraft, err := fetchOpenSkyData(regionDef)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = &AirspaceData{
			Timestamp: time.Now().Unix(),
			Aircraft:  aircraft,
			Region:    region,
			Count:     len(aircraft),
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

// Helper functions for type conversion
func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getStringPtr(v interface{}) *string {
	if s, ok := v.(string); ok {
		return &s
	}
	return nil
}

func getFloat64Ptr(v interface{}) *float64 {
	if f, ok := v.(float64); ok {
		return &f
	}
	return nil
}

func getInt64Ptr(v interface{}) *int64 {
	if f, ok := v.(float64); ok {
		i := int64(f)
		return &i
	}
	return nil
}

func getInt64(v interface{}) int64 {
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}

func getInt(v interface{}) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

func getBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
