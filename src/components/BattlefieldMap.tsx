import React from 'react';
import { BattleEffects, GatherEffects } from '../GameAnimations';
import type { BattleFrame, Game, GameEvent, Player, Territory, Unit } from '../types';
import { Icon } from './Icons';

const PHASE_INFO: Record<string, [string, string]> = {
  roll: ['Roll for initiative', 'Commanders roll one at a time. Tied results are rerolled.'],
  capital: [
    'Choose your capital',
    'Select one of your four territories (45s). Rival picks stay concealed until Round 1.',
  ],
  lobby: ['Assemble your commanders', 'Create a private room. Divide the continent. Write your own campaign.'],
  draft: ['Divide the continent', 'Repeating dice order. Claim one territory on your turn, up to four each.'],
  plan: ['Make your next move', 'Select an army, then an adjacent territory. Artillery defaults to Bombard.'],
  simulate: [
    'The battlefield is revealed',
    'Orders resolve together with the full map revealed. Pause or scrub any step.',
  ],
  gather: ['Rebuild. Replenish. Prepare.', 'Income collected. Lock in when ready. Recruit, split, merge, and allocate rations.'],
  finished: ['The campaign has ended', 'Victory Points (Territories + 2 × Capitals) decide the victor.'],
};

function badgeRowX(centerX: number, slot: number, totalSlots: number): number {
  const count = Math.min(4, Math.max(1, totalSlots));
  const stride = 26;
  const width = (count - 1) * stride + 24;
  return centerX - width / 2 + (slot % 4) * stride;
}

export function BattlefieldMap({
  game,
  me,
  player,
  phase,
  locked,
  selected,
  army,
  unit,
  mode,
  now,
  phaseStarted,
  reducedMotion,
  simulationProgress,
  frameIndex,
  frame,
  frameT,
  beforeFrame,
  displayUnits,
  displayTerritories,
  visibleEvents,
  frameEvents,
  currentEvent,
  scrubPaused,
  draftTurnPlayer,
  isMyDraftTurn,
  draftPickNumber,
  onTogglePause,
  onStepFrame,
  onSelectFrame,
  onToggleReducedMotion,
  onClickTerritory,
  onSelectArmy,
}: {
  game: Game;
  me?: Player;
  player: string;
  phase: string;
  locked: boolean;
  selected: number | null;
  army: string;
  unit?: Unit;
  mode: string;
  now: number;
  phaseStarted: number;
  reducedMotion: boolean;
  simulationProgress: number;
  frameIndex: number;
  frame?: BattleFrame;
  frameT: number;
  beforeFrame?: BattleFrame;
  displayUnits: Unit[];
  displayTerritories: Territory[];
  visibleEvents: GameEvent[];
  frameEvents: GameEvent[];
  currentEvent?: GameEvent;
  scrubPaused: boolean;
  draftTurnPlayer?: Player;
  isMyDraftTurn: boolean;
  draftPickNumber: number;
  onTogglePause: () => void;
  onStepFrame: (delta: number) => void;
  onSelectFrame: (idx: number) => void;
  onToggleReducedMotion: (v: boolean) => void;
  onClickTerritory: (t: Territory) => void;
  onSelectArmy: (tId: number, u: Unit) => void;
}) {
  const totalFrames = Math.max(1, game.timeline?.length || 1);
  // While paused at settled end-of-frame (frameT >= 1), render settled units inside their destination tile row.
  const settled = phase === 'simulate' && scrubPaused && frameT >= 1;
  const movingUnits = settled ? [] : displayUnits.filter(u => u.moving);

  // Precompute stable stationary slot indices per territory from beforeFrame during simulate
  // so surviving stacks never snap horizontally if another stack in the sector is destroyed.
  const slotMap = new Map<string, { slot: number; total: number }>();
  if (phase === 'simulate' && beforeFrame && !settled) {
    const byTile = new Map<number, string[]>();
    for (const u of beforeFrame.units) {
      const next = frame?.units.find(v => v.id === u.id);
      if (next && next.territory !== u.territory) continue;
      const list = byTile.get(u.territory) || [];
      list.push(u.id);
      byTile.set(u.territory, list);
    }
    for (const u of displayUnits) {
      if (u.moving) continue;
      const list = byTile.get(u.territory) || [];
      if (!list.includes(u.id)) {
        list.push(u.id);
        byTile.set(u.territory, list);
      }
    }
    for (const ids of byTile.values()) {
      ids.forEach((id, idx) => slotMap.set(id, { slot: idx, total: ids.length }));
    }
  }

  const renderBadge = (u: Unit, tx: number, ty: number) => {
    const entrenched =
      (phase === 'plan' || (phase === 'simulate' && !frame?.label.startsWith('Retreats'))) &&
      game.orders[u.id]?.kind === 'defend';
    const recoil =
      phase === 'simulate' &&
      frame?.label.startsWith('Bombardment') &&
      game.orders[u.id]?.kind === 'bombard' &&
      frameT > 0.12 &&
      frameT < 0.26 &&
      !reducedMotion
        ? 5 * (1 - (frameT - 0.12) / 0.14)
        : 0;
    const ownerColor = game.players.find(p => p.id === u.owner)?.color;
    const canSelectBadge = phase !== 'simulate' && u.owner === player;

    return (
      <g
        key={u.id}
        transform={`translate(${tx - recoil},${ty})`}
        className={
          'army-marker ' +
          (army === u.id && phase !== 'simulate' ? 'chosen ' : '') +
          (entrenched ? 'entrenched' : '')
        }
        onClick={e => {
          if (canSelectBadge) {
            e.stopPropagation();
            onSelectArmy(u.territory, u);
          }
        }}
      >
        <rect width="24" height="24" rx="4" fill={ownerColor} />
        <g transform="translate(2 1)">
          <Icon type={u.type} size={18} />
        </g>
        {entrenched && (
          <path
            className="entrench-shield"
            d="M20 -2l3.5 1.2v3c0 2.2-1.5 3.8-3.5 4.5-2-.7-3.5-2.3-3.5-4.5v-3z"
          />
        )}
        <text x="22" y="24">
          {u.count}
        </text>
        <rect className="unit-health-track" x="0" y="29" width="24" height="3" />
        <rect className="unit-health-fill" x="0" y="29" width={(24 * u.health) / 100} height="3" />
      </g>
    );
  };

  return (
    <main className="battlefield">
      <div className="map-top">
        <div>
          <span className="eyebrow">THEATER OF OPERATIONS</span>
          <h2>The divided continent</h2>
        </div>
        {phase === 'draft' && draftTurnPlayer ? (
          <span
            className={'draft-map-badge ' + (isMyDraftTurn ? 'my-turn' : '')}
            style={{ '--turn-color': draftTurnPlayer.color } as React.CSSProperties}
          >
            <i />
            {isMyDraftTurn
              ? `YOUR PICK (${draftPickNumber}/4) — CLICK AN UNCLAIMED SECTOR`
              : `WAITING FOR ${draftTurnPlayer.name.toUpperCase()} (${draftPickNumber}/4)`}
          </span>
        ) : (
          <span className="map-badge">24 SECTORS / ALPHA MAP</span>
        )}
      </div>

      <div className="map-wrap">
        {phase === 'simulate' && (
          <div className="simulation-status" role="status" aria-live="polite">
            <div className="simulation-heading">
              <span className={'spinner ' + (scrubPaused ? 'paused' : '')} aria-hidden="true" />
              <strong>{scrubPaused ? 'Battle playback (paused)' : 'Battle playback'}</strong>
              <span>{Math.round(simulationProgress * 100)}%</span>
            </div>
            <div
              className="simulation-progress"
              role="progressbar"
              aria-label="Battle playback progress"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(simulationProgress * 100)}
            >
              <i style={{ width: `${simulationProgress * 100}%` }} />
            </div>
            <p>
              {frame?.label
                ? frameT < 0.62
                  ? frame.label.replace(' resolved', '').replace('complete', 'in progress')
                  : frame.label
                : 'Preparing battlefield playback…'}
            </p>
            <div className="scrubber-row">
              <div className="scrubber-controls" role="group" aria-label="Simulation step controls">
                <button
                  type="button"
                  className="scrubber-btn"
                  disabled={frameIndex <= 0}
                  onClick={() => onStepFrame(-1)}
                  aria-label="Previous step"
                >
                  ◀
                </button>
                <button type="button" className="scrubber-btn primary-scrub" onClick={onTogglePause}>
                  {scrubPaused ? '▶ Play' : '❚❚ Pause'}
                </button>
                <button
                  type="button"
                  className="scrubber-btn"
                  disabled={frameIndex >= totalFrames - 1}
                  onClick={() => onStepFrame(1)}
                  aria-label="Next step"
                >
                  ▶
                </button>
              </div>
              <div className="scrubber-steps">
                {Array.from({ length: totalFrames }, (_, idx) => (
                  <button
                    key={idx}
                    type="button"
                    className={'step-pill ' + (idx === frameIndex ? 'active' : '')}
                    onClick={() => onSelectFrame(idx)}
                    title={game.timeline?.[idx]?.label || `Step ${idx + 1}`}
                  >
                    {idx + 1}
                  </button>
                ))}
              </div>
            </div>
            <div className="scrubber-meta">
              <small>
                Step {frameIndex + 1}/{totalFrames} · {visibleEvents.length}/{game.events.length} dispatches
              </small>
              <label className="battle-motion-settings">
                <input
                  type="checkbox"
                  checked={reducedMotion}
                  onChange={e => onToggleReducedMotion(e.target.checked)}
                />
                Reduced motion
              </label>
            </div>
          </div>
        )}

        <svg
          className={'map ' + (phase === 'simulate' ? 'simulating' : '') + (reducedMotion ? ' motion-reduced' : '')}
          viewBox="0 0 880 640"
          role="img"
          aria-label="Interactive territory map"
        >
          <defs>
            <pattern id="grid" width="24" height="24" patternUnits="userSpaceOnUse">
              <path d="M24 0H0V24" fill="none" stroke="#a9ab95" strokeWidth=".4" opacity=".3" />
            </pattern>
            <pattern id="fog" width="8" height="8" patternUnits="userSpaceOnUse" patternTransform="rotate(35)">
              <rect width="8" height="8" fill="#444d44" />
              <path d="M0 0V8" stroke="#536052" strokeWidth="1" />
            </pattern>
            <marker
              id="arrow"
              viewBox="0 0 10 10"
              refX="8"
              refY="5"
              markerWidth="6"
              markerHeight="6"
              orient="auto-start-reverse"
            >
              <path d="M0 0L10 5L0 10z" fill="#f9d189" />
            </marker>
          </defs>

          <rect width="880" height="640" fill="url(#grid)" />
          <text x="25" y="325" className="sea-label" transform="rotate(-90 25 325)">
            WESTERN REACH
          </text>
          <text x="840" y="260" className="sea-label" transform="rotate(90 840 260)">
            THE PALE SEA
          </text>

          {displayTerritories.map(t => {
            const owner = game.players.find(p => p.id === t.owner);
            const visible = t.visible;
            const targetable =
              phase === 'plan' && !locked && Boolean(unit) && game.territories[unit!.territory]?.neighbors.includes(t.id);
            const claimable = isMyDraftTurn && !t.owner;
            const stationaryHere = displayUnits.filter(u => (settled || !u.moving) && u.territory === t.id);
            const hasCapital = Boolean(t.isCapital) || game.players.some(p => p.capital === t.id);

            return (
              <g
                key={t.id}
                role="button"
                tabIndex={0}
                aria-label={`${t.name}, ${t.resource} ${t.yield}${visible ? '' : ', fog of war'}`}
                onClick={() => onClickTerritory(t)}
                onKeyDown={e => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    onClickTerritory(t);
                  }
                }}
                className={
                  'territory ' +
                  (!visible ? 'fogged ' : '') +
                  (selected === t.id ? 'selected ' : '') +
                  (targetable ? 'targetable ' : '') +
                  (claimable ? 'claimable' : '')
                }
              >
                <polygon
                  points={t.points}
                  fill={visible ? owner?.color || '#aaa98a' : 'url(#fog)'}
                  fillOpacity={visible ? 0.58 : 0.92}
                />
                {hasCapital && visible && (
                  <text x={t.x} y={t.y - 36} className="capital-marker">
                    ★ CAPITAL
                  </text>
                )}
                <text x={t.x} y={t.y - 19} className="territory-name">
                  {t.name}
                </text>
                <text x={t.x} y={t.y + 1} className={'production ' + (!visible ? 'fogged-yield' : '')}>
                  {t.resource === 'food' ? '♧' : '⬡'} {t.yield}
                </text>
                {stationaryHere.map((u, i) => {
                  const stable = slotMap.get(u.id);
                  const slot = stable ? stable.slot : i;
                  const count = stable ? stable.total : stationaryHere.length;
                  const cx = settled ? t.x : (u.motionX ?? t.x);
                  const cy = settled ? t.y : (u.motionY ?? t.y);
                  return renderBadge(u, badgeRowX(cx, slot, count), cy + 10);
                })}
              </g>
            );
          })}

          {movingUnits.length > 0 && (
            <g className="moving-armies">
              {movingUnits.map((u, idx) => {
                const tile = game.territories[u.territory];
                if (!tile) return null;
                const dest = u.destTerritory ?? u.territory;
                const sameDestGroup = movingUnits.filter(v => (v.destTerritory ?? v.territory) === dest);
                const groupSlot = sameDestGroup.findIndex(v => v.id === u.id);
                const slot = groupSlot >= 0 ? groupSlot : idx;
                return renderBadge(
                  u,
                  badgeRowX(u.motionX ?? tile.x, slot, sameDestGroup.length),
                  (u.motionY ?? tile.y) - 6,
                );
              })}
            </g>
          )}

          {(phase === 'plan' || (phase === 'simulate' && frameIndex === 0)) &&
            Object.values(game.orders).map(o => {
              const u = displayUnits.find(v => v.id === o.unit);
              if (!u || o.kind === 'defend' || u.territory === o.target) return null;
              const a = game.territories[u.territory];
              const b = game.territories[o.target];
              if (!a || !b) return null;
              return (
                <path
                  className="order-line"
                  key={o.unit}
                  d={`M${a.x} ${a.y} Q${(a.x + b.x) / 2 + 15} ${(a.y + b.y) / 2 - 25} ${b.x} ${b.y}`}
                  markerEnd="url(#arrow)"
                  strokeDasharray={o.kind === 'bombard' ? '3 8' : '8 5'}
                />
              );
            })}

          {phase === 'simulate' &&
            frameEvents
              .filter(e => e.from !== e.to)
              .map((e, i) => {
                const a = game.territories[e.from];
                const b = game.territories[e.to];
                if (!a || !b) return null;
                return (
                  <path key={i} className="event-line" d={`M${a.x} ${a.y}L${b.x} ${b.y}`} markerEnd="url(#arrow)" />
                );
              })}

          {phase === 'simulate' && currentEvent && game.territories[currentEvent.to] && (
            <circle
              className="battle-focus"
              cx={game.territories[currentEvent.to].x}
              cy={game.territories[currentEvent.to].y}
              r="35"
            />
          )}

          {phase === 'simulate' && beforeFrame && frame && (
            <BattleEffects
              orders={game.orders}
              before={beforeFrame}
              after={frame}
              t={frameT}
              tiles={game.territories}
              reduced={reducedMotion}
              paused={scrubPaused}
            />
          )}

          {phase === 'gather' && me && (
            <GatherEffects
              tiles={game.territories}
              capital={me.capital}
              player={player}
              elapsed={(now - phaseStarted) / 1000}
              reduced={reducedMotion}
            />
          )}

          <g transform="translate(770 570)" className="compass">
            <path d="M0-30L8 0 0 28-8 0zM-24 0H24" />
            <text y="-38">N</text>
          </g>
          <text x="85" y="595" className="map-caption">
            IRON TABLE CARTOGRAPHIC OFFICE — SECTOR SURVEY 01
          </text>
        </svg>
      </div>

      <div className="map-footer">
        <span>
          <i style={{ background: me?.color }} /> Your territories
        </span>
        <span>
          <i className="fog-key" /> Fog of war (yields visible)
        </span>
        <span className="map-tip">
          {unit && phase === 'plan' && !locked
            ? mode === 'bombard'
              ? `Select an adjacent target to bombard with ${unit.type}`
              : `Select an adjacent target to move ${unit.type}`
            : 'Click a territory to inspect'}
        </span>
      </div>

      <div className="phase-banner">
        <span className="phase-symbol">{phase === 'simulate' ? '⚔' : phase === 'gather' ? '♧' : '⌖'}</span>
        <div>
          <h3>{PHASE_INFO[phase]?.[0]}</h3>
          <p>{PHASE_INFO[phase]?.[1]}</p>
        </div>
        {phase === 'capital' && (
          <strong>
            {game.players.filter(p => p.capital >= 0).length}/{game.players.length} capitals chosen
          </strong>
        )}
        {phase === 'draft' && <strong>{game.players.find(p => p.id === game.turn)?.name}’s pick</strong>}
      </div>
    </main>
  );
}
