import React from 'react';
import {
  CAPITAL_CURRENCY_BONUS,
  CAPITAL_HUMANS_BONUS,
  RESOURCE_KEYS,
  RESOURCE_LABELS,
  type EconomySummary,
} from '../combat';
import type { Game, Player } from '../types';
import { Icon } from './Icons';

export function LeftPanel({
  game,
  me,
  player,
  phase,
  now,
  full,
  econ,
  onAddBot,
  onStart,
}: {
  game: Game;
  me?: Player;
  player: string;
  phase: string;
  now: number;
  full: boolean;
  econ: EconomySummary;
  onAddBot: () => void;
  onStart: () => void;
}) {
  const inCampaign = ['plan', 'simulate', 'gather'].includes(phase);
  // During simulate, derive self VP from display frame (econ.vp) so 0% playback never spoils outcomes.
  const myVP = phase === 'simulate' ? econ.vp : (me?.vp ?? econ.vp);

  return (
    <aside className="left-panel">
      <div className="panel-heading">
        YOUR COMMAND <span>⌖</span>
      </div>

      <div className="commander">
        <span style={{ background: me?.color }} className="avatar">
          {me?.name.slice(0, 1)}
        </span>
        <div>
          <strong>{me?.name}</strong>
          <small>
            {econ.territoriesHeld} TERRITORIES · {myVP} VP
          </small>
        </div>
      </div>

      <div className="resource-grid">
        {RESOURCE_KEYS.map(r => {
          const inc = econ.income[r];
          const isFood = r === 'food';
          const deltaClass = isFood ? (econ.foodNet < 0 ? 'delta-neg' : 'delta-pos') : 'delta-pos';
          return (
            <div className="resource" key={r}>
              <Icon type={r} />
              <span>
                {RESOURCE_LABELS[r]}
                <strong>{me?.resources[r] || 0}</strong>
                {inCampaign && (
                  <em className={'resource-delta ' + deltaClass}>
                    {isFood
                      ? `+${inc} / −${econ.foodUpkeep} (${econ.foodNet >= 0 ? '+' : ''}${econ.foodNet})`
                      : `+${inc}/rd`}
                  </em>
                )}
              </span>
            </div>
          );
        })}
      </div>

      {inCampaign && econ.hasOwnCapital && (
        <div className="capital-status active">
          ★ Capital HQ active (+{CAPITAL_CURRENCY_BONUS} ¤ · +{CAPITAL_HUMANS_BONUS} H · +15% def)
        </div>
      )}
      {inCampaign && econ.ownCapitalLost && (
        <div className="capital-status lost">⚠ Capital captured — HQ bonus lost</div>
      )}

      {phase === 'plan' && (
        <div className="reserved">
          Movement reserved <b>¤ {game.reserved}</b>
        </div>
      )}

      <div className="section-label">
        COMMANDERS <span>{game.players.length}/4</span>
      </div>
      <div className="players">
        {game.players.map(p => {
          const picking = phase === 'draft' && game.turn === p.id;
          const showVP = p.id === player || phase === 'finished' || Boolean(me?.eliminated);
          const displayedVP = p.id === player ? myVP : p.vp;
          return (
            <div
              className={'player' + (picking ? ' active-turn' : '')}
              key={p.id}
              style={picking ? ({ '--turn-color': p.color } as React.CSSProperties) : undefined}
            >
              <span className="player-dot" style={{ background: p.color }} />
              <span>
                {p.name}
                {p.id === player && <small> YOU</small>}
                {p.bot && <small> BOT</small>}
                {picking && <i className="turn-tag">PICKING</i>}
                {showVP && inCampaign && <small> · {displayedVP} VP</small>}
              </span>
              <b>
                {p.disconnectedUntil > 0
                  ? `OFFLINE · ${Math.max(0, Math.floor(Math.ceil((p.disconnectedUntil - now) / 1000) / 60))}:${String(Math.max(0, Math.ceil((p.disconnectedUntil - now) / 1000)) % 60).padStart(2, '0')}`
                  : p.eliminated
                    ? 'OUT'
                    : phase === 'draft' || phase === 'roll'
                      ? `⚄ ${p.roll}`
                      : phase === 'finished'
                        ? `${p.vp} VP`
                        : p.ready
                          ? '✓'
                          : '—'}
              </b>
            </div>
          );
        })}
      </div>

      {phase === 'lobby' && (
        <div className="lobby-actions">
          {game.host === player ? (
            <>
              <button className="secondary" disabled={game.players.length >= 4} onClick={onAddBot}>
                + Add bot commander
              </button>
              <button className="primary" disabled={game.players.length < 2} onClick={onStart}>
                Roll dice &amp; start →
              </button>
            </>
          ) : (
            <p>Waiting for the host to start.</p>
          )}
          <p>
            Invite friends using room code <b>{game.code}</b>.
          </p>
        </div>
      )}

      <div className="left-bottom">
        <span className="eyebrow">INTELLIGENCE BRIEF</span>
        <p>
          {full
            ? phase === 'capital'
              ? 'All drafted sectors are visible. Rival capital selections stay secret until Round 1.'
              : 'The full continent is visible.'
            : phase === 'gather'
              ? 'Only your territories are visible during Gather. Static terrain yields remain visible across the map.'
              : 'Your territories and adjacent tiles are visible during Plan. Static terrain yields remain visible across the map.'}
        </p>
        <div className="visibility">
          <span className="live-dot" />
          {full ? 'FULL MAP VISIBILITY' : phase === 'gather' ? 'OWN TERRITORIES ONLY' : 'ADJACENT TILE VISIBILITY'}
        </div>
      </div>
    </aside>
  );
}
