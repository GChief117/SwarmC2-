import React from 'react';

function TelemetryPanel({ aircraft, onClose }) {
  // Format altitude (meters to feet)
  const formatAltitude = (meters) => {
    if (meters == null) return 'N/A';
    const feet = Math.round(meters * 3.28084);
    return feet.toLocaleString();
  };

  // Format velocity (m/s to knots)
  const formatVelocity = (ms) => {
    if (ms == null) return 'N/A';
    const knots = Math.round(ms * 1.944);
    return knots.toLocaleString();
  };

  // Format vertical rate (m/s to ft/min)
  const formatVerticalRate = (ms) => {
    if (ms == null) return 'N/A';
    const ftMin = Math.round(ms * 196.85);
    return ftMin >= 0 ? `+${ftMin.toLocaleString()}` : ftMin.toLocaleString();
  };

  // Format heading
  const formatHeading = (degrees) => {
    if (degrees == null) return 'N/A';
    return `${Math.round(degrees)}°`;
  };

  // Format coordinates
  const formatCoord = (value, isLat) => {
    if (value == null) return 'N/A';
    const dir = isLat ? (value >= 0 ? 'N' : 'S') : (value >= 0 ? 'E' : 'W');
    return `${Math.abs(value).toFixed(4)}° ${dir}`;
  };

  return (
    <div className="c2-telemetry">
      <div className="c2-telemetry-header">
        <div className="c2-telemetry-title">
          <span className="c2-telemetry-callsign">
            {aircraft.callsign?.trim() || 'UNKNOWN'}
          </span>
          <span className="c2-telemetry-icao">{aircraft.icao24.toUpperCase()}</span>
        </div>
        <button className="c2-telemetry-close" onClick={onClose}>
          ✕
        </button>
      </div>

      <div className="c2-telemetry-grid">
        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">ALTITUDE</div>
          <div className="c2-telemetry-value">
            {formatAltitude(aircraft.baroAltitude)}
            <span className="c2-telemetry-unit">FT</span>
          </div>
        </div>

        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">GROUND SPEED</div>
          <div className="c2-telemetry-value">
            {formatVelocity(aircraft.velocity)}
            <span className="c2-telemetry-unit">KTS</span>
          </div>
        </div>

        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">HEADING</div>
          <div className="c2-telemetry-value">
            {formatHeading(aircraft.trueTrack)}
          </div>
        </div>

        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">VERT RATE</div>
          <div className="c2-telemetry-value">
            {formatVerticalRate(aircraft.verticalRate)}
            <span className="c2-telemetry-unit">FT/M</span>
          </div>
        </div>

        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">LATITUDE</div>
          <div className="c2-telemetry-value" style={{ fontSize: '13px' }}>
            {formatCoord(aircraft.latitude, true)}
          </div>
        </div>

        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">LONGITUDE</div>
          <div className="c2-telemetry-value" style={{ fontSize: '13px' }}>
            {formatCoord(aircraft.longitude, false)}
          </div>
        </div>

        <div className="c2-telemetry-item" style={{ gridColumn: 'span 2' }}>
          <div className="c2-telemetry-label">ORIGIN</div>
          <div className="c2-telemetry-value" style={{ fontSize: '14px' }}>
            {aircraft.originCountry || 'Unknown'}
          </div>
        </div>

        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">SQUAWK</div>
          <div className="c2-telemetry-value">
            {aircraft.squawk || 'N/A'}
          </div>
        </div>

        <div className="c2-telemetry-item">
          <div className="c2-telemetry-label">STATUS</div>
          <div className="c2-telemetry-value" style={{ 
            color: aircraft.onGround ? '#ff9500' : '#00ff88' 
          }}>
            {aircraft.onGround ? 'GROUND' : 'AIRBORNE'}
          </div>
        </div>
      </div>
    </div>
  );
}

export default TelemetryPanel;
