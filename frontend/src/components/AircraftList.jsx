import React, { useMemo } from 'react';

function AircraftList({ aircraft, selectedAircraft, onSelectAircraft }) {
  // Sort aircraft: airborne first, then by altitude descending
  const sortedAircraft = useMemo(() => {
    return [...aircraft].sort((a, b) => {
      // Airborne aircraft first
      if (a.onGround !== b.onGround) {
        return a.onGround ? 1 : -1;
      }
      // Then by altitude (highest first)
      const altA = a.baroAltitude || 0;
      const altB = b.baroAltitude || 0;
      return altB - altA;
    });
  }, [aircraft]);

  // Format altitude for display
  const formatAltitude = (meters) => {
    if (meters == null) return '--';
    const feet = Math.round(meters * 3.28084);
    if (feet >= 10000) {
      return `FL${Math.round(feet / 100)}`;
    }
    return `${feet.toLocaleString()}`;
  };

  return (
    <div className="c2-aircraft-list">
      <div className="c2-aircraft-header">
        <span className="c2-aircraft-title">TRACKED AIRCRAFT</span>
      </div>

      <div className="c2-aircraft-scroll">
        {sortedAircraft.length === 0 ? (
          <div className="c2-empty">
            <div className="c2-empty-icon">✈</div>
            <div className="c2-empty-text">NO AIRCRAFT DETECTED</div>
          </div>
        ) : (
          sortedAircraft.map((ac) => (
            <div
              key={ac.icao24}
              className={`c2-aircraft-item ${
                selectedAircraft?.icao24 === ac.icao24 ? 'selected' : ''
              } ${ac.onGround ? 'on-ground' : ''}`}
              onClick={() => onSelectAircraft(ac)}
            >
              <div className="c2-aircraft-icon">
                {ac.onGround ? '◉' : '✈'}
              </div>
              
              <div className="c2-aircraft-info">
                <div className="c2-aircraft-callsign">
                  {ac.callsign?.trim() || ac.icao24.toUpperCase()}
                </div>
                <div className="c2-aircraft-country">
                  {ac.originCountry}
                </div>
              </div>
              
              <div className="c2-aircraft-altitude">
                <div className="c2-aircraft-alt-value">
                  {ac.onGround ? 'GND' : formatAltitude(ac.baroAltitude)}
                </div>
                <div className="c2-aircraft-alt-label">
                  {ac.onGround ? '' : 'FT'}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

export default AircraftList;
