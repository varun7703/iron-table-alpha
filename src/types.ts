export type ResourceKey = 'humans' | 'food' | 'iron' | 'currency';

export type Resources = {
  humans: number;
  food: number;
  iron: number;
  currency: number;
};

export type Player = {
  disconnectedUntil: number;
  capital: number;
  id: string;
  name: string;
  color: string;
  roll: number;
  ready: boolean;
  eliminated: boolean;
  vp: number;
  bot: boolean;
  resources: Resources;
};

export type Territory = {
  neutral: boolean;
  visible: boolean;
  isCapital?: boolean;
  id: number;
  name: string;
  owner: string;
  resource: string;
  yield: number;
  humans: number;
  currency: number;
  neighbors: number[];
  x: number;
  y: number;
  points: string;
};

export type Unit = {
  id: string;
  owner: string;
  type: string;
  territory: number;
  count: number;
  health: number;
  ration: number;
  fed: boolean;
  motionX?: number;
  motionY?: number;
  moving?: boolean;
  destTerritory?: number;
};

export type Order = {
  unit: string;
  kind: string;
  target: number;
};

export type GameEvent = {
  text: string;
  from: number;
  to: number;
  kind: string;
};

export type BattleFrame = {
  label: string;
  units: Unit[];
  owners: string[];
  events: GameEvent[];
  destroyed?: string[];
};

export type Game = {
  rollIndex: number;
  timeline: BattleFrame[];
  simulationDuration: number;
  code: string;
  host: string;
  phase: string;
  round: number;
  deadline: number;
  players: Player[];
  territories: Territory[];
  units: Unit[];
  orders: Record<string, Order>;
  events: GameEvent[];
  lastEvents: GameEvent[];
  turn: string;
  winner: string;
  reserved: number;
};
