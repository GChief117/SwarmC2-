import React, { useState } from 'react';

const THREAT_COLORS = {
  CRITICAL: '#ff3b3b',
  HIGH: '#ff9500',
  MEDIUM: '#ffd000',
  LOW: '#00d4ff',
  NOMINAL: '#00ff88',
  UNKNOWN: '#8b949e',
};

const THREAT_BACKGROUNDS = {
  CRITICAL: 'rgba(255, 59, 59, 0.15)',
  HIGH: 'rgba(255, 149, 0, 0.15)',
  MEDIUM: 'rgba(255, 208, 0, 0.15)',
  LOW: 'rgba(0, 212, 255, 0.15)',
  NOMINAL: 'rgba(0, 255, 136, 0.15)',
  UNKNOWN: 'rgba(139, 148, 158, 0.15)',
};

function AIAnalysisPanel({ analysis, onRefresh, isLoading, onSelectAircraft, aircraft }) {
  const [expanded, setExpanded] = useState(true);

  if (!analysis) {
    return (
      <div className="c2-ai-panel">
        <div className="c2-ai-header">
          <div className="c2-ai-title">
            <span className="c2-ai-icon">⬡</span>
            <span>SENTINEL AI</span>
          </div>
        </div>
        <div className="c2-ai-loading">
          <div className="c2-ai-spinner" />
          <span>INITIALIZING TACTICAL AI...</span>
        </div>
      </div>
    );
  }

  const threatLevel = analysis.overall_threat_level || 'UNKNOWN';
  const threatColor = THREAT_COLORS[threatLevel] || THREAT_COLORS.UNKNOWN;
  const threatBg = THREAT_BACKGROUNDS[threatLevel] || THREAT_BACKGROUNDS.UNKNOWN;

  // Find aircraft by callsign for click handling
  const findAircraftByCallsign = (callsign) => {
    return aircraft?.find(ac => 
      ac.callsign?.trim().toUpperCase() === callsign?.trim().toUpperCase() ||
      ac.icao24?.toUpperCase() === callsign?.toUpperCase()
    );
  };

  return (
    <div className="c2-ai-panel">
      {/* Header */}
      <div className="c2-ai-header">
        <div className="c2-ai-title">
          <span className="c2-ai-icon">⬡</span>
          <span>SENTINEL AI</span>
          <span className="c2-ai-status">ACTIVE</span>
        </div>
        <div className="c2-ai-controls">
          <button 
            className="c2-ai-refresh" 
            onClick={onRefresh}
            disabled={isLoading}
          >
            {isLoading ? '◌' : '↻'}
          </button>
          <button 
            className="c2-ai-toggle"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? '▼' : '▲'}
          </button>
        </div>
      </div>

      {expanded && (
        <>
          {/* Threat Level Display */}
          <div 
            className="c2-ai-threat-display"
            style={{ background: threatBg, borderColor: threatColor }}
          >
            <div className="c2-ai-threat-header">
              <span className="c2-ai-threat-label">THREAT ASSESSMENT</span>
              <span 
                className="c2-ai-threat-level"
                style={{ color: threatColor, textShadow: `0 0 20px ${threatColor}` }}
              >
                {threatLevel}
              </span>
            </div>
            <div className="c2-ai-threat-score">
              <div className="c2-ai-score-bar">
                <div 
                  className="c2-ai-score-fill"
                  style={{ 
                    width: `${analysis.threat_score || 0}%`,
                    background: `linear-gradient(90deg, ${THREAT_COLORS.NOMINAL}, ${threatColor})`
                  }}
                />
              </div>
              <span className="c2-ai-score-value">{analysis.threat_score || 0}/100</span>
            </div>
          </div>

          {/* Summary */}
          <div className="c2-ai-summary">
            <div className="c2-ai-section-label">SITUATION SUMMARY</div>
            <p className="c2-ai-summary-text">{analysis.summary || 'No summary available'}</p>
          </div>

          {/* Key Observations */}
          {analysis.key_observations && analysis.key_observations.length > 0 && (
            <div className="c2-ai-observations">
              <div className="c2-ai-section-label">KEY OBSERVATIONS</div>
              {analysis.key_observations.slice(0, 3).map((obs, idx) => (
                <div 
                  key={idx} 
                  className="c2-ai-observation-item"
                  style={{ 
                    borderLeftColor: THREAT_COLORS[obs.threat_contribution] || THREAT_COLORS.LOW 
                  }}
                >
                  <div className="c2-ai-obs-type">{obs.type}</div>
                  <div className="c2-ai-obs-desc">{obs.description}</div>
                  {obs.aircraft_involved && obs.aircraft_involved.length > 0 && (
                    <div className="c2-ai-obs-aircraft">
                      {obs.aircraft_involved.map((callsign, i) => (
                        <button
                          key={i}
                          className="c2-ai-aircraft-tag"
                          onClick={() => {
                            const ac = findAircraftByCallsign(callsign);
                            if (ac && onSelectAircraft) onSelectAircraft(ac);
                          }}
                        >
                          {callsign}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Aircraft of Interest */}
          {analysis.aircraft_of_interest && analysis.aircraft_of_interest.length > 0 && (
            <div className="c2-ai-aoi">
              <div className="c2-ai-section-label">AIRCRAFT OF INTEREST</div>
              {analysis.aircraft_of_interest.slice(0, 4).map((aoi, idx) => (
                <div 
                  key={idx}
                  className="c2-ai-aoi-item"
                  onClick={() => {
                    const ac = findAircraftByCallsign(aoi.callsign || aoi.icao24);
                    if (ac && onSelectAircraft) onSelectAircraft(ac);
                  }}
                >
                  <div className="c2-ai-aoi-header">
                    <span 
                      className="c2-ai-aoi-callsign"
                      style={{ color: THREAT_COLORS[aoi.threat_level] || THREAT_COLORS.LOW }}
                    >
                      {aoi.callsign || aoi.icao24}
                    </span>
                    <span 
                      className="c2-ai-aoi-level"
                      style={{ 
                        background: THREAT_BACKGROUNDS[aoi.threat_level],
                        color: THREAT_COLORS[aoi.threat_level]
                      }}
                    >
                      {aoi.threat_level}
                    </span>
                  </div>
                  <div className="c2-ai-aoi-reason">{aoi.reason}</div>
                  <div className="c2-ai-aoi-action">
                    <span className="c2-ai-action-label">ACTION:</span>
                    <span className="c2-ai-action-value">{aoi.recommended_action}</span>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Tactical Recommendations */}
          {analysis.tactical_recommendations && analysis.tactical_recommendations.length > 0 && (
            <div className="c2-ai-recommendations">
              <div className="c2-ai-section-label">TACTICAL RECOMMENDATIONS</div>
              {analysis.tactical_recommendations.slice(0, 3).map((rec, idx) => (
                <div key={idx} className="c2-ai-rec-item">
                  <span className="c2-ai-rec-priority">P{rec.priority}</span>
                  <div className="c2-ai-rec-content">
                    <div className="c2-ai-rec-action">{rec.action}</div>
                    <div className="c2-ai-rec-rationale">{rec.rationale}</div>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Pattern Analysis Stats */}
          {analysis.pattern_analysis && (
            <div className="c2-ai-patterns">
              <div className="c2-ai-section-label">PATTERN ANALYSIS</div>
              <div className="c2-ai-pattern-grid">
                <div className="c2-ai-pattern-stat">
                  <span className="c2-ai-pattern-value">{analysis.pattern_analysis.formations_detected || 0}</span>
                  <span className="c2-ai-pattern-label">FORMATIONS</span>
                </div>
                <div className="c2-ai-pattern-stat">
                  <span className="c2-ai-pattern-value">{analysis.pattern_analysis.unusual_behaviors || 0}</span>
                  <span className="c2-ai-pattern-label">ANOMALIES</span>
                </div>
                <div className="c2-ai-pattern-stat">
                  <span className="c2-ai-pattern-value">{analysis.pattern_analysis.potential_threats || 0}</span>
                  <span className="c2-ai-pattern-label">THREATS</span>
                </div>
                <div className="c2-ai-pattern-stat">
                  <span className="c2-ai-pattern-value">{analysis.pattern_analysis.commercial_density || 'N/A'}</span>
                  <span className="c2-ai-pattern-label">DENSITY</span>
                </div>
              </div>
            </div>
          )}

          {/* Footer */}
          <div className="c2-ai-footer">
            <span className="c2-ai-timestamp">
              Updated: {analysis.timestamp ? new Date(analysis.timestamp).toLocaleTimeString() : '--:--:--'}
            </span>
            <span className="c2-ai-priority">
              Next: {analysis.next_update_priority || 'NORMAL'}
            </span>
          </div>
        </>
      )}
    </div>
  );
}

export default AIAnalysisPanel;
