import React from 'react';
import { strength } from '../combat';
import type { Unit } from '../types';

export function CombatStats({
  u,
  entrenched = false,
  onOwnCapital = false,
}: {
  u: Unit;
  entrenched?: boolean;
  onOwnCapital?: boolean;
}) {
  const tags: string[] = [];
  if (entrenched) tags.push('ENTRENCHED');
  if (onOwnCapital) tags.push('HQ +15%');
  const notes: string[] = [];
  if (entrenched) notes.push('+20% · −25% bombard');
  if (onOwnCapital) notes.push('+15% capital HQ');

  return (
    <div className="combat-stats">
      <b>
        Combat strength: {strength(u, '', false, entrenched, onOwnCapital).toFixed(2)}
        {tags.length > 0 ? ` · ${tags.join(' · ')}` : ''}
      </b>
      <span>
        vs infantry {strength(u, 'infantry', false, entrenched, onOwnCapital).toFixed(2)} · tanks{' '}
        {strength(u, 'tank', false, entrenched, onOwnCapital).toFixed(2)} · artillery{' '}
        {strength(u, 'artillery', false, entrenched, onOwnCapital).toFixed(2)}
        {notes.length > 0 ? ` (${notes.join(', ')})` : ''}
      </span>
      {u.type === 'artillery' && (
        <>
          <b>Ranged: {strength(u, '', true).toFixed(2)}</b>
          <span>
            vs infantry {strength(u, 'infantry', true).toFixed(2)} · tanks {strength(u, 'tank', true).toFixed(2)} ·
            artillery {strength(u, 'artillery', true).toFixed(2)}
          </span>
        </>
      )}
    </div>
  );
}
