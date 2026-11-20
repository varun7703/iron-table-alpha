import type { BattleFrame, GameEvent, Unit } from './types.ts';

export type MotionUnit = Unit;
export type MotionEvent = GameEvent;
export type MotionFrame = BattleFrame;

export const clamp = (x: number): number => Math.max(0, Math.min(1, x));

export function sampleBattle(
  before: MotionFrame,
  after: MotionFrame,
  t: number,
  tiles: { x: number; y: number }[],
  reduced: boolean,
) {
  const impact = 0.62;
  const move = clamp((t - 0.08) / 0.54);
  const travel = reduced ? (t < impact ? 0 : 1) : move * move * (3 - 2 * move);
  const units: MotionUnit[] = before.units.flatMap((old): MotionUnit[] => {
    const next = after.units.find(u => u.id === old.id);
    if (!next) return t < impact ? [{ ...old, moving: false }] : [];
    const a = tiles[old.territory];
    const b = tiles[next.territory];
    const healthT = reduced ? (t < impact ? 0 : 1) : clamp((t - impact) / 0.2);
    const isMoving = old.territory !== next.territory;
    if (!a || !b) {
      return [
        {
          ...next,
          count: t < impact ? old.count : next.count,
          health: old.health + (next.health - old.health) * healthT,
          territory: t < impact ? old.territory : next.territory,
          moving: false,
        },
      ];
    }
    return [
      {
        ...next,
        count: t < impact ? old.count : next.count,
        health: old.health + (next.health - old.health) * healthT,
        territory: t < impact ? old.territory : next.territory,
        motionX: a.x + (b.x - a.x) * travel,
        motionY: a.y + (b.y - a.y) * travel - Math.sin(travel * Math.PI) * (isMoving ? 16 : 0),
        moving: isMoving,
        destTerritory: next.territory,
      },
    ];
  });
  // Units newly emerging out of fog appear at impact rather than popping in at t = 0.
  if (t >= impact) {
    for (const u of after.units) {
      if (!before.units.some(v => v.id === u.id)) {
        units.push({ ...u, moving: false });
      }
    }
  }
  return { units, owners: t < impact ? before.owners : after.owners };
}
