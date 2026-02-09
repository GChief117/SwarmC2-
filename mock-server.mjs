// SWARM C2 - Mock Backend Server with OpenAI Integration
// Run with: node mock-server.mjs

import 'dotenv/config';
import http from 'http';
import { WebSocketServer } from 'ws';

const PORT = 8080;
const OPENAI_API_KEY = process.env.OPENAI_API_KEY;

// Tactical AI System Prompt
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
- Watch for: J-16, J-10, H-6 bomber incursions

**Southern California:**
- Military Operating Areas (MOAs) and restricted zones
- Edwards AFB, Point Mugu, China Lake test ranges
- Commercial traffic corridors (LAX, SAN approaches)
- Watch for: Test flights, military exercises

**United Kingdom:**
- RAF and NATO QRA (Quick Reaction Alert) operations
- North Sea patrol patterns and offshore energy infrastructure
- London FIR and Scottish FIR boundaries
- Military areas: Salisbury Plain, Welsh MOD ranges, North Sea danger areas
- Watch for: Russian long-range aviation probing UK ADIZ, P-8 maritime patrol

## ANALYSIS OUTPUT FORMAT

You MUST respond with ONLY valid JSON in this exact structure (no markdown, no explanation):
{
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
      "priority": 1,
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

1. **Emergency Squawks**: 7500 (hijack), 7600 (comm failure), 7700 (emergency) - Always flag as CRITICAL
2. **Military vs Civilian Ambiguity**: When uncertain, analyze trajectory and behavior patterns
3. **Data Gaps**: Note when aircraft disappear from tracking (potential jamming or low-altitude flight)
4. **Coordinated Activity**: Multiple aircraft with synchronized heading/altitude changes
5. **Shadow Tracking**: Aircraft following same route as another with offset timing
6. **Holding Patterns**: Could indicate reconnaissance or waiting for clearance
7. **Unusual Speed/Altitude**: Jets at low altitude or slow aircraft at high altitude
8. **Border Proximity**: Aircraft approaching ADIZ or FIR boundaries
9. **Night Operations**: Increased suspicion for non-commercial flights at night
10. **Transponder Anomalies**: Squawk changes, intermittent signals

## DECISION CRITERIA

For each aircraft, evaluate:
- Origin country and likely affiliation (military/civilian)
- Current position relative to sensitive areas
- Trajectory and whether it approaches protected assets
- Speed/altitude consistency with declared aircraft type
- Coordination with other aircraft

Be decisive but indicate confidence levels. Prioritize actionable intelligence over excessive caution.`;

// Region bounding boxes
const REGIONS = {
  taiwan: { name: 'Taiwan Strait', minLat: 21.5, maxLat: 26.0, minLon: 117.0, maxLon: 123.0 },
  socal: { name: 'Southern California', minLat: 32.5, maxLat: 34.5, minLon: -120.0, maxLon: -117.0 },
  europe: { name: 'United Kingdom', minLat: 49.9, maxLat: 60.9, minLon: -8.2, maxLon: 1.8 },
};

// Cache
let airspaceCache = {};
let analysisCache = {};

// ─── OpenSky Authentication ───────────────────────────────────────────────
// Supports both OAuth2 (new accounts) and Basic Auth (legacy accounts)
const OPENSKY_CLIENT_ID = process.env.OPENSKY_CLIENT_ID || '';
const OPENSKY_CLIENT_SECRET = process.env.OPENSKY_CLIENT_SECRET || '';
const OPENSKY_USERNAME = process.env.OPENSKY_USERNAME || '';
const OPENSKY_PASSWORD = process.env.OPENSKY_PASSWORD || '';

const hasOAuth2 = OPENSKY_CLIENT_ID && OPENSKY_CLIENT_SECRET;
const hasBasicAuth = OPENSKY_USERNAME && OPENSKY_PASSWORD;
const hasOpenSkyAuth = hasOAuth2 || hasBasicAuth;

// OAuth2 token management
let oauthToken = null;
let oauthTokenExpiry = 0;

async function getOpenSkyToken() {
  if (!hasOAuth2) return null;
  
  // Return cached token if still valid (with 60s buffer)
  if (oauthToken && Date.now() < oauthTokenExpiry - 60000) {
    return oauthToken;
  }

  try {
    const tokenUrl = 'https://auth.opensky-network.org/auth/realms/opensky-network/protocol/openid-connect/token';
    const body = new URLSearchParams({
      grant_type: 'client_credentials',
      client_id: OPENSKY_CLIENT_ID,
      client_secret: OPENSKY_CLIENT_SECRET,
    });

    const resp = await fetch(tokenUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    });

    if (!resp.ok) {
      const errText = await resp.text();
      console.error(`❌ OpenSky OAuth2 token request failed (${resp.status}): ${errText}`);
      return null;
    }

    const data = await resp.json();
    oauthToken = data.access_token;
    // Token expires in `expires_in` seconds (usually 1800 = 30 min)
    oauthTokenExpiry = Date.now() + (data.expires_in || 1800) * 1000;
    console.log(`🔑 OpenSky OAuth2 token acquired (expires in ${data.expires_in}s)`);
    return oauthToken;
  } catch (err) {
    console.error('❌ OpenSky OAuth2 token error:', err.message);
    return null;
  }
}

// Rate limiter: enforce minimum gap between OpenSky calls
let lastOpenSkyCall = 0;
const MIN_GAP_MS = hasOpenSkyAuth ? 3000 : 6000; // 3s authenticated, 6s anonymous
let openSkyQueue = Promise.resolve();

// Fetch real data from OpenSky (rate-limited & sequentialized)
async function fetchOpenSky(region = 'taiwan') {
  // Chain all calls through a single queue to prevent concurrent requests
  const result = openSkyQueue.then(async () => {
    const bounds = REGIONS[region] || REGIONS.taiwan;
    const url = `https://opensky-network.org/api/states/all?lamin=${bounds.minLat}&lomin=${bounds.minLon}&lamax=${bounds.maxLat}&lomax=${bounds.maxLon}`;
    
    try {
      // Enforce minimum gap
      const elapsed = Date.now() - lastOpenSkyCall;
      if (elapsed < MIN_GAP_MS) {
        const wait = MIN_GAP_MS - elapsed;
        await new Promise(r => setTimeout(r, wait));
      }
      lastOpenSkyCall = Date.now();

      const headers = {};
      if (hasOAuth2) {
        const token = await getOpenSkyToken();
        if (token) {
          headers['Authorization'] = `Bearer ${token}`;
        }
      } else if (hasBasicAuth) {
        headers['Authorization'] = 'Basic ' + Buffer.from(`${OPENSKY_USERNAME}:${OPENSKY_PASSWORD}`).toString('base64');
      }

      const response = await fetch(url, { headers });
      
      if (response.status === 429) {
        console.error('⏳ OpenSky rate limited (429) — will retry next cycle');
        return airspaceCache[region] || { timestamp: Math.floor(Date.now() / 1000), region, aircraft: [], count: 0 };
      }
      
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const data = await response.json();
      
      const aircraft = (data.states || []).map(state => ({
        icao24: state[0],
        callsign: state[1]?.trim() || null,
        originCountry: state[2],
        timePosition: state[3],
        lastContact: state[4],
        longitude: state[5],
        latitude: state[6],
        baroAltitude: state[7],
        onGround: state[8],
        velocity: state[9],
        trueTrack: state[10],
        verticalRate: state[11],
        geoAltitude: state[13],
        squawk: state[14],
        spi: state[15],
        positionSource: state[16],
      })).filter(ac => ac.latitude && ac.longitude);

      const cacheResult = {
        timestamp: Math.floor(Date.now() / 1000),
        region,
        aircraft,
        count: aircraft.length,
      };

      airspaceCache[region] = cacheResult;
      return cacheResult;
    } catch (err) {
      console.error('OpenSky fetch error:', err.message);
      return airspaceCache[region] || { timestamp: Math.floor(Date.now() / 1000), region, aircraft: [], count: 0 };
    }
  });
  
  openSkyQueue = result.catch(() => {}); // keep queue alive on errors
  return result;
}

// Call OpenAI for tactical analysis
async function analyzeWithAI(region, aircraft) {
  if (!OPENAI_API_KEY) {
    console.log('⚠ OPENAI_API_KEY not set, skipping AI analysis');
    return null;
  }

  if (!aircraft || aircraft.length === 0) {
    return null;
  }

  const regionInfo = REGIONS[region] || REGIONS.taiwan;
  
  // Summarize aircraft data to reduce tokens (instead of sending full JSON)
  const airborne = aircraft.filter(ac => !ac.onGround);
  const onGround = aircraft.filter(ac => ac.onGround);
  const noCallsign = aircraft.filter(ac => !ac.callsign);
  const highAlt = airborne.filter(ac => ac.baroAltitude > 12000);
  const lowAlt = airborne.filter(ac => ac.baroAltitude < 1000 && ac.baroAltitude > 0);
  const fastMoving = airborne.filter(ac => ac.velocity > 250);
  
  // Get top 15 most interesting aircraft (no callsign, unusual altitude, fast)
  const interestingAircraft = [
    ...noCallsign.slice(0, 5),
    ...lowAlt.slice(0, 3),
    ...fastMoving.slice(0, 5),
    ...airborne.slice(0, 2)
  ].slice(0, 15).map(ac => ({
    icao24: ac.icao24,
    callsign: ac.callsign || 'UNKNOWN',
    origin: ac.originCountry,
    alt: Math.round(ac.baroAltitude || 0),
    speed: Math.round(ac.velocity || 0),
    heading: Math.round(ac.trueTrack || 0),
    vrate: Math.round(ac.verticalRate || 0),
    lat: ac.latitude?.toFixed(3),
    lon: ac.longitude?.toFixed(3),
  }));

  const userPrompt = `TACTICAL ANALYSIS REQUEST - ${regionInfo.name.toUpperCase()}

Time: ${new Date().toISOString()}
Region: Lat ${regionInfo.minLat}-${regionInfo.maxLat}, Lon ${regionInfo.minLon}-${regionInfo.maxLon}

SUMMARY:
- Total: ${aircraft.length} aircraft (${airborne.length} airborne, ${onGround.length} ground)
- No callsign: ${noCallsign.length}
- High altitude (>FL400): ${highAlt.length}
- Low altitude (<1000ft): ${lowAlt.length}
- Fast (>250kts): ${fastMoving.length}

AIRCRAFT OF INTEREST (sample):
${JSON.stringify(interestingAircraft, null, 1)}

Analyze and respond with ONLY JSON.`;

  try {
    console.log(`🤖 Calling OpenAI for ${region} analysis (${aircraft.length} aircraft)...`);
    
    const response = await fetch('https://api.openai.com/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${OPENAI_API_KEY}`,
      },
      body: JSON.stringify({
        model: 'gpt-4.1-mini',  // Much higher rate limits (200k TPM vs 30k)
        messages: [
          { role: 'system', content: TACTICAL_SYSTEM_PROMPT },
          { role: 'user', content: userPrompt },
        ],
        temperature: 0.3,
        max_tokens: 1000,  // Reduced from 2000
      }),
    });

    if (!response.ok) {
      const err = await response.text();
      throw new Error(`OpenAI API error: ${response.status} - ${err}`);
    }

    const data = await response.json();
    const content = data.choices?.[0]?.message?.content || '';
    
    // Extract JSON from response (handle potential markdown wrapping)
    let jsonStr = content;
    const jsonMatch = content.match(/```(?:json)?\s*([\s\S]*?)```/);
    if (jsonMatch) {
      jsonStr = jsonMatch[1];
    } else {
      const braceMatch = content.match(/\{[\s\S]*\}/);
      if (braceMatch) {
        jsonStr = braceMatch[0];
      }
    }

    const analysis = JSON.parse(jsonStr);
    analysis.timestamp = new Date().toISOString();
    analysis.region = region;
    analysisCache[region] = analysis;
    
    return analysis;
  } catch (err) {
    console.error('❌ OpenAI analysis error:', err.message);
    return null;
  }
}

// HTTP Server
const server = http.createServer(async (req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', '*');
  
  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  const url = new URL(req.url, `http://localhost:${PORT}`);
  
  // Health check
  if (url.pathname === '/api/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      status: 'ok', 
      timestamp: Date.now(), 
      aiEnabled: !!OPENAI_API_KEY,
      regions: Object.keys(REGIONS),
    }));
    return;
  }
  
  // Get regions
  if (url.pathname === '/api/regions') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(REGIONS));
    return;
  }
  
  // Get aircraft data
  if (url.pathname === '/api/aircraft') {
    const region = url.searchParams.get('region') || 'taiwan';
    const data = await fetchOpenSky(region);
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(data));
    return;
  }

  // Get cached analysis
  if (url.pathname === '/api/analysis') {
    const region = url.searchParams.get('region') || 'taiwan';
    const analysis = analysisCache[region] || {
      timestamp: new Date().toISOString(),
      region,
      overall_threat_level: 'NOMINAL',
      threat_score: 0,
      summary: OPENAI_API_KEY 
        ? 'Awaiting initial analysis... Click refresh or wait for automatic update.'
        : 'AI analysis disabled - set OPENAI_API_KEY environment variable to enable.',
      key_observations: [],
      aircraft_of_interest: [],
      tactical_recommendations: [],
      pattern_analysis: {
        formations_detected: 0,
        unusual_behaviors: 0,
        potential_threats: 0,
        commercial_density: 'NORMAL',
      },
      next_update_priority: 'NORMAL',
    };
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(analysis));
    return;
  }

  // Trigger fresh analysis
  if (url.pathname === '/api/analyze' && req.method === 'POST') {
    const region = url.searchParams.get('region') || 'taiwan';
    
    if (!OPENAI_API_KEY) {
      res.writeHead(503, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'OPENAI_API_KEY not configured. Set environment variable to enable AI analysis.' }));
      return;
    }

    const cached = airspaceCache[region];
    if (!cached || !cached.aircraft.length) {
      // Try to fetch fresh data first
      const freshData = await fetchOpenSky(region);
      if (!freshData.aircraft.length) {
        res.writeHead(503, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'No aircraft data available for analysis' }));
        return;
      }
    }

    const data = airspaceCache[region];
    console.log(`\n📊 Manual analysis requested for ${region} (${data.aircraft.length} aircraft)`);
    
    const analysis = await analyzeWithAI(region, data.aircraft);
    
    if (analysis) {
      console.log(`✅ Analysis complete: ${analysis.overall_threat_level} (Score: ${analysis.threat_score})`);
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(analysis));
      
      // Broadcast to WebSocket clients
      broadcastAnalysis(region, analysis);
    } else {
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Analysis failed - check server logs' }));
    }
    return;
  }
  
  res.writeHead(404);
  res.end('Not Found');
});

// WebSocket Server
const wss = new WebSocketServer({ server, path: '/ws' });
const clients = new Map();

function broadcastAnalysis(region, analysis) {
  const message = JSON.stringify({ type: 'analysis', region, analysis });
  let count = 0;
  for (const [ws, clientRegion] of clients) {
    if (clientRegion === region && ws.readyState === ws.OPEN) {
      ws.send(message);
      count++;
    }
  }
  if (count > 0) {
    console.log(`📡 Broadcast analysis to ${count} client(s)`);
  }
}

function broadcastAircraft(region, data) {
  const message = JSON.stringify(data);
  for (const [ws, clientRegion] of clients) {
    if (clientRegion === region && ws.readyState === ws.OPEN) {
      ws.send(message);
    }
  }
}

wss.on('connection', (ws, req) => {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const region = url.searchParams.get('region') || 'taiwan';
  clients.set(ws, region);
  
  console.log(`\n🔗 Client connected (region: ${region}, total: ${clients.size})`);
  
  // Send initial aircraft data
  fetchOpenSky(region).then(async (data) => {
    if (ws.readyState === ws.OPEN) {
      ws.send(JSON.stringify(data));
    }
    
    // If no cached analysis, trigger one now
    if (!analysisCache[region] && OPENAI_API_KEY && data.aircraft.length > 0) {
      console.log(`🤖 First connection - analyzing ${region}...`);
      const analysis = await analyzeWithAI(region, data.aircraft);
      if (analysis) {
        console.log(`✅ ${region}: ${analysis.overall_threat_level}`);
        broadcastAnalysis(region, analysis);
      }
    }
  });

  // Send cached analysis if available
  if (analysisCache[region]) {
    ws.send(JSON.stringify({ type: 'analysis', region, analysis: analysisCache[region] }));
  }
  
  ws.on('message', (msg) => {
    try {
      const { action, region: newRegion } = JSON.parse(msg);
      if (action === 'subscribe' && newRegion && REGIONS[newRegion]) {
        clients.set(ws, newRegion);
        console.log(`↔ Client switched to: ${newRegion}`);
        
        // Send data for new region
        fetchOpenSky(newRegion).then(async (data) => {
          if (ws.readyState === ws.OPEN) {
            ws.send(JSON.stringify(data));
          }
          
          // If no cached analysis for new region, trigger one
          if (!analysisCache[newRegion] && OPENAI_API_KEY && data.aircraft.length > 0) {
            console.log(`🤖 Region switch - analyzing ${newRegion}...`);
            const analysis = await analyzeWithAI(newRegion, data.aircraft);
            if (analysis) {
              console.log(`✅ ${newRegion}: ${analysis.overall_threat_level}`);
              broadcastAnalysis(newRegion, analysis);
            }
          }
        });
        
        // Send cached analysis if available
        if (analysisCache[newRegion]) {
          ws.send(JSON.stringify({ type: 'analysis', region: newRegion, analysis: analysisCache[newRegion] }));
        }
      }
    } catch (e) {}
  });
  
  ws.on('close', () => {
    clients.delete(ws);
    console.log(`🔌 Client disconnected (remaining: ${clients.size})`);
  });
});

// Background: Fetch aircraft data every 15s (sequential per region to respect rate limits)
const POLL_INTERVAL = hasOpenSkyAuth ? 10000 : 15000;

setInterval(async () => {
  const activeRegions = [...new Set(clients.values())];
  if (activeRegions.length === 0) return;
  
  // Fetch regions one at a time (rate limiter handles the gap)
  for (const region of activeRegions) {
    const data = await fetchOpenSky(region);
    console.log(`[${region}] 📍 ${data.count} aircraft tracked`);
    broadcastAircraft(region, data);
  }
}, POLL_INTERVAL);

// Background: Run AI analysis every 60 seconds ONLY for regions with active clients
if (OPENAI_API_KEY) {
  setInterval(async () => {
    const activeRegions = [...new Set(clients.values())];
    if (activeRegions.length === 0) return;
    
    // Only analyze regions that have connected clients
    for (const region of activeRegions) {
      const cached = airspaceCache[region];
      if (cached && cached.aircraft.length > 0) {
        console.log(`\n⏰ Scheduled analysis for ${region} (${clients.size} clients)...`);
        const analysis = await analyzeWithAI(region, cached.aircraft);
        if (analysis) {
          console.log(`✅ ${region}: ${analysis.overall_threat_level} (Score: ${analysis.threat_score})`);
          broadcastAnalysis(region, analysis);
        }
        // Only analyze one region per interval to avoid rate limits
        break;
      }
    }
  }, 60000);
}

server.listen(PORT, () => {
  const openSkyStatus = hasOAuth2 
    ? `✅ OAuth2 (client: ${OPENSKY_CLIENT_ID.slice(0,12)}...)`
    : hasBasicAuth 
      ? `✅ Basic Auth (${OPENSKY_USERNAME})`
      : `⚠️  Anonymous (add OPENSKY_CLIENT_ID + SECRET for 10x credits)`;

  console.log(`
╔══════════════════════════════════════════════════════════════╗
║              SWARM C2 - Mock Backend Server                  ║
╠══════════════════════════════════════════════════════════════╣
║  HTTP Server:  http://localhost:${PORT}                         ║
║  WebSocket:    ws://localhost:${PORT}/ws                        ║
╠══════════════════════════════════════════════════════════════╣
║  AI Status:    ${OPENAI_API_KEY ? '✅ ENABLED (gpt-4.1-mini)' : '❌ DISABLED (set OPENAI_API_KEY)'}          ║
║  OpenSky:      ${openSkyStatus}  ║
║  Poll Rate:    Every ${POLL_INTERVAL/1000}s per region                            ║
╠══════════════════════════════════════════════════════════════╣
║  Endpoints:                                                  ║
║    GET  /api/health          - Server status                 ║
║    GET  /api/regions         - Available regions             ║
║    GET  /api/aircraft        - Aircraft data                 ║
║    GET  /api/analysis        - Latest AI analysis            ║
║    POST /api/analyze         - Trigger fresh analysis        ║
╚══════════════════════════════════════════════════════════════╝
  `);
});
