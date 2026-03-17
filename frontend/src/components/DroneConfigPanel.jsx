import React, { useState } from 'react';

const API_BASE = import.meta.env.DEV ? 'http://localhost:8080' : '';

export default function DroneConfigPanel({ drone }) {
  const [config, setConfig] = useState({
    droneId: drone.droneId,
    energyBudgetLimit: drone.energy.budgetLimit,
    safetyRadius: 100,
    maxSpeed: 25,
    maxAltitude: 400,
    criticalBatteryPct: 15,
    watchdogTimeoutSec: 10,
    downlinkRateHz: 1,
  });
  const [validation, setValidation] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState(null);

  const handleChange = (field, value) => {
    setConfig(prev => ({ ...prev, [field]: parseFloat(value) || 0 }));
    setValidation(null);
    setMessage(null);
  };

  const handleValidate = async () => {
    setSubmitting(true);
    try {
      const res = await fetch(`${API_BASE}/api/drones/validate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...config, droneId: drone.droneId }),
      });
      const data = await res.json();
      setValidation(data);
    } catch (err) {
      setMessage({ type: 'error', text: 'Validation request failed' });
    }
    setSubmitting(false);
  };

  const handleDeploy = async () => {
    setSubmitting(true);
    try {
      const res = await fetch(`${API_BASE}/api/drones/config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...config, droneId: drone.droneId }),
      });
      const data = await res.json();
      if (data.error) {
        setMessage({ type: 'error', text: data.error });
      } else {
        setMessage({ type: 'success', text: 'Configuration deployed successfully' });
        setValidation(data.validation);
      }
    } catch (err) {
      setMessage({ type: 'error', text: 'Deploy request failed' });
    }
    setSubmitting(false);
  };

  const allPass = validation && Array.isArray(validation) && validation.every(v => v.pass);

  const fields = [
    { key: 'energyBudgetLimit', label: 'Energy Budget (Wh)', min: 100, max: 2000 },
    { key: 'safetyRadius', label: 'Safety Radius (m)', min: 10, max: 500 },
    { key: 'maxSpeed', label: 'Max Speed (m/s)', min: 5, max: 50 },
    { key: 'maxAltitude', label: 'Max Altitude (m)', min: 50, max: 1000 },
    { key: 'criticalBatteryPct', label: 'Critical Battery (%)', min: 5, max: 50 },
    { key: 'watchdogTimeoutSec', label: 'Watchdog Timeout (s)', min: 5, max: 60 },
    { key: 'downlinkRateHz', label: 'Downlink Rate (Hz)', min: 1, max: 10 },
  ];

  return (
    <div className="drone-config-panel">
      <div className="drone-config-header">REMOTE CONFIG</div>

      <div className="drone-config-fields">
        {fields.map(f => (
          <div key={f.key} className="drone-config-field">
            <label>{f.label}</label>
            <input
              type="number"
              value={config[f.key]}
              min={f.min}
              max={f.max}
              onChange={e => handleChange(f.key, e.target.value)}
            />
          </div>
        ))}
      </div>

      <div className="drone-config-actions">
        <button className="drone-config-btn validate" onClick={handleValidate} disabled={submitting}>
          {submitting ? 'VALIDATING...' : 'VALIDATE'}
        </button>
        <button
          className="drone-config-btn deploy"
          onClick={handleDeploy}
          disabled={submitting || !allPass}
        >
          DEPLOY
        </button>
      </div>

      {message && (
        <div className={`drone-config-message ${message.type}`}>
          {message.text}
        </div>
      )}

      {validation && Array.isArray(validation) && (
        <div className="drone-validation-results">
          {validation.map((gate, i) => (
            <div key={i} className={`drone-validation-gate ${gate.pass ? 'pass' : 'fail'}`}>
              <div className="drone-validation-gate-header">
                <span className={`drone-validation-indicator ${gate.pass ? 'pass' : 'fail'}`}>
                  {gate.pass ? 'PASS' : 'FAIL'}
                </span>
                <span className="drone-validation-gate-name">{gate.gate.replace('_', ' ').toUpperCase()}</span>
              </div>
              <div className="drone-validation-summary">{gate.summary}</div>
              <div className="drone-validation-evidence">
                {gate.evidence?.map((e, j) => (
                  <div key={j} className={`drone-validation-proof ${e.pass ? 'pass' : 'fail'}`}>
                    <span className="drone-proof-status">{e.pass ? '✓' : '✗'}</span>
                    <span className="drone-proof-property">{e.property}</span>
                    <span className="drone-proof-actual">{e.actual}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
