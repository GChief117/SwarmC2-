import React, { useState, useEffect, useRef, useCallback } from 'react';
import FlightMap from './components/FlightMap';
import Globe3D from './components/Globe3D';
import TelemetryPanel from './components/TelemetryPanel';
import AircraftList from './components/AircraftList';
import AIAnalysisPanel from './components/AIAnalysisPanel';
import Clock from './components/Clock';

const REGIONS = {
  taiwan: { name: 'Taiwan Strait', center: [120.5, 23.5], zoom: 6 },
  socal: { name: 'Southern California', center: [-118.5, 33.5], zoom: 7 },
  europe: { name: 'United Kingdom', center: [-2.5, 54.5], zoom: 5.5 },
};

function App() {
  const [aircraft, setAircraft] = useState([]);
  const [selectedAircraft, setSelectedAircraft] = useState(null);
  const [region, setRegion] = useState('taiwan');
  const [viewMode, setViewMode] = useState('map');
  const [connected, setConnected] = useState(false);
  const [lastUpdate, setLastUpdate] = useState(null);
  const [aiAnalysis, setAiAnalysis] = useState(null);
  const [aiLoading, setAiLoading] = useState(false);
  const [dataLoading, setDataLoading] = useState(true);  // True until first aircraft data arrives
  const wsRef = useRef(null);
  const reconnectTimer = useRef(null);
  const intentionalClose = useRef(false);
  const regionRef = useRef(region);  // Always tracks current region for WS filtering

  const getApiBaseUrl = useCallback(() => {
    return import.meta.env.DEV ? 'http://localhost:8080' : '';
  }, []);

  // Stable WS URL — region is NOT in the URL, we use subscribe messages instead
  const getWsUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = import.meta.env.DEV ? 'localhost:8080' : window.location.host;
    return `${protocol}//${host}/ws`;
  }, []);

  const fetchAnalysis = useCallback(async (rgn) => {
    try {
      const response = await fetch(`${getApiBaseUrl()}/api/analysis?region=${rgn}`);
      if (response.ok) {
        const data = await response.json();
        setAiAnalysis(data);
      }
    } catch (err) {
      console.error('Failed to fetch analysis:', err);
    }
  }, [getApiBaseUrl]);

  const refreshAnalysis = useCallback(async () => {
    setAiLoading(true);
    try {
      const response = await fetch(`${getApiBaseUrl()}/api/analyze?region=${region}`, {
        method: 'POST',
      });
      if (response.ok) {
        const data = await response.json();
        setAiAnalysis(data);
      }
    } catch (err) {
      console.error('Failed to refresh analysis:', err);
    } finally {
      setAiLoading(false);
    }
  }, [region, getApiBaseUrl]);

  // Single WebSocket connection — survives region changes
  useEffect(() => {
    const connect = () => {
      // Clear any pending reconnect
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }

      intentionalClose.current = false;
      const ws = new WebSocket(`${getWsUrl()}?region=taiwan`);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('WebSocket connected');
        setConnected(true);
        fetchAnalysis('taiwan');
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          if (data.type === 'analysis') {
            // Only accept analysis for current region
            if (!data.region || data.region === regionRef.current) {
              setAiAnalysis(data.analysis);
            }
          } else if (data.aircraft) {
            // Only accept aircraft data for current region
            if (!data.region || data.region === regionRef.current) {
              setAircraft(data.aircraft || []);
              setLastUpdate(new Date(data.timestamp * 1000));
              setDataLoading(false);
            }
          }
        } catch (err) {
          console.error('Parse error:', err);
        }
      };

      ws.onclose = () => {
        console.log('WebSocket disconnected');
        setConnected(false);
        wsRef.current = null;
        // Only auto-reconnect if this wasn't an intentional close
        if (!intentionalClose.current) {
          reconnectTimer.current = setTimeout(connect, 3000);
        }
      };

      ws.onerror = (err) => {
        console.error('WebSocket error:', err);
      };
    };

    connect();

    return () => {
      intentionalClose.current = true;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
        reconnectTimer.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [getWsUrl, fetchAnalysis]);

  // Handle region change — just send subscribe, don't recreate WS
  const handleRegionChange = (newRegion) => {
    if (newRegion === region) return;

    // Update ref FIRST so WS message filter uses new region immediately
    regionRef.current = newRegion;

    // Clear stale data
    setAircraft([]);
    setSelectedAircraft(null);
    setAiAnalysis(null);
    setDataLoading(true);
    setRegion(newRegion);

    // Tell server to switch region
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ action: 'subscribe', region: newRegion }));
    }

    // Fetch analysis for new region
    setTimeout(() => fetchAnalysis(newRegion), 500);
  };

  const handleSelectAircraft = (ac) => {
    setSelectedAircraft(ac);
  };

  return (
    <div className="c2-container">
      <header className="c2-header">
        <div className="c2-header-left">
          <div className="c2-logo">
            <span className="c2-logo-icon">◈</span>
            <span className="c2-logo-text">SWARM C2</span>
          </div>
          <div className="c2-status">
            <span className={`c2-status-indicator ${connected ? 'online' : 'offline'}`} />
            <span className="c2-status-text">{connected ? 'CONNECTED' : 'RECONNECTING'}</span>
          </div>
        </div>

        <div className="c2-header-center">
          <div className="c2-region-selector">
            {Object.entries(REGIONS).map(([key, value]) => (
              <button
                key={key}
                className={`c2-region-btn ${region === key ? 'active' : ''}`}
                onClick={() => handleRegionChange(key)}
              >
                {value.name}
              </button>
            ))}
          </div>
        </div>

        <div className="c2-header-right">
          <div className="c2-view-toggle">
            <button
              className={`c2-view-btn ${viewMode === 'map' ? 'active' : ''}`}
              onClick={() => setViewMode('map')}
            >
              2D MAP
            </button>
            <button
              className={`c2-view-btn ${viewMode === 'globe' ? 'active' : ''}`}
              onClick={() => setViewMode('globe')}
            >
              3D GLOBE
            </button>
          </div>
          <div className="c2-stats">
            <div className="c2-stat">
              <span className="c2-stat-value">{aircraft.length}</span>
              <span className="c2-stat-label">TRACKED</span>
            </div>
            <div className="c2-stat">
              <Clock />
              <span className="c2-stat-label">LOCAL TIME</span>
            </div>
          </div>
        </div>
      </header>

      <main className="c2-main">
        <div className="c2-viewport">
          {viewMode === 'map' ? (
            <FlightMap
              aircraft={aircraft}
              region={REGIONS[region]}
              selectedAircraft={selectedAircraft}
              onSelectAircraft={handleSelectAircraft}
            />
          ) : (
            <Globe3D
              aircraft={aircraft}
              region={REGIONS[region]}
              selectedAircraft={selectedAircraft}
              onSelectAircraft={handleSelectAircraft}
            />
          )}

          <div className="c2-overlay-stats">
            <div className="c2-overlay-stat">
              <span className="c2-overlay-label">AIRBORNE</span>
              <span className="c2-overlay-value">{aircraft.filter(a => !a.onGround).length}</span>
            </div>
            <div className="c2-overlay-stat">
              <span className="c2-overlay-label">ON GROUND</span>
              <span className="c2-overlay-value">{aircraft.filter(a => a.onGround).length}</span>
            </div>
          </div>

          {/* Loading overlay */}
          {dataLoading && (
            <div className="c2-loading-overlay">
              <div className="c2-loading-content">
                <div className="c2-loading-radar">
                  <svg viewBox="0 0 80 80" width="80" height="80">
                    <circle cx="40" cy="40" r="35" fill="none" stroke="rgba(0,212,255,0.15)" strokeWidth="1" />
                    <circle cx="40" cy="40" r="24" fill="none" stroke="rgba(0,212,255,0.1)" strokeWidth="0.5" />
                    <circle cx="40" cy="40" r="12" fill="none" stroke="rgba(0,212,255,0.08)" strokeWidth="0.5" />
                    <circle cx="40" cy="40" r="2" fill="rgba(0,212,255,0.4)" />
                    <line x1="40" y1="40" x2="40" y2="5" stroke="rgba(0,212,255,0.6)" strokeWidth="1.5" className="c2-radar-sweep" />
                  </svg>
                </div>
                <div className="c2-loading-text">SCANNING AIRSPACE</div>
                <div className="c2-loading-region">{REGIONS[region]?.name?.toUpperCase()}</div>
              </div>
            </div>
          )}
        </div>

        <aside className="c2-sidebar">
          <AIAnalysisPanel 
            analysis={aiAnalysis}
            onRefresh={refreshAnalysis}
            isLoading={aiLoading}
            onSelectAircraft={handleSelectAircraft}
            aircraft={aircraft}
          />

          {selectedAircraft && (
            <TelemetryPanel
              aircraft={selectedAircraft}
              onClose={() => setSelectedAircraft(null)}
            />
          )}

          <AircraftList
            aircraft={aircraft}
            selectedAircraft={selectedAircraft}
            onSelectAircraft={handleSelectAircraft}
          />
        </aside>
      </main>
    </div>
  );
}

export default App;
