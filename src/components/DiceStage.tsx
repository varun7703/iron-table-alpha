import React from 'react';
import type { Game } from '../types';

const DIE_FACES = ['', '⚀', '⚁', '⚂', '⚃', '⚄', '⚅'];

export function DiceStage({ game, now }: { game: Game; now: number }) {
  return (
    <section className="dice-stage" aria-live="polite">
      <span className="eyebrow">INITIATIVE ROLLS · ONE COMMANDER AT A TIME</span>
      <div className="dice-line">
        {game.players.map((p, i) => (
          <div key={p.id} className={'dice-player ' + (i === game.rollIndex ? 'rolling' : '')}>
            <strong>{p.name}</strong>
            <span className="die">
              {p.roll
                ? DIE_FACES[p.roll]
                : i === game.rollIndex
                  ? DIE_FACES[1 + (Math.floor(now / 150) % 6)]
                  : '—'}
            </span>
            <small>{p.roll ? 'Rolled ' + p.roll : i === game.rollIndex ? 'Rolling…' : 'Waiting'}</small>
          </div>
        ))}
      </div>
    </section>
  );
}
