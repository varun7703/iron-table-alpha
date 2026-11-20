import { clamp, type MotionFrame } from './battleMotion';
import { CAPITAL_CURRENCY_BONUS, CAPITAL_HUMANS_BONUS } from './combat';
import './game-animations.css';

type Tile = {
  id: number;
  x: number;
  y: number;
  owner: string;
  resource: string;
  yield: number;
  humans: number;
  currency: number;
};

function Burst({ x, y, t, reduced }: { x: number; y: number; t: number; reduced: boolean }) {
  const p = clamp((t - 0.62) / 0.35);
  if (t < 0.62 || t > 0.97) return null;
  return (
    <g className="combat-impact">
      <circle cx={x} cy={y} r={reduced ? 22 : 12 + 35 * p} fill="#ffe4a0" opacity={(1 - p) * 0.75} />
      {!reduced &&
        [0, 1, 2, 3].map(i => (
          <circle
            key={i}
            cx={x + Math.cos(i * 2.3) * p * 26}
            cy={y - p * 30 + Math.sin(i * 2.3) * 10}
            r={6 + 12 * p}
            fill="#414b3b"
            opacity={(1 - p) * 0.5}
          />
        ))}
    </g>
  );
}

export function BattleEffects({
  before,
  after,
  t,
  tiles,
  reduced,
  paused = false,
  orders,
}: {
  before: MotionFrame;
  after: MotionFrame;
  t: number;
  tiles: Tile[];
  reduced: boolean;
  paused?: boolean;
  orders: Record<string, { kind: string; target: number }>;
}) {
  const events = after.events.slice(before.events.length);
  const flight = clamp((t - 0.15) / 0.47);
  const destroyedSet = new Set(after.destroyed || []);
  const damaged = before.units.filter(u => {
    const next = after.units.find(v => v.id === u.id);
    if (!next) return destroyedSet.has(u.id);
    return next.health < u.health || next.count < u.count;
  });

  return (
    <g className="game-fx" pointerEvents="none" aria-hidden="true">
      {events.map((e, i) => {
        const a = tiles[e.from];
        const b = tiles[e.to];
        if (!a || !b || e.kind === 'move') return null;

        if (e.kind === 'bombard') {
          const cx = (a.x + b.x) / 2;
          const cy = Math.min(a.y, b.y) - 70;
          const inv = 1 - flight;
          const sx = inv * inv * a.x + 2 * inv * flight * cx + flight * flight * b.x;
          const sy = inv * inv * a.y + 2 * inv * flight * cy + flight * flight * b.y;
          return (
            <g key={i}>
              {t > 0.12 && t < 0.26 && (
                <circle cx={a.x} cy={a.y} r={reduced ? 9 : 15 * (1 - (t - 0.12) / 0.14)} fill="#fff0af" />
              )}
              {!reduced && t >= 0.15 && t < 0.62 && (
                <>
                  <path
                    d={`M${a.x} ${a.y}Q${cx} ${cy} ${b.x} ${b.y}`}
                    fill="none"
                    stroke="#f7dfa4"
                    strokeWidth="2"
                    strokeDasharray="3 7"
                  />
                  <circle cx={sx} cy={sy} r="4" fill="#fff4c6" stroke="#885c30" />
                </>
              )}
              <Burst x={b.x} y={b.y} t={t} reduced={reduced} />
            </g>
          );
        }

        if (e.kind === 'battle') {
          const fighters = before.units.filter(u =>
            e.from !== e.to
              ? ((u.territory === e.from && orders[u.id]?.target === e.to) ||
                  (u.territory === e.to && orders[u.id]?.target === e.from)) &&
                orders[u.id]?.kind === 'move'
              : u.territory === e.to,
          );
          return (
            <g key={i}>
              {fighters.map(u => {
                const origin = tiles[u.territory];
                const enemy = fighters.find(v => v.owner !== u.owner);
                if (!origin || !enemy) return null;
                const target = tiles[enemy.territory];
                if (!target) return null;
                const local = clamp((t - 0.22) / 0.4);
                const same = origin.id === target.id;
                const side = u.owner === fighters[0].owner ? -1 : 1;
                const x = origin.x + (same ? side * 24 : 0);
                const y = origin.y + 15;
                const tx = target.x + (same ? -side * 24 : 0);
                const ty = target.y + 15;
                return (
                  <g key={u.id}>
                    {t > 0.2 && t < 0.6 && !reduced && (
                      <>
                        <path
                          d={`M${x} ${y}L${tx} ${ty}`}
                          stroke="#ffdf88"
                          strokeWidth={u.type === 'tank' ? 3 : 1.5}
                          strokeDasharray={u.type === 'infantry' ? '3 9' : '7 14'}
                          strokeDashoffset={-local * 50}
                        />
                        <circle
                          cx={x}
                          cy={y}
                          r={u.type === 'tank' ? 9 : 5}
                          fill="#ffe5a1"
                          opacity={0.5 + Math.sin(t * 70) * 0.5}
                        />
                      </>
                    )}
                    <Burst x={x} y={y} t={t} reduced={reduced} />
                  </g>
                );
              })}
            </g>
          );
        }

        return null;
      })}

      {before.units.map(u => {
        const next = after.units.find(v => v.id === u.id);
        if (!next || u.territory === next.territory || !after.label.startsWith('Retreats')) return null;
        const a = tiles[u.territory];
        const b = tiles[next.territory];
        if (!a || !b) return null;
        return (
          <g key={u.id}>
            <path d={`M${a.x} ${a.y}L${b.x} ${b.y}`} stroke="#ffe3a2" strokeDasharray="5 4" strokeWidth="2" />
            <text x={(a.x + b.x) / 2} y={(a.y + b.y) / 2 - 8} textAnchor="middle" className="game-capture-label">
              RETREAT
            </text>
          </g>
        );
      })}

      {damaged.map((u, idx) => {
        const next = after.units.find(v => v.id === u.id);
        const pos = tiles[u.territory];
        if (!pos) return null;
        const lost = u.count - (next?.count || 0);
        const health = u.health - (next?.health || 0);
        const slot = damaged.slice(0, idx).filter(v => v.territory === u.territory).length;
        const labelOpacity = reduced || paused ? 1 : 1 - clamp((t - 0.8) / 0.2);
        const rise = reduced || paused ? 0 : clamp((t - 0.62) / 0.38) * 20;
        return (
          t >= 0.62 && (
            <g key={u.id} opacity={labelOpacity}>
              <text x={pos.x} y={pos.y - 25 - slot * 16 - rise} textAnchor="middle" className="game-damage-label">
                {next ? (lost ? `−${lost} troops · −${health.toFixed(1)}%` : `−${health.toFixed(1)}%`) : 'DESTROYED'}
              </text>
            </g>
          )
        );
      })}

      {after.owners.map((owner, id) => {
        if (!owner || owner === before.owners[id] || t < 0.62) return null;
        const tile = tiles[id];
        if (!tile) return null;
        const lift = reduced || paused ? 1 : clamp((t - 0.62) / 0.3);
        return (
          <g key={id} transform={`translate(${tile.x + 32} ${tile.y - 6})`}>
            <path d="M0 14V-30" stroke="#f5e8b6" strokeWidth="2" />
            <path d={`M1 ${14 - 44 * lift}l21 4v14l-21-4z`} fill="#d8b87b" stroke="#fff0c7" />
            <text x="0" y="35" textAnchor="middle" className="game-capture-label">
              CAPTURED
            </text>
          </g>
        );
      })}
    </g>
  );
}

export function GatherEffects({
  tiles,
  capital,
  player,
  elapsed,
  reduced,
}: {
  tiles: Tile[];
  capital: number;
  player: string;
  elapsed: number;
  reduced: boolean;
}) {
  if (elapsed > 5) return null;
  const owned = tiles.filter(t => t.owner === player);
  if (owned.length === 0) return null;
  const ownCapital = tiles[capital];
  const hasOwnCapital = Boolean(ownCapital && ownCapital.owner === player);
  const target = hasOwnCapital ? ownCapital! : owned[0];

  return (
    <g className="game-fx" pointerEvents="none" aria-hidden="true">
      <g transform={`translate(${target.x} ${target.y - 44})`}>
        <rect x="-56" y="-22" width="112" height={hasOwnCapital ? 32 : 24} rx="3" fill="#243e2b" stroke="#ead9a8" />
        <text textAnchor="middle" y={hasOwnCapital ? -8 : -6} className="stockpile-label">
          STOCKPILE
        </text>
        {hasOwnCapital && (
          <text textAnchor="middle" y="4" className="stockpile-hq-bonus">
            ★ HQ +{CAPITAL_CURRENCY_BONUS} ¤ · +{CAPITAL_HUMANS_BONUS} H
          </text>
        )}
      </g>
      {owned.map((tile, i) => {
        const delay = (i % 8) * 0.18;
        const t = clamp((elapsed - delay) / 2.5);
        const travel = t * t * (3 - 2 * t);
        if (elapsed < delay || t >= 1) return null;
        const x = reduced ? target.x : tile.x + (target.x - tile.x) * travel;
        const y = reduced ? target.y - 60 : tile.y + (target.y - 44 - tile.y) * travel - Math.sin(t * Math.PI) * 30;
        const cur = tile.currency + (hasOwnCapital && tile.id === capital ? CAPITAL_CURRENCY_BONUS : 0);
        const hum = tile.humans + (hasOwnCapital && tile.id === capital ? CAPITAL_HUMANS_BONUS : 0);
        return (
          <g key={tile.id} transform={`translate(${x} ${y})`} opacity={1 - clamp((t - 0.85) / 0.15)}>
            <rect x="-56" y="-28" width="112" height="45" rx="4" fill="#f0e5bb" stroke="#536f43" />
            <text textAnchor="middle" y="-11" className="resource-flight-label">
              +{cur} ¤ · +{hum} humans
            </text>
            <text textAnchor="middle" y="6" className="resource-flight-label">
              +{tile.yield} {tile.resource}
            </text>
          </g>
        );
      })}
    </g>
  );
}
