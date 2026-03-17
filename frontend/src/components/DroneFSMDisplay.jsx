import React from 'react';

// FSM from paper Appendix B:
// M = (S, I, f, s₀, A)
// S = {BOOT, CRUISE, EVADE, HOLD, SAFE_MODE}
// I = {proximity_alert, proximity_clear, power_drop, formation_signal}
// 5 states × 4 inputs = 20 defined pairs

const STATES = [
  { id: 'BOOT', label: 'BOOT', x: 200, y: 40 },
  { id: 'CRUISE', label: 'CRUISE', x: 200, y: 130 },
  { id: 'EVADE', label: 'EVADE', x: 60, y: 220 },
  { id: 'HOLD', label: 'HOLD', x: 340, y: 220 },
  { id: 'SAFE_MODE', label: 'SAFE MODE', x: 200, y: 310 },
];

// Transitions matching paper's Appendix B table exactly
const TRANSITIONS = [
  // From BOOT
  { from: 'BOOT', to: 'CRUISE', label: 'prox_clear' },
  { from: 'BOOT', to: 'SAFE_MODE', label: 'prox_alert' },
  // From CRUISE
  { from: 'CRUISE', to: 'EVADE', label: 'prox_alert' },
  { from: 'CRUISE', to: 'SAFE_MODE', label: 'power_drop' },
  // From EVADE
  { from: 'EVADE', to: 'HOLD', label: 'prox_clear' },
  { from: 'EVADE', to: 'SAFE_MODE', label: 'power_drop' },
  // From HOLD
  { from: 'HOLD', to: 'EVADE', label: 'prox_alert' },
  { from: 'HOLD', to: 'CRUISE', label: 'form_signal' },
  { from: 'HOLD', to: 'SAFE_MODE', label: 'power_drop' },
];

const STATE_COLORS = {
  BOOT: '#00d4ff',
  CRUISE: '#00ff88',
  EVADE: '#ff9500',
  HOLD: '#ffd000',
  SAFE_MODE: '#ff3b3b',
};

// Moore outputs g(state) — per Section 4.2
const MOORE_OUTPUTS = {
  BOOT: 'INITIALIZING',
  CRUISE: 'GREEN BEACON',
  EVADE: 'RED BEACON',
  HOLD: 'YELLOW BEACON',
  SAFE_MODE: 'EMERGENCY',
};

export default function DroneFSMDisplay({ drone }) {
  const currentState = drone?.fsmState || 'BOOT';
  const mooreOutput = drone?.mooreOutput || MOORE_OUTPUTS[currentState] || '';

  const getStatePath = (state) => {
    const w = 90, h = 36;
    const x = state.x - w / 2, y = state.y - h / 2;
    const r = 8;
    return `M${x + r},${y} h${w - 2 * r} a${r},${r} 0 0 1 ${r},${r} v${h - 2 * r} a${r},${r} 0 0 1 -${r},${r} h-${w - 2 * r} a${r},${r} 0 0 1 -${r},-${r} v-${h - 2 * r} a${r},${r} 0 0 1 ${r},-${r} z`;
  };

  const getEdgePath = (t) => {
    const from = STATES.find(s => s.id === t.from);
    const to = STATES.find(s => s.id === t.to);
    if (!from || !to) return '';
    const dx = to.x - from.x;
    const dy = to.y - from.y;
    const cx = (from.x + to.x) / 2 + dy * 0.2;
    const cy = (from.y + to.y) / 2 - dx * 0.2;
    return `M${from.x},${from.y} Q${cx},${cy} ${to.x},${to.y}`;
  };

  return (
    <div className="drone-fsm-display">
      <div className="drone-fsm-header">
        <span className="drone-fsm-label">STATE MACHINE (Appendix B)</span>
        <span className="drone-fsm-current" style={{ color: STATE_COLORS[currentState] }}>
          {currentState}
        </span>
      </div>

      {/* Moore output display */}
      <div className="drone-fsm-moore">
        <span className="drone-fsm-moore-label">Moore g(s):</span>
        <span className="drone-fsm-moore-value" style={{ color: STATE_COLORS[currentState] }}>
          {mooreOutput}
        </span>
      </div>

      <svg viewBox="0 0 400 370" className="drone-fsm-svg">
        <defs>
          <marker id="arrowhead" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto">
            <polygon points="0 0, 8 3, 0 6" fill="#8b949e" />
          </marker>
          {Object.entries(STATE_COLORS).map(([id, color]) => (
            <filter key={id} id={`glow-${id}`}>
              <feGaussianBlur stdDeviation="4" result="blur" />
              <feFlood floodColor={color} floodOpacity="0.6" />
              <feComposite in2="blur" operator="in" />
              <feMerge>
                <feMergeNode />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          ))}
        </defs>

        {/* Transition arrows */}
        {TRANSITIONS.map((t, i) => {
          const midX = (STATES.find(s => s.id === t.from)?.x + STATES.find(s => s.id === t.to)?.x) / 2;
          const midY = (STATES.find(s => s.id === t.from)?.y + STATES.find(s => s.id === t.to)?.y) / 2;
          return (
            <g key={i}>
              <path
                d={getEdgePath(t)}
                fill="none"
                stroke="#484f58"
                strokeWidth="1"
                markerEnd="url(#arrowhead)"
              />
              <text
                x={midX + (t.from === t.to ? 0 : 8)}
                y={midY - 4}
                fill="#6e7681"
                fontSize="6"
                fontFamily="'JetBrains Mono', monospace"
                textAnchor="middle"
              >
                {t.label}
              </text>
            </g>
          );
        })}

        {/* State nodes */}
        {STATES.map(state => {
          const isActive = state.id === currentState;
          const color = STATE_COLORS[state.id];
          return (
            <g key={state.id}>
              <path
                d={getStatePath(state)}
                fill={isActive ? color + '22' : '#0d1117'}
                stroke={isActive ? color : '#484f58'}
                strokeWidth={isActive ? 2 : 1}
                filter={isActive ? `url(#glow-${state.id})` : undefined}
              />
              <text
                x={state.x}
                y={state.y + 1}
                textAnchor="middle"
                dominantBaseline="middle"
                fill={isActive ? color : '#8b949e'}
                fontSize="10"
                fontFamily="'JetBrains Mono', monospace"
                fontWeight={isActive ? 'bold' : 'normal'}
              >
                {state.label}
              </text>
              {isActive && (
                <circle cx={state.x + 40} cy={state.y - 14} r="4" fill={color}>
                  <animate attributeName="opacity" values="1;0.3;1" dur="1.5s" repeatCount="indefinite" />
                </circle>
              )}
            </g>
          );
        })}

        {/* FSM metadata */}
        <text x="200" y="360" textAnchor="middle" fill="#484f58" fontSize="7" fontFamily="'JetBrains Mono', monospace">
          |S|=5  |I|=4  |S×I|=20 pairs  Thm 2: COMPLETE
        </text>
      </svg>

      {/* Transition table */}
      <div className="drone-fsm-table">
        <div className="drone-fsm-table-header">TRANSITION TABLE f(s,i)</div>
        <table className="drone-fsm-transition-table">
          <thead>
            <tr>
              <th></th>
              <th>prox_alert</th>
              <th>prox_clear</th>
              <th>pwr_drop</th>
              <th>form_sig</th>
            </tr>
          </thead>
          <tbody>
            {[
              ['BOOT', 'SAFE_MODE', 'CRUISE', 'SAFE_MODE', 'CRUISE'],
              ['CRUISE', 'EVADE', 'CRUISE', 'SAFE_MODE', 'CRUISE'],
              ['EVADE', 'EVADE', 'HOLD', 'SAFE_MODE', 'HOLD'],
              ['HOLD', 'EVADE', 'HOLD', 'SAFE_MODE', 'CRUISE'],
              ['SAFE_MODE', 'SAFE_MODE', 'SAFE_MODE', 'SAFE_MODE', 'SAFE_MODE'],
            ].map(([state, ...transitions]) => (
              <tr key={state} className={state === currentState ? 'active' : ''}>
                <td className="state-cell" style={{ color: STATE_COLORS[state] }}>{state}</td>
                {transitions.map((t, i) => (
                  <td key={i} className={t === currentState ? 'highlight' : ''}>{t}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
