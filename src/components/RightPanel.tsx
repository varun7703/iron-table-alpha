import React, { useEffect, useState } from 'react';
import {
  CAPITAL_CURRENCY_BONUS,
  CAPITAL_HUMANS_BONUS,
  CAPITAL_VP,
  isOnOwnCapital,
  MAX_STACKS_PER_SECTOR,
  moveCost,
  rationMultiplier,
  unitCost,
  type EconomySummary,
} from '../combat';
import type { BattleFrame, Game, GameEvent, Order, Player, Territory, Unit } from '../types';
import { CombatStats } from './CombatStats';
import { Icon } from './Icons';

function orderLabel(order: Order | undefined, territories: Territory[]): string {
  if (!order) return '';
  if (order.kind === 'defend') return '🛡 Entrenched';
  const targetName = territories[order.target]?.name ?? `Sector ${order.target + 1}`;
  if (order.kind === 'bombard') return `⌖ Bombard ${targetName}`;
  return `→ ${targetName}`;
}

export function RightPanel({
  game,
  me,
  player,
  phase,
  connected,
  exiting,
  locked,
  econ,
  territory,
  unit,
  army,
  mode,
  amount,
  rightTab,
  frame,
  previousFrame,
  displayUnits,
  displayTerritories,
  visibleEvents,
  currentEvent,
  changedUnits,
  destroyedUnits,
  onSetRightTab,
  onChooseArmy,
  onClearArmy,
  onSetMode,
  onSetAmount,
  onSend,
}: {
  game: Game;
  me?: Player;
  player: string;
  phase: string;
  connected: boolean;
  exiting: boolean;
  locked: boolean;
  econ: EconomySummary;
  territory?: Territory;
  unit?: Unit;
  army: string;
  mode: string;
  amount: number;
  rightTab: 'sector' | 'report';
  frame?: BattleFrame;
  previousFrame?: BattleFrame;
  displayUnits: Unit[];
  displayTerritories: Territory[];
  visibleEvents: GameEvent[];
  currentEvent?: GameEvent;
  changedUnits: Unit[];
  destroyedUnits: Unit[];
  onSetRightTab: (tab: 'sector' | 'report') => void;
  onChooseArmy: (u: Unit) => void;
  onClearArmy: () => void;
  onSetMode: (m: string) => void;
  onSetAmount: (n: number) => void;
  onSend: (action: string, data?: Record<string, unknown>) => void;
}) {
  const [splitCount, setSplitCount] = useState(1);

  useEffect(() => {
    if (unit && unit.count >= 2) {
      setSplitCount(Math.max(1, Math.min(unit.count - 1, Math.floor(unit.count / 2))));
    } else {
      setSplitCount(1);
    }
  }, [unit?.id, unit?.count]);

  const sectorUnits = territory ? displayUnits.filter(u => u.territory === territory.id) : [];
  const ownUnits = sectorUnits.filter(u => u.owner === player);
  const enemyUnits = sectorUnits.filter(u => u.owner !== player);
  const sectorStacks = sectorUnits.length;
  const capitalOwner = territory ? game.players.find(p => p.capital === territory.id) : undefined;
  const isCapitalTile = Boolean(territory && (territory.isCapital || capitalOwner));
  const canShowTabs = phase === 'gather' || phase === 'plan';
  const lastEvents = game.lastEvents || [];
  const lastReportRound = phase === 'gather' ? game.round : Math.max(1, game.round - 1);
  const isBorderClashFrame = Boolean(frame?.label.startsWith('Border clashes'));

  const canSplitSelected =
    Boolean(unit) &&
    unit!.owner === player &&
    unit!.count >= 2 &&
    sectorStacks < MAX_STACKS_PER_SECTOR &&
    phase === 'gather' && !unit!.fed;

  const renderSplitControls = () => {
    if (phase !== 'gather' || !unit || unit.owner !== player || unit.count < 2) return null;
    if (phase === 'gather' && unit.fed) return null;
    const capped = sectorStacks >= MAX_STACKS_PER_SECTOR;
    const clampedSplit = Math.max(1, Math.min(unit.count - 1, splitCount));
    const kept = unit.count - clampedSplit;
    return (
      <div className="split-box">
        <div className="section-label">
          SPLIT ARMY{' '}
          <span>
            {sectorStacks}/{MAX_STACKS_PER_SECTOR} STACKS
          </span>
        </div>
        {capped ? (
          <p className="muted">
            {`Sector at ${MAX_STACKS_PER_SECTOR}-army limit. Merge existing stacks first.`}
          </p>
        ) : (
          <>
            {unit.count > 2 && (
              <div className="split-controls">
                <input
                  type="range"
                  min={1}
                  max={unit.count - 1}
                  value={clampedSplit}
                  disabled={locked}
                  aria-label="Troops to split off"
                  onChange={e => setSplitCount(Number(e.target.value))}
                />
                <span className="split-readout">
                  {kept} + {clampedSplit}
                </span>
              </div>
            )}
            <button
              className="secondary"
              disabled={locked || !canSplitSelected}
              onClick={() => {
                onSend('split', { unit: unit.id, count: clampedSplit });
                onClearArmy();
              }}
            >
              Split detachment ({kept} keep / {clampedSplit} detach)
            </button>
          </>
        )}
      </div>
    );
  };

  return (
    <aside className="right-panel">
      <div className="panel-heading">
        {phase === 'simulate'
          ? 'BATTLE REPORT'
          : rightTab === 'report' && canShowTabs
            ? 'LAST ROUND REPORT'
            : 'SECTOR INTELLIGENCE'}
        <span>↗</span>
      </div>

      {canShowTabs && (
        <div className="panel-tabs" role="tablist" aria-label="Right panel view">
          <button
            id="tab-sector"
            role="tab"
            aria-selected={rightTab === 'sector'}
            aria-controls="panel-sector"
            className={rightTab === 'sector' ? 'active' : ''}
            onClick={() => onSetRightTab('sector')}
          >
            Sector Intel
          </button>
          <button
            id="tab-report"
            role="tab"
            aria-selected={rightTab === 'report'}
            aria-controls="panel-report"
            className={rightTab === 'report' ? 'active' : ''}
            onClick={() => onSetRightTab('report')}
          >
            Last Round ({lastEvents.length})
          </button>
        </div>
      )}

      {phase === 'finished' ? (
        <div className="empty-panel">
          <span>⚑</span>
          <h2>Campaign complete</h2>
          <p>{game.winner}</p>
          <button className="secondary" disabled={!connected || exiting} onClick={() => onSend('leave')}>
            Return to war room
          </button>
        </div>
      ) : phase === 'simulate' ? (
        <div className="report">
          <span className="eyebrow">ROUND {game.round} / FIELD DISPATCHES</span>
          <h3>{frame?.label}</h3>
          <div className="battle-armies">
            {displayUnits
              .filter(u => !currentEvent || u.territory === currentEvent.from || u.territory === currentEvent.to)
              .map(u => {
                const owner = game.players.find(p => p.id === u.owner);
                return (
                  <div className="battle-army" key={u.id}>
                    <strong style={{ color: owner?.color }}>
                      {owner?.name} · {u.type} ×{u.count}
                    </strong>
                    <div className="health">
                      <i style={{ width: `${u.health}%` }} />
                    </div>
                    <span>{u.health.toFixed(2)}% health</span>
                    <CombatStats
                      u={u}
                      entrenched={!frame?.label.startsWith('Retreats') && game.orders[u.id]?.kind === 'defend'}
                      onOwnCapital={!isBorderClashFrame && isOnOwnCapital(game, u, displayTerritories)}
                    />
                  </div>
                );
              })}
          </div>
          <div aria-live="polite">
            {changedUnits.map(u => {
              const old = previousFrame!.units.find(v => v.id === u.id)!;
              return (
                <p className="damage-change" key={u.id}>
                  {u.type}: {old.count} → {u.count} troops · {old.health.toFixed(2)}% → {u.health.toFixed(2)}% health
                </p>
              );
            })}
            {destroyedUnits.map(u => (
              <p className="damage-change" key={u.id}>
                {u.type} ×{u.count} destroyed
              </p>
            ))}
          </div>
          {visibleEvents.length ? (
            visibleEvents.map((e, i) => (
              <div className={'dispatch ' + (i === visibleEvents.length - 1 ? 'current-dispatch' : '')} key={i}>
                <small>
                  {String(i + 1).padStart(2, '0')} / {e.kind.toUpperCase()}
                </small>
                <p>{e.text}</p>
              </div>
            ))
          ) : (
            <p>No engagements within line of sight.</p>
          )}
        </div>
      ) : canShowTabs && rightTab === 'report' ? (
        <div id="panel-report" role="tabpanel" aria-labelledby="tab-report" className="report">
          <span className="eyebrow">ROUND {lastReportRound} / LINE-OF-SIGHT DISPATCHES</span>
          {lastEvents.length ? (
            lastEvents.map((e, i) => (
              <div className="dispatch" key={i}>
                <small>
                  {String(i + 1).padStart(2, '0')} / {e.kind.toUpperCase()}
                </small>
                <p>{e.text}</p>
              </div>
            ))
          ) : (
            <p>No dispatches recorded within line of sight last round.</p>
          )}
        </div>
      ) : territory ? (
        <div
          id={canShowTabs ? 'panel-sector' : undefined}
          role={canShowTabs ? 'tabpanel' : undefined}
          aria-labelledby={canShowTabs ? 'tab-sector' : undefined}
          className="sector"
        >
          <span className="eyebrow">SECTOR {String(territory.id + 1).padStart(2, '0')}</span>
          <h2>{territory.name}</h2>
          <p className="ownership">
            {!territory.visible
              ? 'Outside line of sight · static terrain survey'
              : territory.owner
                ? game.players.find(p => p.id === territory.owner)?.name
                : 'Neutral · undefended'}
          </p>

          {isCapitalTile && territory.visible && (
            <div className="capital-badge">
              ★{' '}
              {capitalOwner
                ? capitalOwner.id === player
                  ? 'Your Capital HQ'
                  : `${capitalOwner.name}’s Capital`
                : 'Conquered Capital'}{' '}
              ·{' '}
              {capitalOwner && capitalOwner.id === territory.owner
                ? `+${CAPITAL_CURRENCY_BONUS} ¤ · +${CAPITAL_HUMANS_BONUS} H · +15% def · +${CAPITAL_VP} VP`
                : `Captured (+${CAPITAL_VP} VP to holder)`}
            </div>
          )}

          {phase === 'capital' && territory.owner === player && (
            <button
              className="primary"
              disabled={!connected || (me?.capital ?? -1) >= 0}
              onClick={() => onSend('capital', { target: territory.id })}
            >
              {(me?.capital ?? -1) >= 0 ? 'Capital confirmed ✓' : 'Choose as capital ★'}
            </button>
          )}

          <div className="yield-row">
            <span>
              {territory.resource === 'food' ? '♧' : '⬡'} {territory.yield} {territory.resource}
            </span>
            <span>+{territory.humans} humans</span>
            <span>¤ {territory.currency}</span>
          </div>

          <div className="section-label">
            GARRISON{' '}
            <span>
              {territory.visible ? `${sectorStacks}/${MAX_STACKS_PER_SECTOR} ARMIES` : 'UNKNOWN'}
            </span>
          </div>

          {ownUnits.map(u => {
            const ord = phase === 'plan' ? game.orders[u.id] : undefined;
            const badge = orderLabel(ord, game.territories);
            return (
              <button
                key={u.id}
                className={'unit-card ' + (army === u.id ? 'active' : '')}
                onClick={() => onChooseArmy(u)}
              >
                <Icon type={u.type} size={30} />
                <div>
                  <strong>
                    {u.type}
                    <span>×{u.count}</span>
                  </strong>
                  <div className="health">
                    <i style={{ width: `${u.health}%` }} />
                  </div>
                  <small>
                    {Math.round(u.health)}% HEALTH ·{' '}
                    {u.fed
                      ? 'FED'
                      : phase === 'gather'
                        ? 'NEEDS RATIONS'
                        : `${Math.round(rationMultiplier(u.ration) * 100)}% ATTACK`}
                  </small>
                  {badge && <div className="order-badge">{badge}</div>}
                  <CombatStats
                    u={u}
                    entrenched={phase === 'plan' && ord?.kind === 'defend'}
                    onOwnCapital={isOnOwnCapital(game, u, displayTerritories)}
                  />
                </div>
              </button>
            );
          })}

          {enemyUnits.map(u => {
            const owner = game.players.find(p => p.id === u.owner);
            return (
              <div key={u.id} className="unit-card enemy-intel">
                <Icon type={u.type} size={30} />
                <div>
                  <strong>
                    <span style={{ color: owner?.color }}>
                      {owner?.name} · {u.type}
                    </span>
                    <span>×{u.count}</span>
                  </strong>
                  <div className="health">
                    <i style={{ width: `${u.health}%` }} />
                  </div>
                  <small>SCOUTED GARRISON · {Math.round(u.health)}% HEALTH</small>
                  <CombatStats u={u} onOwnCapital={isOnOwnCapital(game, u, displayTerritories)} />
                </div>
              </div>
            );
          })}

          {!ownUnits.length && !enemyUnits.length && (
            <p className="muted">
              {territory.visible
                ? 'No armies stationed here.'
                : phase === 'gather'
                  ? 'Only your territories are visible during Gather.'
                  : 'Sector is beyond adjacent-tile visibility. Capture a neighboring territory to scout it.'}
            </p>
          )}

          {unit && unit.owner === player && phase === 'plan' && (
            <div className="actions">
              <div className="section-label">ISSUE ORDER</div>
              <div className="segmented">
                <button disabled={locked} className={mode === 'move' ? 'active' : ''} onClick={() => onSetMode('move')}>
                  Move / attack
                </button>
                {unit.type === 'artillery' && (
                  <button
                    disabled={locked}
                    className={mode === 'bombard' ? 'active' : ''}
                    onClick={() => onSetMode('bombard')}
                  >
                    Bombard
                  </button>
                )}
              </div>
              <p className="muted">
                {mode === 'move'
                  ? `Attack / neutral move: ¤ ${moveCost(unit)} · Friendly transfer: free.`
                  : 'Select an adjacent territory. Fires after movement (×1.5 vs tanks).'}
              </p>
              <button
                className={'secondary ' + (game.orders[unit.id]?.kind === 'defend' ? 'entrenched-btn' : '')}
                disabled={locked}
                onClick={() => {
                  onSend('order', { unit: unit.id, kind: 'defend', target: unit.territory });
                  onClearArmy();
                }}
              >
                {game.orders[unit.id]?.kind === 'defend'
                  ? 'Holding & entrenched ✓'
                  : 'Hold & defend (+20% · −25% bombard)'}
              </button>
              {game.orders[unit.id] && (
                <button className="text-button" disabled={locked} onClick={() => onSend('cancel', { unit: unit.id })}>
                  Cancel current order
                </button>
              )}
            </div>
          )}

          {phase === 'gather' && territory.owner === player && (
            <div className="actions">
              {me && me.resources.currency < 12 && econ.hasBorderArmies && (
                <div className="reserve-warning">
                  Low Treasury (¤ {me.resources.currency}) — keep currency in reserve for hostile attacks next Plan.
                </div>
              )}
              <div className="section-label">RECRUITMENT</div>
              <label>
                NUMBER OF TROOPS
                <input
                  type="number"
                  min="1"
                  max="100"
                  step="1"
                  value={amount}
                  onChange={e => onSetAmount(Math.max(1, Math.min(100, Math.trunc(Number(e.target.value)) || 1)))}
                />
              </label>
              {(['infantry', 'artillery', 'tank'] as const).map(t => {
                const c = unitCost(t, amount);
                return (
                  <button
                    key={t}
                    className="recruit-button"
                    disabled={locked || sectorStacks >= MAX_STACKS_PER_SECTOR}
                    onClick={() => onSend('recruit', { type: t, count: amount, target: territory.id })}
                  >
                    <Icon type={t} />
                    <span>
                      {t}
                      <small>
                        {c.humans} humans · ¤ {c.currency}
                        {c.iron > 0 ? ` · ${c.iron} iron` : ''}
                      </small>
                    </span>
                    <b>+</b>
                  </button>
                );
              })}
              {sectorStacks >= MAX_STACKS_PER_SECTOR && (
                <p className="muted">
                  Sector at {MAX_STACKS_PER_SECTOR}/{MAX_STACKS_PER_SECTOR} stacks — reinforce or merge existing armies.
                </p>
              )}

              {unit && unit.owner === player && (
                <>
                  <div className="section-label">ARMY RATIONS</div>
                  <div className="ration-buttons">
                    {([0.5, 1, 1.5] as const).map((r, i) => (
                      <button
                        key={r}
                        disabled={locked || unit.fed}
                        onClick={() => onSend('feed', { unit: unit.id, ration: r })}
                      >
                        <b>{['Minimum', 'Normal', 'Extra'][i]}</b>
                        <small>{Math.ceil(unit.count * r)} food</small>
                      </button>
                    ))}
                  </div>
                  <p className="muted">
                    {unit.fed ? 'Rations allocated for this round.' : 'Minimum: reduced attack. Extra: +20 health.'}
                  </p>
                  <button
                    className="secondary"
                    disabled={locked || unit.fed}
                    onClick={() => onSend('reinforce', { unit: unit.id, count: amount })}
                  >
                    Reinforce +{amount} troops
                  </button>
                  {ownUnits
                    .filter(u => u.id !== unit.id && u.type === unit.type && !u.fed)
                    .map(u => (
                      <button
                        className="secondary"
                        key={u.id}
                        disabled={locked || unit.fed}
                        onClick={() => onSend('merge', { unit: unit.id, other: u.id })}
                      >
                        Merge with {u.count} {u.type}
                      </button>
                    ))}
                  {renderSplitControls()}
                </>
              )}
            </div>
          )}
        </div>
      ) : (
        <div
          id={canShowTabs ? 'panel-sector' : undefined}
          role={canShowTabs ? 'tabpanel' : undefined}
          aria-labelledby={canShowTabs ? 'tab-sector' : undefined}
          className="empty-panel"
        >
          <span>⌖</span>
          <h3>{phase === 'draft' ? 'Choose your foothold.' : 'Read the battlefield.'}</h3>
          <p>
            {phase === 'draft'
              ? 'Select an unclaimed territory when it is your turn.'
              : 'Select a territory to inspect production and command its armies.'}
          </p>
          <div className="unit-legend">
            {(['infantry', 'artillery', 'tank'] as const).map(t => (
              <div key={t}>
                <Icon type={t} size={30} />
                <span>{t}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="alpha-note">
        ALPHA FIELD BUILD
        <br />
        <span>Balance values are provisional.</span>
      </div>
    </aside>
  );
}
