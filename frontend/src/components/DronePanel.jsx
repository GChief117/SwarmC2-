import React, { useState } from 'react';
import DroneFSMDisplay from './DroneFSMDisplay';
import DroneTelemetryGrid from './DroneTelemetryGrid';
import DroneEventLog from './DroneEventLog';
import DroneConfigPanel from './DroneConfigPanel';

const TABS = [
  { id: 'telemetry', label: 'GROUND' },
  { id: 'fsm', label: 'FSM' },
  { id: 'events', label: 'EVENTS' },
  { id: 'config', label: 'CONFIG' },
];

export default function DronePanel({ drones, selectedDroneId, onSelectDrone, droneEvents }) {
  const [activeTab, setActiveTab] = useState('telemetry');

  const selectedDrone = drones.find(d => d.droneId === selectedDroneId) || drones[0];

  return (
    <div className="drone-panel">
      <div className="drone-panel-header">
        <h2 className="drone-panel-title">DRONE OPS</h2>
        <div className="drone-selector">
          {drones.map(d => (
            <button
              key={d.droneId}
              className={`drone-selector-btn ${d.droneId === (selectedDrone?.droneId) ? 'active' : ''}`}
              onClick={() => onSelectDrone(d.droneId)}
            >
              <span className={`drone-status-dot ${d.fsmState === 'CRUISE' ? 'nominal' : d.fsmState === 'EVADE' ? 'warning' : d.fsmState === 'SAFE_MODE' ? 'critical' : 'info'}`} />
              {d.callsign}
            </button>
          ))}
        </div>
      </div>

      <div className="drone-tabs">
        {TABS.map(tab => (
          <button
            key={tab.id}
            className={`drone-tab ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="drone-tab-content">
        {selectedDrone && activeTab === 'telemetry' && (
          <DroneTelemetryGrid drone={selectedDrone} />
        )}
        {selectedDrone && activeTab === 'fsm' && (
          <DroneFSMDisplay drone={selectedDrone} />
        )}
        {activeTab === 'events' && (
          <DroneEventLog events={droneEvents} droneId={selectedDrone?.droneId} />
        )}
        {selectedDrone && activeTab === 'config' && (
          <DroneConfigPanel drone={selectedDrone} />
        )}
      </div>
    </div>
  );
}
