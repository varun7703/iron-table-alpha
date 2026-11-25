import React, { useEffect, useState } from 'react';
import {
  CAPITAL_CURRENCY_BONUS,
  CAPITAL_HUMANS_BONUS,
  CAPITAL_VP,
  MAX_STACKS_PER_SECTOR,
  moveRate,
  unitCost,
} from '../combat';
import { Icon } from './Icons';

type ManualTab = 'flow' | 'units' | 'combat' | 'economy';

const TABS: { id: ManualTab; num: string; label: string }[] = [
  { id: 'flow', num: '01', label: 'Game Flow' },
  { id: 'units', num: '02', label: 'Units & Orders' },
  { id: 'combat', num: '03', label: 'Battle & Fog' },
  { id: 'economy', num: '04', label: 'Supply & Economy' },
];

function initialTabForPhase(phase?: string): ManualTab {
  if (phase === 'plan') return 'units';
  if (phase === 'simulate') return 'combat';
  if (phase === 'gather') return 'economy';
  return 'flow';
}

const UNIT_SPECS = [
  {
    type: 'infantry' as const,
    name: 'Infantry',
    role: 'Frontline',
    hp: 10,
    attackLabel: '2 Direct',
    strongAgainst: '×1.5 vs Artillery',
    note: 'Affordable troops for holding borders and claiming sectors.',
  },
  {
    type: 'artillery' as const,
    name: 'Artillery',
    role: 'Ranged Support',
    hp: 14,
    attackLabel: '4 Ranged · 1 Melee',
    strongAgainst: '×1.5 vs Tanks · ×1.2 vs Inf',
    note: 'Bombards across 1 border after movement. Fragile in melee.',
  },
  {
    type: 'tank' as const,
    name: 'Tank',
    role: 'Heavy Armor',
    hp: 20,
    attackLabel: '6 Direct',
    strongAgainst: '×1.4 vs Inf · ×1.5 vs Art',
    note: 'High durability and shock power. Watch out for enemy artillery.',
  },
];

function MatchupTriangle() {
  return (
    <div className="manual-diagram-card">
      <div className="manual-card-header">
        <span className="eyebrow">COMBINED-ARMS COUNTERS</span>
        <span className="manual-hint">Deterministic combat — no RNG</span>
      </div>
      <svg
        className="matchup-svg"
        viewBox="0 0 560 212"
        role="img"
        aria-label="Combined arms matchup triangle: Tanks beat Infantry (1.4x), Infantry beat Artillery in melee (1.5x), Artillery bombards Tanks (1.5x) and Infantry (1.2x)"
      >
        <defs>
          <marker
            id="fm-arrow"
            viewBox="0 0 10 10"
            refX="8"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path d="M0 1L9 5L0 9z" fill="#d8b87b" />
          </marker>
        </defs>

        {/* Tank -> Infantry */}
        <path
          d="M224 58 Q144 82 126 132"
          fill="none"
          stroke="#d8b87b"
          strokeWidth="2.2"
          markerEnd="url(#fm-arrow)"
        />
        <rect x="98" y="72" width="68" height="22" rx="11" fill="#2b4031" stroke="#d8b87b88" />
        <text x="132" y="87" textAnchor="middle" className="svg-pill-text">
          ×1.4 ATK
        </text>

        {/* Infantry -> Artillery */}
        <path
          d="M178 168 Q280 198 382 168"
          fill="none"
          stroke="#d8b87b"
          strokeWidth="2.2"
          markerEnd="url(#fm-arrow)"
        />
        <rect x="234" y="176" width="92" height="22" rx="11" fill="#2b4031" stroke="#d8b87b88" />
        <text x="280" y="191" textAnchor="middle" className="svg-pill-text">
          ×1.5 MELEE
        </text>

        {/* Artillery -> Tank */}
        <path
          d="M434 132 Q416 82 336 58"
          fill="none"
          stroke="#d8b87b"
          strokeWidth="2.2"
          strokeDasharray="4 4"
          markerEnd="url(#fm-arrow)"
        />
        <rect x="378" y="72" width="96" height="22" rx="11" fill="#3b3123" stroke="#e4a66899" />
        <text x="426" y="87" textAnchor="middle" className="svg-pill-text warm">
          ×1.5 BOMBARD
        </text>

        {/* Center secondary notes */}
        <text x="280" y="106" textAnchor="middle" className="svg-center-note">
          Artillery: ×1.2 bombard vs Infantry
        </text>
        <text x="280" y="122" textAnchor="middle" className="svg-center-note">
          Tanks: ×1.5 melee vs Artillery
        </text>

        {/* Tank Node */}
        <g transform="translate(226 12)">
          <rect width="108" height="58" rx="6" fill="#172520" stroke="#d8b87b88" strokeWidth="1.4" />
          <g transform="translate(12 14)" style={{ color: '#e7cc92' }}>
            <Icon type="tank" size={28} />
          </g>
          <text x="48" y="30" className="svg-node-title">
            TANK
          </text>
          <text x="48" y="46" className="svg-node-sub">
            6 ATK · 20 HP
          </text>
        </g>

        {/* Infantry Node */}
        <g transform="translate(60 136)">
          <rect width="118" height="58" rx="6" fill="#172520" stroke="#9db58d88" strokeWidth="1.4" />
          <g transform="translate(10 14)" style={{ color: '#c6d9b3' }}>
            <Icon type="infantry" size={28} />
          </g>
          <text x="44" y="30" className="svg-node-title">
            INFANTRY
          </text>
          <text x="44" y="46" className="svg-node-sub">
            2 ATK · 10 HP
          </text>
        </g>

        {/* Artillery Node */}
        <g transform="translate(382 136)">
          <rect width="122" height="58" rx="6" fill="#172520" stroke="#e0a66b88" strokeWidth="1.4" />
          <g transform="translate(10 14)" style={{ color: '#ebba82' }}>
            <Icon type="artillery" size={28} />
          </g>
          <text x="44" y="30" className="svg-node-title">
            ARTILLERY
          </text>
          <text x="44" y="46" className="svg-node-sub">
            4 RNG · 14 HP
          </text>
        </g>
      </svg>
    </div>
  );
}

function FogPreview({ mode }: { mode: 'plan' | 'simulate' | 'gather' }) {
  // 7-cell mini cluster: center (index 0) + 6 surrounding sectors
  const cells = [
    { cx: 52, cy: 36, own: true, adj: false },
    { cx: 52, cy: 14, own: false, adj: true },
    { cx: 74, cy: 25, own: false, adj: true },
    { cx: 74, cy: 47, own: false, adj: true },
    { cx: 52, cy: 58, own: false, adj: true },
    { cx: 30, cy: 47, own: false, adj: true },
    { cx: 30, cy: 25, own: false, adj: true },
    { cx: 96, cy: 36, own: false, adj: false },
    { cx: 8, cy: 36, own: false, adj: false },
  ];

  return (
    <svg className="fog-mini-svg" viewBox="-2 0 108 72" aria-hidden="true">
      <defs>
        <pattern id={`mfog-${mode}`} width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(35)">
          <rect width="6" height="6" fill="#323d34" />
          <path d="M0 0V6" stroke="#485748" strokeWidth="1.2" />
        </pattern>
      </defs>
      {cells.map((c, idx) => {
        const lit =
          mode === 'simulate' ? true : mode === 'plan' ? c.own || c.adj : c.own;
        const fill = c.own
          ? '#d8b87b'
          : lit
            ? '#7c9470'
            : `url(#mfog-${mode})`;
        return (
          <circle
            key={idx}
            cx={c.cx}
            cy={c.cy}
            r={c.own ? 10 : 8.5}
            fill={fill}
            fillOpacity={c.own ? 0.92 : lit ? 0.65 : 0.95}
            stroke={c.own ? '#fff1c7' : lit ? '#d5d2b3' : '#566454'}
            strokeWidth={c.own ? 1.6 : 1}
          />
        );
      })}
    </svg>
  );
}

export function ManualModal({ phase, onClose }: { phase?: string; onClose: () => void }) {
  const [tab, setTab] = useState<ManualTab>(() => initialTabForPhase(phase));
  const currentTabIndex = TABS.findIndex(t => t.id === tab);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose();
      }
    }
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const suggestedTab = phase ? initialTabForPhase(phase) : null;

  return (
    <div
      className="modal-backdrop"
      onClick={e => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <section
        className="manual manual-visual"
        role="dialog"
        aria-modal="true"
        aria-labelledby="manual-title"
      >
        <header className="manual-header">
          <div className="manual-title-row">
            <div>
              <span className="eyebrow">COMMANDER’S FIELD MANUAL</span>
              <h2 id="manual-title">One map. Three phases.</h2>
            </div>
            <button autoFocus className="close" onClick={onClose} aria-label="Close field manual">
              ×
            </button>
          </div>

          <nav className="manual-tabs" role="tablist" aria-label="Field manual sections">
            {TABS.map(t => {
              const active = t.id === tab;
              const isCurrentPhase = suggestedTab === t.id && Boolean(phase && phase !== 'lobby');
              return (
                <button
                  key={t.id}
                  id={`manual-tab-${t.id}`}
                  type="button"
                  role="tab"
                  aria-selected={active}
                  aria-controls={`manual-panel-${t.id}`}
                  className={'manual-tab ' + (active ? 'active' : '')}
                  onClick={() => setTab(t.id)}
                >
                  <span className="manual-tab-num">{t.num}</span>
                  <span>{t.label}</span>
                  {isCurrentPhase && <i className="manual-tab-dot" title="Matches current phase" />}
                </button>
              );
            })}
          </nav>
        </header>

        <div
          id={`manual-panel-${tab}`}
          role="tabpanel"
          aria-labelledby={`manual-tab-${tab}`}
          className="manual-body"
        >
          {tab === 'flow' && (
            <div className="manual-section-stack">
              <div>
                <div className="manual-block-label">
                  <span>STEP 1 — OPENING SETUP</span>
                  <small>Played once before Round 1</small>
                </div>
                <div className="setup-flow-grid">
                  <div className="flow-card">
                    <span className="flow-badge">⚄ ROLL</span>
                    <strong>Initiative Dice</strong>
                    <p>Highest roll picks first. Ties automatically reroll.</p>
                  </div>
                  <span className="flow-arrow" aria-hidden="true">
                    →
                  </span>
                  <div className="flow-card">
                    <span className="flow-badge">⚑ DRAFT</span>
                    <strong>Claim 4 Sectors</strong>
                    <p>Repeating order (A→B→C→D). Each tile starts with <b>4 infantry</b>.</p>
                  </div>
                  <span className="flow-arrow" aria-hidden="true">
                    →
                  </span>
                  <div className="flow-card">
                    <span className="flow-badge">★ CAPITAL</span>
                    <strong>Secret HQ (45s)</strong>
                    <p>Choose 1 of your 4 tiles. Hidden from rivals until Round 1.</p>
                  </div>
                </div>
              </div>

              <div>
                <div className="manual-block-label">
                  <span>STEP 2 — THE 3-PHASE LOOP</span>
                  <small>Repeats up to 20 rounds</small>
                </div>
                <div className="loop-grid">
                  <div className="phase-card">
                    <div className="phase-card-top">
                      <span className="phase-num">01</span>
                      <strong>PLAN</strong>
                      <span className="phase-timer">90s</span>
                    </div>
                    <p>Issue 1 secret order per army: Move / Attack, Bombard, or Hold &amp; Defend.</p>
                    <div className="phase-card-tags">
                      <span className="mini-tag">Visibility: Own + Adjacent</span>
                      <span className="mini-tag accent">Lock in to advance early</span>
                    </div>
                  </div>

                  <div className="phase-card">
                    <div className="phase-card-top">
                      <span className="phase-num">02</span>
                      <strong>SIMULATE</strong>
                      <span className="phase-timer">≥12s</span>
                    </div>
                    <p>All orders execute simultaneously. Pause or step through frames to inspect fights.</p>
                    <div className="phase-card-tags">
                      <span className="mini-tag bright">Visibility: Full Map</span>
                      <span className="mini-tag accent">All lock in to skip</span>
                    </div>
                  </div>

                  <div className="phase-card">
                    <div className="phase-card-top">
                      <span className="phase-num">03</span>
                      <strong>GATHER</strong>
                      <span className="phase-timer">No timer</span>
                    </div>
                    <p>Income arrives automatically. Recruit, reinforce, split/merge stacks, and set rations.</p>
                    <div className="phase-card-tags">
                      <span className="mini-tag">Visibility: Own Only</span>
                      <span className="mini-tag accent">Unfed auto-feed on lock</span>
                    </div>
                  </div>
                </div>
              </div>

              <div className="victory-banner">
                <div className="victory-formula">
                  <span className="eyebrow">HOW TO WIN</span>
                  <div className="vp-equation">
                    <span className="vp-chip">Territories × 1</span>
                    <b>+</b>
                    <span className="vp-chip gold">Capitals Controlled × {CAPITAL_VP}</span>
                    <b>=</b>
                    <span className="vp-result">Victory Points</span>
                  </div>
                </div>
                <p>
                  Highest VP after <b>Round 20</b> wins (ties break on territories, then capitals). Holding all{' '}
                  <b>24 sectors</b> ends the match immediately.
                </p>
              </div>
            </div>
          )}

          {tab === 'units' && (
            <div className="manual-section-stack">
              <MatchupTriangle />

              <div className="unit-spec-grid">
                {UNIT_SPECS.map(u => {
                  const cost = unitCost(u.type, 1);
                  const rate = moveRate(u.type);
                  return (
                    <div className="unit-spec-card" key={u.type}>
                      <div className="unit-spec-head">
                        <div className="unit-icon-wrap">
                          <Icon type={u.type} size={28} />
                        </div>
                        <div>
                          <strong>{u.name}</strong>
                          <span className="unit-role-pill">{u.role}</span>
                        </div>
                      </div>

                      <div className="unit-stat-bar">
                        <span>
                          <small>HP</small>
                          <b>{u.hp}</b>
                        </span>
                        <span>
                          <small>ATTACK</small>
                          <b>{u.attackLabel}</b>
                        </span>
                        <span>
                          <small>HOSTILE MOVE</small>
                          <b>¤{rate}</b>
                        </span>
                      </div>

                      <div className="unit-cost-row">
                        <span className="cost-label">RECRUIT</span>
                        <span className="cost-chip">
                          <Icon type="humans" size={13} /> {cost.humans}
                        </span>
                        <span className="cost-chip">¤{cost.currency}</span>
                        {cost.iron > 0 && (
                          <span className="cost-chip">
                            <Icon type="iron" size={13} /> {cost.iron}
                          </span>
                        )}
                      </div>

                      <div className="unit-counter-line">{u.strongAgainst}</div>
                      <p>{u.note}</p>
                    </div>
                  );
                })}
              </div>

              <div>
                <div className="manual-block-label">
                  <span>ORDERS DURING PLAN</span>
                  <small>1 order per army each round</small>
                </div>
                <div className="orders-grid">
                  <div className="order-card">
                    <div className="order-visual">
                      <svg viewBox="0 0 80 24" aria-hidden="true">
                        <path d="M6 18 Q40 2 70 14" fill="none" stroke="#f9d189" strokeWidth="2.5" strokeDasharray="6 4" />
                        <path d="M66 9 L74 15 L65 18z" fill="#f9d189" />
                      </svg>
                      <strong>Move / Attack</strong>
                    </div>
                    <p>
                      Move 1 adjacent sector. <b>Free</b> into friendly tiles; attacking or claiming neutral tiles
                      costs <b>¤1 / ¤2 / ¤3</b> per infantry / artillery / tank.
                    </p>
                  </div>

                  <div className="order-card">
                    <div className="order-visual">
                      <svg viewBox="0 0 80 24" aria-hidden="true">
                        <path d="M6 18 Q40 0 70 14" fill="none" stroke="#e4a668" strokeWidth="2.5" strokeDasharray="2 5" />
                        <circle cx="72" cy="14" r="4" fill="none" stroke="#e4a668" strokeWidth="1.8" />
                      </svg>
                      <strong>Bombard (Artillery)</strong>
                    </div>
                    <p>
                      Default artillery order. Shells an adjacent sector <b>after movement</b>. Cannot capture
                      territory and is suppressed if overrun in melee.
                    </p>
                  </div>

                  <div className="order-card">
                    <div className="order-visual">
                      <span className="order-shield-icon">🛡</span>
                      <strong>Hold &amp; Defend</strong>
                    </div>
                    <p>
                      Entrench in place for <b>+20% direct combat</b> and <b>−25% bombardment damage taken</b>.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {tab === 'combat' && (
            <div className="manual-section-stack">
              <div>
                <div className="manual-block-label">
                  <span>SIMULATE RESOLUTION ORDER</span>
                  <small>Every round resolves in this exact sequence</small>
                </div>
                <div className="resolution-grid">
                  <div className="res-step">
                    <div className="res-step-head">
                      <span className="phase-num">1</span>
                      <strong>Border Clashes</strong>
                    </div>
                    <div className="res-symbol">⇄</div>
                    <p>Armies crossing the same border head-on collide immediately and hold their origin tiles.</p>
                  </div>

                  <div className="res-step">
                    <div className="res-step-head">
                      <span className="phase-num">2</span>
                      <strong>Movement</strong>
                    </div>
                    <div className="res-symbol">→</div>
                    <p>Unblocked armies enter target sectors. Overrunning enemy artillery suppresses its fire.</p>
                  </div>

                  <div className="res-step">
                    <div className="res-step-head">
                      <span className="phase-num">3</span>
                      <strong>Bombardment</strong>
                    </div>
                    <div className="res-symbol">⌒</div>
                    <p>Surviving artillery fires across borders onto enemies that just arrived or stayed put.</p>
                  </div>

                  <div className="res-step">
                    <div className="res-step-head">
                      <span className="phase-num">4</span>
                      <strong>Battles &amp; Retreats</strong>
                    </div>
                    <div className="res-symbol">⚔</div>
                    <p>Contested tiles fight simultaneously. Losers retreat to a friendly neighbor (or perish). Ties favor the defender.</p>
                  </div>
                </div>
              </div>

              <div className="defense-strip">
                <div className="defense-card">
                  <span className="defense-badge">🛡 HOLD &amp; DEFEND</span>
                  <strong>+20% direct combat · −25% bombardment taken</strong>
                  <p>Best when holding a bottleneck or bracing for enemy artillery.</p>
                </div>
                <div className="defense-card gold">
                  <span className="defense-badge">★ OWN CAPITAL HQ</span>
                  <strong>+15% direct defense (+38% if also entrenched)</strong>
                  <p>Applies in destination battles while defending your own capital.</p>
                </div>
              </div>

              <div>
                <div className="manual-block-label">
                  <span>FOG OF WAR BY PHASE</span>
                  <small>Static terrain yields (♧ Food / ⬡ Iron) are always visible</small>
                </div>
                <div className="fog-grid">
                  <div className="fog-card">
                    <FogPreview mode="plan" />
                    <div>
                      <strong>Plan — Own + Adjacent</strong>
                      <p>See your territories and scout neighboring enemy garrisons.</p>
                    </div>
                  </div>
                  <div className="fog-card">
                    <FogPreview mode="simulate" />
                    <div>
                      <strong>Simulate — Full Continent</strong>
                      <p>Entire map is revealed. Review battles anytime in the <b>Last Round</b> tab.</p>
                    </div>
                  </div>
                  <div className="fog-card">
                    <FogPreview mode="gather" />
                    <div>
                      <strong>Gather — Own Only</strong>
                      <p>Only your sectors are visible while recruiting and feeding.</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}

          {tab === 'economy' && (
            <div className="manual-section-stack">
              <div className="economy-top-grid">
                <div className="econ-card">
                  <span className="eyebrow">SECTOR INCOME (EVERY GATHER)</span>
                  <div className="income-chips">
                    <div className="income-chip">
                      <Icon type="humans" size={20} />
                      <span>
                        <b>+3</b>
                        <small>Manpower</small>
                      </span>
                    </div>
                    <div className="income-chip">
                      <Icon type="currency" size={20} />
                      <span>
                        <b>+¤8</b>
                        <small>Treasury</small>
                      </span>
                    </div>
                    <div className="income-chip">
                      <Icon type="food" size={20} />
                      <span>
                        <b>+6–12</b>
                        <small>Food or Iron</small>
                      </span>
                    </div>
                  </div>
                  <p className="econ-note">
                    Starting stock: <b>16 Manpower · 40 Food · 16 Iron · ¤60</b>. Newly captured sectors pay out in the
                    very next Gather.
                  </p>
                </div>

                <div className="econ-card capital-card">
                  <span className="eyebrow">★ CAPITAL HQ BONUS</span>
                  <div className="capital-bonus-list">
                    <span>
                      <b>+{CAPITAL_CURRENCY_BONUS} ¤</b> Treasury / round
                    </span>
                    <span>
                      <b>+{CAPITAL_HUMANS_BONUS}</b> Manpower / round
                    </span>
                    <span>
                      <b>+15%</b> Direct defense
                    </span>
                    <span>
                      <b>+{CAPITAL_VP} VP</b> Per controlled capital
                    </span>
                  </div>
                  <p className="econ-note">
                    Resource &amp; defense bonuses require holding <b>your own</b> capital. Capturing a rival’s capital
                    still awards <b>+{CAPITAL_VP} VP</b>.
                  </p>
                </div>
              </div>

              <div>
                <div className="manual-block-label">
                  <span>ARMY RATIONS</span>
                  <small>Unfed stacks auto-feed Normal when Gather advances</small>
                </div>
                <div className="ration-tier-grid">
                  <div className="ration-card starve">
                    <div className="ration-head">
                      <strong>Starve</strong>
                      <span>0 food</span>
                    </div>
                    <div className="ration-meter">
                      <i style={{ width: '30%' }} />
                    </div>
                    <b>30% attack</b>
                    <small>−25 HP (min 1)</small>
                  </div>

                  <div className="ration-card minimum">
                    <div className="ration-head">
                      <strong>Minimum</strong>
                      <span>0.5× food</span>
                    </div>
                    <div className="ration-meter">
                      <i style={{ width: '65%' }} />
                    </div>
                    <b>65% attack</b>
                    <small>Conserves food (rounded up)</small>
                  </div>

                  <div className="ration-card normal">
                    <div className="ration-head">
                      <strong>Normal</strong>
                      <span>1.0× food</span>
                    </div>
                    <div className="ration-meter">
                      <i style={{ width: '100%' }} />
                    </div>
                    <b>100% attack</b>
                    <small>Maintains current health</small>
                  </div>

                  <div className="ration-card extra">
                    <div className="ration-head">
                      <strong>Extra</strong>
                      <span>1.5× food</span>
                    </div>
                    <div className="ration-meter">
                      <i style={{ width: '100%' }} />
                    </div>
                    <b>100% attack</b>
                    <small>+20 HP heal (max 100)</small>
                  </div>
                </div>
              </div>

              <div className="stack-rules-strip">
                <div>
                  <strong>Max {MAX_STACKS_PER_SECTOR} stacks / sector</strong>
                  <p>Merge matching unfed stacks to free up slots.</p>
                </div>
                <div>
                  <strong>Reinforce</strong>
                  <p>Adds troops at 100% HP (same unit cost; blends stack health).</p>
                </div>
                <div>
                  <strong>Split (Gather only)</strong>
                  <p>Divide an unfed stack into two detachments before Plan.</p>
                </div>
              </div>
            </div>
          )}
        </div>

        <footer className="manual-footer">
          <span className="manual-footnote">5 min disconnect grace · In-memory alpha server</span>
          <div className="manual-nav-actions">
            {currentTabIndex > 0 && (
              <button
                type="button"
                className="secondary manual-nav-btn"
                onClick={() => setTab(TABS[currentTabIndex - 1].id)}
              >
                ← {TABS[currentTabIndex - 1].label}
              </button>
            )}
            {currentTabIndex < TABS.length - 1 ? (
              <button
                type="button"
                className="primary manual-nav-btn"
                onClick={() => setTab(TABS[currentTabIndex + 1].id)}
              >
                Next: {TABS[currentTabIndex + 1].label} →
              </button>
            ) : (
              <button type="button" className="primary manual-nav-btn" onClick={onClose}>
                Ready to command ✓
              </button>
            )}
          </div>
        </footer>
      </section>
    </div>
  );
}
