import assert from 'node:assert/strict';
import { sampleBattle } from '../src/battleMotion.ts';
import { calcEconomy, isOnOwnCapital, moveCost, strength, unitCost } from '../src/combat.ts';

// 1. Battle motion interpolation & moving flag (for snap-free badge rendering)
const u = { id: 'a', owner: 'p', type: 'tank', territory: 0, count: 5, health: 100, ration: 1, fed: true };
const stationary = { id: 's', owner: 'p', type: 'infantry', territory: 1, count: 4, health: 100, ration: 1, fed: true };
const before = { label: 'Before', units: [u, stationary], owners: ['p', ''], events: [] };
const after = {
  label: 'After',
  units: [{ ...u, territory: 1, count: 3, health: 55 }, stationary],
  owners: ['p', 'p'],
  events: [],
};
const tiles = [
  { x: 0, y: 0 },
  { x: 100, y: 0 },
];

const start = sampleBattle(before, after, 0, tiles, false);
assert.equal(start.units[0].count, 5);
assert.equal(start.units[0].health, 100);
assert.equal(start.units[0].moving, true);
assert.equal(start.units[1].moving, false);
assert.equal(start.owners[1], '');

const flight = sampleBattle(before, after, 0.35, tiles, false);
assert(flight.units[0].motionX > 0 && flight.units[0].motionX < 100);
assert.equal(flight.units[0].count, 5);
assert.equal(flight.units[0].moving, true);

const impact = sampleBattle(before, after, 0.7, tiles, false);
assert.equal(impact.units[0].count, 3);
assert(impact.units[0].health < 100 && impact.units[0].health > 55);
assert.equal(impact.owners[1], 'p');

const end = sampleBattle(before, after, 1, tiles, false);
assert.equal(end.units[0].motionX, 100);
assert.equal(end.units[0].health, 55);
assert.equal(end.units[0].territory, 1);

const destroyed = { ...after, units: [stationary] };
assert.equal(sampleBattle(before, destroyed, 0.3, tiles, false).units.length, 2);
assert.equal(sampleBattle(before, destroyed, 0.8, tiles, false).units.length, 1);
assert.equal(sampleBattle(before, after, 0.3, tiles, true).units[0].motionX, 0);
assert.equal(sampleBattle(before, after, 0.8, tiles, true).units[0].motionX, 100);
assert.equal(before.units[0].count, 5);
assert.equal(before.units[0].health, 100);
assert.equal(flight.units[0].destTerritory, 1);

// 1b. Units emerging from fog must not pop in at t < 0.62; appear at impact (t >= 0.62).
const emerged = { id: 'fog-in', owner: 'e', type: 'infantry', territory: 1, count: 3, health: 100, ration: 1, fed: true };
const afterEmerge = { ...after, units: [...after.units, emerged] };
assert.equal(
  sampleBattle(before, afterEmerge, 0.3, tiles, false).units.some(x => x.id === 'fog-in'),
  false,
);
assert.equal(
  sampleBattle(before, afterEmerge, 0.7, tiles, false).units.some(x => x.id === 'fog-in'),
  true,
);

// 2. P2 combat & economy math (capital HQ defense, entrenchment stacking, income/upkeep/VP, unitCost, moveCost)
assert.deepEqual(unitCost('tank', 2), { humans: 6, food: 0, iron: 16, currency: 16 });
assert.deepEqual(unitCost('artillery', 3), { humans: 6, food: 0, iron: 12, currency: 12 });
assert.equal(moveCost({ type: 'tank', count: 2, owner: 'p1' }, { owner: 'p1' }), 0);
assert.equal(moveCost({ type: 'tank', count: 2, owner: 'p1' }, { owner: 'p2' }), 6);
const inf = { type: 'infantry', count: 5, health: 100, ration: 1 };
assert.ok(Math.abs(strength(inf, '', false, false, false) - 10.0) < 1e-9);
assert.ok(Math.abs(strength(inf, '', false, true, false) - 12.0) < 1e-9);
assert.ok(Math.abs(strength(inf, '', false, false, true) - 11.5) < 1e-9);
assert.ok(Math.abs(strength(inf, '', false, true, true) - 13.8) < 1e-9);
// Ranged bombardment does NOT get capital HQ defense bonus
const art = { type: 'artillery', count: 2, health: 100, ration: 1 };
assert.ok(Math.abs(strength(art, 'tank', true, true, true) - 12.0) < 1e-9);

const econGame = {
  players: [
    { id: 'p1', capital: 0 },
  ],
  territories: [
    { id: 0, owner: 'p1', isCapital: true, resource: 'food', yield: 10, humans: 3, currency: 8, neighbors: [1, 2] },
    { id: 1, owner: 'p1', isCapital: true, resource: 'iron', yield: 8, humans: 3, currency: 8, neighbors: [0, 2] },
    { id: 2, owner: 'p2', isCapital: false, resource: 'food', yield: 9, humans: 3, currency: 8, neighbors: [0, 1] },
  ],
  units: [
    { id: 'u1', owner: 'p1', territory: 0, count: 6 },
    { id: 'u2', owner: 'p1', territory: 1, count: 6 },
  ],
};
assert.equal(isOnOwnCapital(econGame, { owner: 'p1', territory: 0 }), true);
assert.equal(isOnOwnCapital(econGame, { owner: 'p1', territory: 1 }), false);
const econ = calcEconomy(econGame, 'p1');
assert.equal(econ.hasOwnCapital, true);
assert.equal(econ.ownCapitalLost, false);
assert.equal(econ.income.currency, 16 + 4);
assert.equal(econ.income.humans, 6 + 2);
assert.equal(econ.income.food, 10);
assert.equal(econ.income.iron, 8);
assert.equal(econ.foodUpkeep, 12);
assert.equal(econ.foodNet, -2);
assert.equal(econ.capitalsControlled, 2);
assert.equal(econ.vp, 2 + 4);
assert.equal(econ.hasBorderArmies, true);

console.log(
  'PASS: movement interpolation, moving flag, impact-timed casualties, health interpolation, capture timing, destruction, reduced motion, capital defense/economy math, and immutable snapshots.',
);
