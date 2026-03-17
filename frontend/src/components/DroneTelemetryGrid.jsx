import React from 'react';

function MetricCard({ label, value, unit, status }) {
  const statusClass = status || 'nominal';
  return (
    <div className={`drone-metric-card ${statusClass}`}>
      <div className="drone-metric-label">{label}</div>
      <div className="drone-metric-value">{value}</div>
      {unit && <div className="drone-metric-unit">{unit}</div>}
    </div>
  );
}

function getBatteryStatus(pct) {
  if (pct < 15) return 'critical';
  if (pct < 30) return 'warning';
  return 'nominal';
}

function getLinkStatus(state) {
  if (state === 'LOST') return 'critical';
  if (state === 'DEGRADED') return 'warning';
  return 'nominal';
}

function getSubsystemStatus(drone) {
  const subsystems = [
    {
      name: 'MissionControl',
      abbr: 'MC',
      status: drone.fsmState === 'SAFE_MODE' ? 'critical' : 'nominal',
      detail: drone.fsmState,
    },
    {
      name: 'GNC',
      abbr: 'GNC',
      status: drone.fsmState === 'BOOT' ? 'warning' : 'nominal',
      detail: drone.fsmState === 'BOOT' ? 'INIT' : 'ACTIVE',
    },
    {
      name: 'PowerMgmt',
      abbr: 'PWR',
      status: getBatteryStatus(drone.energy.batteryPercent),
      detail: `${drone.energy.batteryPercent.toFixed(0)}%`,
    },
    {
      name: 'Communications',
      abbr: 'COM',
      status: getLinkStatus(drone.linkState),
      detail: drone.linkState,
    },
    {
      name: 'ObjectDetection',
      abbr: 'OBJ',
      status: drone.threatLevel === 'NONE' ? 'nominal' : drone.threatLevel === 'HIGH' || drone.threatLevel === 'CRITICAL' ? 'critical' : 'warning',
      detail: drone.threatLevel,
    },
    {
      name: 'HealthMonitor',
      abbr: 'HLT',
      status: drone.fsmState === 'SAFE_MODE' ? 'warning' : 'nominal',
      detail: drone.fsmState === 'SAFE_MODE' ? 'ALERT' : 'OK',
    },
  ];
  return subsystems;
}

export default function DroneTelemetryGrid({ drone }) {
  if (!drone) return null;

  const speed = Math.sqrt(
    drone.velocity.vx ** 2 + drone.velocity.vy ** 2 + drone.velocity.vz ** 2
  ).toFixed(1);

  const subsystems = getSubsystemStatus(drone);

  // Per-cycle energy from paper's model
  const perCycleEnergy = drone.energy?.perCycleEnergy || 0;
  const budgetPerCycle = drone.energy?.budgetPerCycle || 0.045;
  const maxPerCycle = drone.energy?.maxPerCycle || 0.050;
  const invariantHolds = perCycleEnergy <= budgetPerCycle;

  return (
    <div className="drone-ground-view">
      {/* Metrics grid */}
      <div className="drone-telemetry-grid">
        <MetricCard
          label="BATTERY"
          value={drone.energy.batteryPercent.toFixed(1) + '%'}
          status={getBatteryStatus(drone.energy.batteryPercent)}
        />
        <MetricCard
          label="SOLAR (Ps)"
          value={drone.energy.solarInputWatts.toFixed(1)}
          unit="W"
        />
        <MetricCard
          label="ALTITUDE"
          value={drone.position.altitude.toFixed(0)}
          unit="m MSL"
        />
        <MetricCard
          label="HEADING"
          value={drone.heading.toFixed(0) + '\u00B0'}
        />
        <MetricCard
          label="SPEED"
          value={speed}
          unit="m/s"
        />
        <MetricCard
          label="POWER DRAW"
          value={drone.energy.powerDrawWatts.toFixed(1)}
          unit="W"
        />
        <MetricCard
          label="ENDURANCE"
          value={drone.energy.estimatedEndurance.toFixed(0)}
          unit="min"
          status={drone.energy.estimatedEndurance < 30 ? 'warning' : 'nominal'}
        />
        <MetricCard
          label="S-BAND LINK"
          value={drone.linkState}
          status={getLinkStatus(drone.linkState)}
        />
      </div>

      {/* F Prime subsystem status */}
      <div className="drone-subsystem-section">
        <div className="drone-subsystem-header">F PRIME COMPONENTS</div>
        <div className="drone-subsystem-grid">
          {subsystems.map((sub) => (
            <div key={sub.abbr} className={`drone-subsystem-item ${sub.status}`}>
              <span className="drone-subsystem-abbr">{sub.abbr}</span>
              <span className="drone-subsystem-detail">{sub.detail}</span>
              <span className={`drone-subsystem-dot ${sub.status}`} />
            </div>
          ))}
        </div>
      </div>

      {/* Sensor interfaces (Section 5, Tables 6-7) */}
      {drone.sensors && drone.sensors.length > 0 && (
        <div className="drone-subsystem-section">
          <div className="drone-subsystem-header">SENSOR INTERFACES</div>
          <div className="drone-sensor-list">
            {drone.sensors.map((sen, i) => (
              <div key={i} className={`drone-sensor-item ${sen.status === 'NOMINAL' ? 'nominal' : 'warning'}`}>
                <span className="drone-sensor-name">{sen.component}</span>
                <span className="drone-sensor-proto">{sen.protocol}</span>
                <span className={`drone-subsystem-dot ${sen.status === 'NOMINAL' ? 'nominal' : 'warning'}`} />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Communication links (Section 5.3, Table 8) */}
      {drone.commLinks && drone.commLinks.length > 0 && (
        <div className="drone-subsystem-section">
          <div className="drone-subsystem-header">COMM LINKS</div>
          <div className="drone-sensor-list">
            {drone.commLinks.map((cl, i) => (
              <div key={i} className={`drone-sensor-item ${cl.state === 'NOMINAL' || cl.state === 'ACTIVE' ? 'nominal' : cl.state === 'STANDBY' ? '' : 'warning'}`}>
                <span className="drone-sensor-name">{cl.name}</span>
                <span className="drone-sensor-proto">{cl.bandwidth}</span>
                <span className="drone-sensor-proto">{cl.state}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Scheduling rates (Section 6) */}
      {drone.schedulingHz && (
        <div className="drone-subsystem-section">
          <div className="drone-subsystem-header">CONTROL PIPELINE (RTOS)</div>
          <div className="drone-sensor-list">
            {Object.entries(drone.schedulingHz).map(([task, hz]) => (
              <div key={task} className="drone-sensor-item nominal">
                <span className="drone-sensor-name">{task}</span>
                <span className="drone-sensor-proto">{hz} Hz</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Watchdog timer (Section 6.3) */}
      {drone.watchdog && (
        <div className="drone-subsystem-section">
          <div className="drone-subsystem-header">WATCHDOG TIMER</div>
          <div className="drone-sensor-list">
            <div className={`drone-sensor-item ${drone.watchdog.triggered ? 'critical' : 'nominal'}`}>
              <span className="drone-sensor-name">Status</span>
              <span className="drone-sensor-proto">{drone.watchdog.triggered ? 'TRIGGERED' : 'OK'}</span>
              <span className={`drone-subsystem-dot ${drone.watchdog.triggered ? 'critical' : 'nominal'}`} />
            </div>
            <div className="drone-sensor-item nominal">
              <span className="drone-sensor-name">Timeout</span>
              <span className="drone-sensor-proto">{drone.watchdog.timeoutSec}s</span>
            </div>
          </div>
        </div>
      )}

      {/* Energy invariance display (Theorem 1, Appendix A) */}
      <div className="drone-invariance-bar">
        <div className="drone-invariance-label">
          THEOREM 1: ENERGY INVARIANCE
          <span className={`drone-invariance-status ${invariantHolds ? 'pass' : 'fail'}`}>
            {invariantHolds ? ' PASS' : ' FAIL'}
          </span>
        </div>

        {/* Per-cycle energy bar: E(C) vs B vs (Ps+Pb)·T */}
        <div className="drone-invariance-formula">
          E(C) = {perCycleEnergy.toFixed(4)} J &le; B = {budgetPerCycle.toFixed(4)} J &le; (Ps+Pb)&middot;T = {maxPerCycle.toFixed(4)} J
        </div>

        <div className="drone-invariance-track">
          <div
            className="drone-invariance-fill"
            style={{
              width: `${Math.min(100, (perCycleEnergy / maxPerCycle) * 100)}%`,
              backgroundColor: invariantHolds ? '#00ff88' : '#ff3b3b',
            }}
          />
          {/* Budget marker */}
          <div
            className="drone-invariance-marker"
            style={{
              left: `${(budgetPerCycle / maxPerCycle) * 100}%`,
            }}
          />
        </div>
        <div className="drone-invariance-values">
          <span>E(C)={perCycleEnergy.toFixed(4)}J</span>
          <span>B={budgetPerCycle.toFixed(3)}J</span>
          <span>(Ps+Pb)T={maxPerCycle.toFixed(3)}J</span>
        </div>
      </div>

      {/* Active tasks (Appendix A: εᵢ values) */}
      {drone.energy?.activeTasks && drone.energy.activeTasks.length > 0 && (
        <div className="drone-subsystem-section">
          <div className="drone-subsystem-header">ACTIVE TASKS E(C) = &Sigma;&epsilon;&#7522;</div>
          <div className="drone-sensor-list">
            {drone.energy.activeTasks.filter(t => t.active).map((task, i) => (
              <div key={i} className="drone-sensor-item nominal">
                <span className="drone-sensor-name">{task.name}</span>
                <span className="drone-sensor-proto">&epsilon;={task.epsilon.toFixed(4)} J</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
