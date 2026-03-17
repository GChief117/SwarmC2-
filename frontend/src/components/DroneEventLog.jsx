import React from 'react';

const SEVERITY_CLASS = {
  info: 'event-info',
  warning: 'event-warning',
  critical: 'event-critical',
};

const CATEGORY_ICONS = {
  fsm: 'FSM',
  power: 'PWR',
  proximity: 'PRX',
  comms: 'COM',
  health: 'HLT',
};

export default function DroneEventLog({ events, droneId }) {
  const filtered = (events || [])
    .filter(e => !droneId || e.droneId === droneId)
    .slice(0, 50);

  return (
    <div className="drone-event-log">
      <div className="drone-event-log-header">
        <span>EVENT LOG</span>
        <span className="drone-event-count">{filtered.length} events</span>
      </div>
      <div className="drone-event-list">
        {filtered.length === 0 ? (
          <div className="drone-event-empty">No events recorded</div>
        ) : (
          filtered.map((evt, i) => (
            <div key={i} className={`drone-event-item ${SEVERITY_CLASS[evt.severity] || ''}`}>
              <div className="drone-event-meta">
                <span className={`drone-event-badge ${evt.severity}`}>
                  {CATEGORY_ICONS[evt.category] || evt.category?.toUpperCase()}
                </span>
                <span className="drone-event-time">
                  {new Date(evt.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div className="drone-event-message">{evt.message}</div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
