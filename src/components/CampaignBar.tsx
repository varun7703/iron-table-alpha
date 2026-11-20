import React from 'react';
import type { Game, Player } from '../types';

export function CampaignBar({
  game,
  me,
  phase,
  seconds,
  connected,
  readyCount,
  activeCount,
  draftTurnPlayer,
  isMyDraftTurn,
  draftPickNumber,
  onToggleReady,
}: {
  game: Game;
  me?: Player;
  phase: string;
  seconds: number;
  connected: boolean;
  readyCount: number;
  activeCount: number;
  draftTurnPlayer?: Player;
  isMyDraftTurn: boolean;
  draftPickNumber: number;
  onToggleReady: () => void;
}) {
  const countdownLabel =
    phase === 'capital'
      ? 'CAPITAL TIMER'
      : phase === 'roll'
        ? 'ROLLING INITIATIVE'
        : phase === 'gather'
          ? 'GATHER TIMER'
          : phase === 'finished'
            ? 'CAMPAIGN COMPLETE'
            : phase === 'lobby'
              ? 'STANDING BY'
              : phase === 'draft'
                ? 'DRAFT TIMER'
                : 'PHASE ENDS IN';

  const countdownValue =
    phase === 'lobby' || phase === 'finished'
      ? '—'
      : `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;

  return (
    <section className="campaign-bar">
      <div>
        <span className="eyebrow">OPERATION / THE DIVIDED CONTINENT</span>
        <h1>
          {game.phase === 'lobby' ? 'War council' : `Round ${String(game.round || 1).padStart(2, '0')}`}{' '}
          <span>/ 20</span>
        </h1>
      </div>

      <div className="phase-track">
        {(['plan', 'simulate', 'gather'] as const).map((p, i) => (
          <div key={p} className={phase === p ? 'active' : ''}>
            <span>{String(i + 1).padStart(2, '0')}</span>
            {p}
            <small>{p === 'plan' ? 'ISSUE ORDERS' : p === 'simulate' ? 'WATCH THE BATTLE' : 'SUPPLY & RECRUIT'}</small>
          </div>
        ))}
      </div>

      <div className="phase-controls">
        <div className="countdown">
          <span>{countdownLabel}</span>
          <strong>{countdownValue}</strong>
        </div>
        {phase === 'draft' && draftTurnPlayer && (
          <div
            className={'draft-turn ' + (isMyDraftTurn ? 'my-turn' : '')}
            role="status"
            aria-live="polite"
            style={{ '--turn-color': draftTurnPlayer.color } as React.CSSProperties}
          >
            <span className="draft-turn-dot" />
            <div>
              <span>{isMyDraftTurn ? 'YOUR TURN' : 'WAITING FOR PICK'}</span>
              <strong>
                {isMyDraftTurn
                  ? `Claim sector (${draftPickNumber}/4)`
                  : `${draftTurnPlayer.name} (${draftPickNumber}/4)`}
              </strong>
            </div>
          </div>
        )}
        {['plan', 'simulate', 'gather'].includes(phase) && (
          <div className="lock-control">
            <button
              className={me?.ready ? 'secondary' : 'primary'}
              disabled={Boolean(me?.eliminated || !connected)}
              onClick={onToggleReady}
            >
              {me?.eliminated
                ? 'Eliminated · spectating'
                : me?.ready
                  ? 'Locked in ✓ · Unlock'
                  : phase === 'plan'
                    ? 'Lock orders →'
                    : phase === 'simulate'
                      ? game.round >= 20
                        ? 'Finish campaign →'
                        : 'Continue to Gather →'
                      : 'Lock Gather →'}
            </button>
            <small role="status">
              {readyCount}/{activeCount} active commanders locked in
            </small>
          </div>
        )}
      </div>
    </section>
  );
}
