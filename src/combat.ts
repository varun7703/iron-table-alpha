import type { Game, ResourceKey, Resources, Territory, Unit } from './types.ts';

export const MAX_STACKS_PER_SECTOR = 4;
export const CAPITAL_CURRENCY_BONUS = 4;
export const CAPITAL_HUMANS_BONUS = 2;
export const CAPITAL_DEFENSE_BONUS = 1.15;
export const CAPITAL_VP = 2;

export const RESOURCE_KEYS: readonly ResourceKey[] = ['humans', 'food', 'iron', 'currency'] as const;

export const RESOURCE_LABELS: Record<ResourceKey, string> = {
  humans: 'Manpower',
  food: 'Food',
  iron: 'Iron',
  currency: 'Treasury',
};

export function moveRate(type: string): number {
  if (type === 'tank') return 3;
  if (type === 'artillery') return 2;
  return 1;
}

export function unitCost(type: string, count: number): Resources {
  if (type === 'tank') {
    return { humans: 3 * count, food: 0, iron: 8 * count, currency: 8 * count };
  }
  if (type === 'artillery') {
    return { humans: 2 * count, food: 0, iron: 4 * count, currency: 4 * count };
  }
  return { humans: count, food: 0, iron: 0, currency: 4 * count };
}

export function moveCost(u: Pick<Unit, 'type' | 'count' | 'owner'>, target?: Pick<Territory, 'owner'>): number {
  if (target && target.owner === u.owner) return 0;
  return u.count * moveRate(u.type);
}

export function rationMultiplier(ration: number): number {
  if (ration === 0) return 0.3;
  if (ration < 1) return 0.65;
  return 1;
}

export function strength(
  u: Pick<Unit, 'type' | 'count' | 'health' | 'ration'>,
  target = '',
  ranged = false,
  entrenched = false,
  onOwnCapital = false,
): number {
  const base = u.type === 'tank' ? 6 : u.type === 'artillery' ? (ranged ? 4 : 1) : 2;
  let modifier = 1;
  if (target === 'artillery' && u.type !== 'artillery') {
    modifier = 1.5;
  } else if (u.type === 'tank' && target === 'infantry') {
    modifier = 1.4;
  } else if (u.type === 'artillery' && ranged) {
    modifier = target === 'tank' ? 1.5 : target === 'infantry' ? 1.2 : 1;
  }
  const defend = !ranged && entrenched ? 1.2 : 1;
  const capital = !ranged && onOwnCapital ? CAPITAL_DEFENSE_BONUS : 1;
  return u.count * base * (u.health / 100) * rationMultiplier(u.ration) * modifier * defend * capital;
}

export function isOnOwnCapital(
  game: Pick<Game, 'players' | 'territories'>,
  u: Pick<Unit, 'owner' | 'territory'>,
  territoriesOverride?: Pick<Territory, 'id' | 'owner'>[],
): boolean {
  const owner = game.players.find(p => p.id === u.owner);
  if (!owner || owner.capital < 0 || owner.capital !== u.territory) return false;
  const tiles = territoriesOverride ?? game.territories;
  const tile = tiles[u.territory];
  return Boolean(tile && tile.owner === u.owner);
}

export type EconomySummary = {
  income: Resources;
  foodUpkeep: number;
  foodNet: number;
  hasOwnCapital: boolean;
  ownCapitalLost: boolean;
  territoriesHeld: number;
  capitalsControlled: number;
  vp: number;
  hasBorderArmies: boolean;
};

export function calcEconomy(
  game: Pick<Game, 'players' | 'territories' | 'units'>,
  playerId: string,
  territoriesOverride?: Pick<Territory, 'id' | 'owner' | 'resource' | 'yield' | 'humans' | 'currency' | 'neighbors' | 'isCapital'>[],
  unitsOverride?: Pick<Unit, 'owner' | 'territory' | 'count'>[],
): EconomySummary {
  const tiles = territoriesOverride ?? game.territories;
  const units = unitsOverride ?? game.units;
  const income: Resources = { humans: 0, food: 0, iron: 0, currency: 0 };
  let territoriesHeld = 0;
  let capitalsControlled = 0;
  for (const t of tiles) {
    if (t.owner !== playerId) continue;
    territoriesHeld++;
    income.humans += t.humans;
    income.currency += t.currency;
    if (t.resource === 'food') {
      income.food += t.yield;
    } else if (t.resource === 'iron') {
      income.iron += t.yield;
    }
    const isCap = Boolean(t.isCapital) || game.players.some(p => p.capital === t.id);
    if (isCap) {
      capitalsControlled++;
    }
  }
  const me = game.players.find(p => p.id === playerId);
  const hasOwnCapital = Boolean(me && me.capital >= 0 && tiles[me.capital]?.owner === playerId);
  const ownCapitalLost = Boolean(me && me.capital >= 0 && !hasOwnCapital);
  if (hasOwnCapital) {
    income.currency += CAPITAL_CURRENCY_BONUS;
    income.humans += CAPITAL_HUMANS_BONUS;
  }
  let foodUpkeep = 0;
  let hasBorderArmies = false;
  for (const u of units) {
    if (u.owner !== playerId || u.count <= 0) continue;
    foodUpkeep += u.count;
    const tile = tiles[u.territory];
    if (tile && tile.neighbors.some(n => tiles[n]?.owner !== playerId)) {
      hasBorderArmies = true;
    }
  }
  return {
    income,
    foodUpkeep,
    foodNet: income.food - foodUpkeep,
    hasOwnCapital,
    ownCapitalLost,
    territoriesHeld,
    capitalsControlled,
    vp: territoriesHeld + CAPITAL_VP * capitalsControlled,
    hasBorderArmies,
  };
}
