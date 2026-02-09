import React, { useEffect, useRef, useState, useCallback } from 'react';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import AircraftPopup from './AircraftPopup';

function FlightMap({ aircraft, region, selectedAircraft, onSelectAircraft }) {
  const mapContainer = useRef(null);
  const map = useRef(null);
  const markersRef = useRef({});
  const [mapReady, setMapReady] = useState(false);
  const [mapInstance, setMapInstance] = useState(null);
  const [povMode, setPovMode] = useState(false);
  const [povAircraft, setPovAircraft] = useState(null);
  const povAcRef = useRef(null);
  const regionRef = useRef(region);
  const prevRegionRef = useRef(region);
  const onSelectRef = useRef(onSelectAircraft);
  const aircraftMapRef = useRef(new Map());
  const terrainAdded = useRef(false);
  const savedView = useRef(null);

  const MAPTILER_KEY = import.meta.env.VITE_MAPTILER_KEY;
  const hasMaptiler = MAPTILER_KEY && MAPTILER_KEY !== 'your_maptiler_key_here';

  useEffect(() => { regionRef.current = region; }, [region]);
  useEffect(() => { onSelectRef.current = onSelectAircraft; }, [onSelectAircraft]);

  const getMapStyle = () => {
    if (hasMaptiler) {
      return `https://api.maptiler.com/maps/hybrid/style.json?key=${MAPTILER_KEY}`;
    }
    return {
      version: 8,
      sources: {
        'nasa-bluemarble': {
          type: 'raster',
          tiles: ['https://gibs.earthdata.nasa.gov/wmts/epsg3857/best/BlueMarble_ShadedRelief_Bathymetry/default//GoogleMapsCompatible_Level8/{z}/{y}/{x}.jpeg'],
          tileSize: 256, maxzoom: 8, attribution: '© NASA GIBS'
        },
        'carto-dark': {
          type: 'raster',
          tiles: [
            'https://a.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
            'https://b.basemaps.cartocdn.com/dark_nolabels/{z}/{x}/{y}@2x.png',
          ],
          tileSize: 256, maxzoom: 20, attribution: '© CARTO'
        }
      },
      layers: [
        { id: 'carto-dark', type: 'raster', source: 'carto-dark', minzoom: 0, maxzoom: 20 },
        { id: 'nasa-bluemarble', type: 'raster', source: 'nasa-bluemarble', minzoom: 0, maxzoom: 8, paint: { 'raster-opacity': 0.6 } },
      ]
    };
  };

  const getColor = (ac, isSelected) => {
    if (isSelected) return '#00ff88';
    if (ac.onGround) return '#ff9500';
    return '#00d4ff';
  };

  const getGlow = (ac, isSelected) => {
    if (isSelected) return 'drop-shadow(0 0 6px rgba(0,255,136,0.8))';
    if (ac.onGround) return 'drop-shadow(0 0 3px rgba(255,149,0,0.5))';
    return 'drop-shadow(0 0 3px rgba(0,212,255,0.5))';
  };

  const createMarkerEl = useCallback((ac, isSelected) => {
    const el = document.createElement('div');
    el.style.width = '22px';
    el.style.height = '22px';
    el.style.cursor = 'pointer';

    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('viewBox', '0 0 24 24');
    svg.style.width = '100%';
    svg.style.height = '100%';
    svg.style.transition = 'transform 0.8s ease, filter 0.3s ease';
    svg.style.transform = `rotate(${ac.trueTrack || 0}deg)`;
    svg.style.filter = getGlow(ac, isSelected);

    const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    path.setAttribute('d', 'M12 2L4 12L6 14L10 12V20L8 21V22H16V21L14 20V12L18 14L20 12L12 2Z');
    path.setAttribute('fill', getColor(ac, isSelected));
    path.setAttribute('stroke', 'rgba(0,0,0,0.5)');
    path.setAttribute('stroke-width', '0.5');
    path.style.transition = 'fill 0.3s ease';

    svg.appendChild(path);
    el.appendChild(svg);

    el.addEventListener('click', (e) => {
      e.stopPropagation();
      const liveAc = aircraftMapRef.current.get(ac.icao24);
      if (liveAc) onSelectRef.current(liveAc);
    });

    return el;
  }, []);

  const clearAllMarkers = useCallback(() => {
    Object.values(markersRef.current).forEach(entry => entry.marker.remove());
    markersRef.current = {};
  }, []);

  // --- POV Mode ---
  // Convert aircraft altitude (meters) to MapLibre zoom level
  // At zoom z, camera altitude ≈ 78271484 / 2^z (at equator)
  // Higher altitude = lower zoom. With pitch 85°, we look forward toward horizon.
  const altToZoom = useCallback((altMeters, lat) => {
    const latRad = (lat || 0) * Math.PI / 180;
    // Adjusted formula: account for latitude and pitch
    const baseAlt = 78271484 * Math.cos(latRad);
    const z = Math.log2(baseAlt / Math.max(altMeters, 50));
    // Clamp between sane values
    return Math.max(6, Math.min(18, z));
  }, []);

  const enterPOV = useCallback((ac) => {
    if (!map.current || !mapReady) return;
    const m = map.current;

    // Save current view to restore later
    savedView.current = {
      center: m.getCenter().toArray(),
      zoom: m.getZoom(),
      pitch: m.getPitch(),
      bearing: m.getBearing(),
    };

    // Add terrain source (MapTiler terrain-rgb) for 3D ground
    try {
      if (!terrainAdded.current) {
        const terrainKey = MAPTILER_KEY || '';
        if (terrainKey && !m.getSource('terrain-dem')) {
          m.addSource('terrain-dem', {
            type: 'raster-dem',
            url: `https://api.maptiler.com/tiles/terrain-rgb-v2/tiles.json?key=${terrainKey}`,
          });
        }
        if (m.getSource('terrain-dem')) {
          m.setTerrain({ source: 'terrain-dem', exaggeration: 1.5 });
          terrainAdded.current = true;
        }
      }
    } catch (e) { console.warn('Terrain setup failed:', e); }

    // Max pitch for cockpit view
    m.setMaxPitch(85);

    const heading = ac.trueTrack || 0;
    const altMeters = ac.baroAltitude || ac.geoAltitude || 10000;
    const zoom = altToZoom(altMeters, ac.latitude);

    // Offset center ahead of aircraft so camera looks FORWARD
    // Distance scales with altitude — higher up, further ahead
    const headingRad = (heading * Math.PI) / 180;
    const offsetDeg = Math.max(0.02, altMeters / 200000); // ~50m at low alt, ~50km at FL350
    const lookLng = ac.longitude + Math.sin(headingRad) * offsetDeg;
    const lookLat = ac.latitude + Math.cos(headingRad) * offsetDeg;

    m.flyTo({
      center: [lookLng, lookLat],
      zoom: zoom,
      pitch: 78,
      bearing: heading,
      duration: 2500,
      essential: true,
    });

    povAcRef.current = ac;
    setPovAircraft(ac);
    setPovMode(true);
    onSelectRef.current(null); // close popup
  }, [mapReady, MAPTILER_KEY, altToZoom]);

  const exitPOV = useCallback(() => {
    if (!map.current) return;
    const m = map.current;

    // Remove terrain
    try {
      if (terrainAdded.current) {
        m.setTerrain(null);
        terrainAdded.current = false;
      }
    } catch (e) { console.warn('Terrain cleanup failed:', e); }

    m.setMaxPitch(60);

    // Restore saved view
    if (savedView.current) {
      m.flyTo({
        ...savedView.current,
        duration: 1500,
        essential: true,
      });
      savedView.current = null;
    }

    setPovMode(false);
    setPovAircraft(null);
    povAcRef.current = null;
  }, []);

  // Track POV aircraft position as it moves
  useEffect(() => {
    if (!povMode || !povAcRef.current || !map.current || !aircraft) return;

    const id = povAcRef.current.icao24;
    const liveAc = aircraft.find(a => a.icao24 === id);
    if (!liveAc || liveAc.longitude == null) return;

    povAcRef.current = liveAc;
    setPovAircraft(liveAc);

    const heading = liveAc.trueTrack || 0;
    const altMeters = liveAc.baroAltitude || liveAc.geoAltitude || 10000;
    const zoom = altToZoom(altMeters, liveAc.latitude);

    // Look-ahead offset
    const headingRad = (heading * Math.PI) / 180;
    const offsetDeg = Math.max(0.02, altMeters / 200000);
    const lookLng = liveAc.longitude + Math.sin(headingRad) * offsetDeg;
    const lookLat = liveAc.latitude + Math.cos(headingRad) * offsetDeg;

    map.current.easeTo({
      center: [lookLng, lookLat],
      zoom: zoom,
      bearing: heading,
      pitch: 78,
      duration: 3000,
      essential: true,
    });
  }, [aircraft, povMode, altToZoom]);

  // Initialize map (once)
  useEffect(() => {
    if (map.current || !mapContainer.current) return;

    const m = new maplibregl.Map({
      container: mapContainer.current,
      style: getMapStyle(),
      center: regionRef.current.center,
      zoom: regionRef.current.zoom,
      antialias: true,
    });
    map.current = m;
    m.on('load', () => {
      setMapReady(true);
      setMapInstance(m);
    });
    m.addControl(new maplibregl.NavigationControl(), 'bottom-right');

    return () => {
      clearAllMarkers();
      map.current?.remove();
      map.current = null;
    };
  }, []);

  // Fly to region
  useEffect(() => {
    if (!map.current || !mapReady || povMode) return;

    const prev = prevRegionRef.current;
    if (prev.center[0] !== region.center[0] || prev.center[1] !== region.center[1]) {
      clearAllMarkers();
      prevRegionRef.current = region;
    }

    map.current.flyTo({
      center: region.center,
      zoom: region.zoom,
      duration: 1500,
      essential: true,
    });
  }, [region.center[0], region.center[1], region.zoom, mapReady, clearAllMarkers, povMode]);

  // Update markers (skip in POV mode)
  useEffect(() => {
    if (!map.current || !mapReady || !aircraft) return;

    const liveMap = new Map();
    aircraft.forEach(ac => liveMap.set(ac.icao24, ac));
    aircraftMapRef.current = liveMap;

    // In POV mode, hide all markers for cleaner view
    if (povMode) {
      Object.values(markersRef.current).forEach(e => e.marker.getElement().style.display = 'none');
      return;
    }

    const entries = markersRef.current;
    const activeIds = new Set();

    aircraft.forEach((ac) => {
      if (ac.longitude == null || ac.latitude == null) return;

      const id = ac.icao24;
      activeIds.add(id);
      const isSelected = selectedAircraft?.icao24 === id;
      const heading = ac.trueTrack || 0;
      const color = getColor(ac, isSelected);

      if (entries[id]) {
        entries[id].marker.setLngLat([ac.longitude, ac.latitude]);
        entries[id].marker.getElement().style.display = '';

        const el = entries[id].marker.getElement();
        const svg = el.querySelector('svg');
        const path = el.querySelector('path');

        if (entries[id].heading !== heading && svg) {
          svg.style.transform = `rotate(${heading}deg)`;
          entries[id].heading = heading;
        }
        if (entries[id].color !== color && path) {
          path.setAttribute('fill', color);
          svg.style.filter = getGlow(ac, isSelected);
          entries[id].color = color;
        }
      } else {
        const el = createMarkerEl(ac, isSelected);
        const marker = new maplibregl.Marker({ element: el, anchor: 'center' })
          .setLngLat([ac.longitude, ac.latitude])
          .addTo(map.current);
        entries[id] = { marker, heading, color };
      }
    });

    Object.keys(entries).forEach((id) => {
      if (!activeIds.has(id)) {
        entries[id].marker.remove();
        delete entries[id];
      }
    });
  }, [aircraft, selectedAircraft, mapReady, createMarkerEl, povMode]);

  // HUD data
  const hud = povAircraft ? {
    callsign: povAircraft.callsign || 'UNKNOWN',
    alt: povAircraft.baroAltitude || povAircraft.geoAltitude,
    speed: povAircraft.velocity,
    heading: povAircraft.trueTrack,
    vRate: povAircraft.verticalRate,
  } : null;

  return (
    <>
      <div ref={mapContainer} className="c2-map-container" />

      {/* Popup — only when NOT in POV */}
      {!povMode && (
        <AircraftPopup
          map={mapInstance}
          aircraft={selectedAircraft}
          onClose={() => onSelectRef.current(null)}
          onPOV={enterPOV}
        />
      )}

      {/* POV HUD Overlay */}
      {povMode && hud && (
        <div className="pov-hud">
          {/* Sky gradient to mask black horizon */}
          <div className="pov-sky-gradient" />

          {/* Top bar */}
          <div className="pov-hud-top">
            <div className="pov-hud-callsign">{hud.callsign}</div>
            <div className="pov-hud-label">SYNTHETIC VISION • PILOT POV</div>
            <button className="pov-exit-btn" onClick={exitPOV}>✕ EXIT POV</button>
          </div>

          {/* Bottom instruments */}
          <div className="pov-hud-bottom">
            <div className="pov-instrument">
              <div className="pov-instrument-val">
                {hud.alt ? `FL${Math.round(hud.alt * 3.28084 / 100)}` : '—'}
              </div>
              <div className="pov-instrument-lbl">ALT</div>
            </div>
            <div className="pov-instrument">
              <div className="pov-instrument-val">
                {hud.speed ? Math.round(hud.speed * 1.944) : '—'}
              </div>
              <div className="pov-instrument-lbl">KTS</div>
            </div>
            <div className="pov-instrument pov-instrument-hdg">
              <div className="pov-compass-ring">
                <svg viewBox="0 0 60 60" width="60" height="60">
                  <circle cx="30" cy="30" r="26" fill="none" stroke="rgba(0,212,255,0.2)" strokeWidth="1"/>
                  <text x="30" y="10" textAnchor="middle" fill="rgba(0,212,255,0.6)" fontSize="7" fontFamily="monospace">N</text>
                  <text x="54" y="33" textAnchor="middle" fill="rgba(0,212,255,0.4)" fontSize="7" fontFamily="monospace">E</text>
                  <text x="30" y="56" textAnchor="middle" fill="rgba(0,212,255,0.4)" fontSize="7" fontFamily="monospace">S</text>
                  <text x="6" y="33" textAnchor="middle" fill="rgba(0,212,255,0.4)" fontSize="7" fontFamily="monospace">W</text>
                  <g transform={`rotate(${hud.heading || 0}, 30, 30)`}>
                    <polygon points="30,8 26,22 30,19 34,22" fill="#00d4ff"/>
                  </g>
                </svg>
              </div>
              <div className="pov-instrument-val">{hud.heading != null ? `${Math.round(hud.heading)}°` : '—'}</div>
              <div className="pov-instrument-lbl">HDG</div>
            </div>
            <div className="pov-instrument">
              <div className="pov-instrument-val">
                {hud.vRate != null ? `${hud.vRate > 0 ? '+' : ''}${Math.round(hud.vRate * 196.85)}` : '—'}
              </div>
              <div className="pov-instrument-lbl">FPM</div>
            </div>
            <div className="pov-instrument">
              <div className="pov-instrument-val">
                {hud.speed ? Math.round(hud.speed * 2.237) : '—'}
              </div>
              <div className="pov-instrument-lbl">MPH</div>
            </div>
          </div>

          {/* Crosshair */}
          <div className="pov-crosshair">
            <svg viewBox="0 0 80 40" width="120" height="60">
              <line x1="0" y1="20" x2="30" y2="20" stroke="rgba(0,212,255,0.5)" strokeWidth="1"/>
              <line x1="50" y1="20" x2="80" y2="20" stroke="rgba(0,212,255,0.5)" strokeWidth="1"/>
              <line x1="30" y1="20" x2="35" y2="25" stroke="rgba(0,212,255,0.5)" strokeWidth="1"/>
              <line x1="50" y1="20" x2="45" y2="25" stroke="rgba(0,212,255,0.5)" strokeWidth="1"/>
              <circle cx="40" cy="20" r="2" fill="none" stroke="rgba(0,212,255,0.4)" strokeWidth="0.5"/>
            </svg>
          </div>
        </div>
      )}
    </>
  );
}

export default FlightMap;
