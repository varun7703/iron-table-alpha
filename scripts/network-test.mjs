import assert from 'node:assert/strict';

const base = process.env.GAME_URL || 'http://127.0.0.1:8080';

class Commander {
  constructor() {
    this.ws = new WebSocket(base.replace('http', 'ws') + '/ws');
    this.messages = [];
    this.listeners = [];
    this.ws.addEventListener('message', e => {
      const m = JSON.parse(e.data);
      this.messages.push(m);
      if (m.type === 'state') this.game = m.game;
      if (m.type === 'session') this.session = m;
      for (const f of this.listeners) f();
    });
  }
  async connect() {
    await new Promise((resolve, reject) => {
      this.ws.addEventListener('open', resolve, { once: true });
      this.ws.addEventListener('error', reject, { once: true });
    });
  }
  wait(test, timeout = 5000) {
    if (test()) return Promise.resolve();
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.listeners = this.listeners.filter(x => x !== check);
        reject(new Error('Timed out waiting for server state'));
      }, timeout);
      const check = () => {
        if (test()) {
          clearTimeout(timer);
          this.listeners = this.listeners.filter(x => x !== check);
          resolve();
        }
      };
      this.listeners.push(check);
    });
  }
  async command(action, data = {}) {
    const id = crypto.randomUUID();
    this.ws.send(JSON.stringify({ id, action, ...data }));
    await this.wait(() => this.messages.some(m => m.id === id));
    const error = this.messages.find(m => m.type === 'error' && m.id === id);
    if (error) throw Error(error.message);
    return id;
  }
  close() {
    this.ws.close();
  }
}

const a = new Commander();
const b = new Commander();
let reconnect;

try {
  await Promise.all([a.connect(), b.connect()]);
  await a.command('create', { name: 'Network Alpha' });
  await b.command('join', { name: 'Network Bravo', code: a.session.code });
  await a.wait(() => a.game.players.length === 2);
  assert.equal(b.game.players.length, 2);

  // Initiative roll + Repeating Draft (8 picks)
  await a.command('start');
  await Promise.all([a.wait(() => a.game.phase === 'draft', 15000), b.wait(() => b.game.phase === 'draft', 15000)]);
  const picks = [];
  for (let i = 0; i < 8; i++) {
    const current = a.game.turn === a.session.player ? a : b;
    picks.push(current.session.player);
    await current.wait(() => current.game.phase === 'draft');
    // Pick tiles from opposite corners so some territories are 2+ hops apart for LoS verification
    const unclaimed = current.game.territories.filter(t => !t.owner);
    const t = current === a ? unclaimed[0] : unclaimed[unclaimed.length - 1];
    await current.command('claim', { target: t.id });
    await Promise.all([
      a.wait(() => a.game.territories[t.id].owner === current.session.player),
      b.wait(() => b.game.territories[t.id].owner === current.session.player),
    ]);
  }
  const first = picks[0];
  const second = picks[1];
  assert.notEqual(first, second);
  assert.deepEqual(picks, [first, second, first, second, first, second, first, second]);

  // Capital selection: 45s timer & concealed rival pick
  await Promise.all([a.wait(() => a.game.phase === 'capital'), b.wait(() => b.game.phase === 'capital')]);
  assert(a.game.deadline > Date.now(), 'Capital phase must have an AFK deadline');
  const aCapTile = a.game.territories.find(t => t.owner === a.session.player);
  const bCapTile = b.game.territories.find(t => t.owner === b.session.player);
  await a.command('capital', { target: aCapTile.id });
  await b.wait(() => b.game.phase === 'capital');
  assert.equal(
    b.game.players.find(p => p.id === a.session.player).capital,
    -1,
    'Rival capital pick must be concealed during capital phase',
  );
  await b.command('capital', { target: bCapTile.id });

  // Plan phase: 1-hop LoS (static yields visible everywhere, distant tiles fogged, private orders)
  await Promise.all([a.wait(() => a.game.phase === 'plan'), b.wait(() => b.game.phase === 'plan')]);
  assert(a.game.territories.every(t => t.yield > 0 && (t.resource === 'food' || t.resource === 'iron')));
  assert(a.game.territories.some(t => !t.visible && t.owner === '' && !t.neutral));
  assert(a.game.units.every(u => u.owner === a.session.player || a.game.territories[u.territory].visible));
  assert.equal(a.game.units.filter(u => u.owner === a.session.player).length, 4);

  // Army Split is rejected during Plan
  const splitTarget = a.game.units.find(v => v.owner === a.session.player && v.count === 4);
  assert(splitTarget);
  await assert.rejects(() => a.command('split', { unit: splitTarget.id, count: 2 }), /Gather/);
  assert.equal(a.game.units.filter(u => u.owner === a.session.player).length, 4);
  assert.equal(a.game.units.find(u => u.id === splitTarget.id).count, 4);

  // Free friendly move + paid neutral/hostile move + private orders
  const u = a.game.units.find(
    v =>
      v.owner === a.session.player &&
      a.game.territories[v.territory].neighbors.some(n => a.game.territories[n].owner !== a.session.player),
  );
  assert(u);
  const target = a.game.territories[u.territory].neighbors.find(n => a.game.territories[n].owner !== a.session.player);
  const friendly = a.game.units.find(
    v =>
      v.owner === a.session.player &&
      v.id !== u.id &&
      a.game.territories[v.territory].neighbors.some(n => a.game.territories[n].owner === a.session.player),
  );
  if (friendly) {
    const fTarget = a.game.territories[friendly.territory].neighbors.find(
      n => a.game.territories[n].owner === a.session.player,
    );
    await a.command('order', { unit: friendly.id, kind: 'move', target: fTarget });
    assert.equal(a.game.reserved, 0);
  }
  const beforeCurrency = a.game.players.find(p => p.id === a.session.player).resources.currency;
  const movePrice = u.count * 1;
  await a.command('order', { unit: u.id, kind: 'move', target });
  assert.equal(a.game.reserved, movePrice);
  assert.equal(Object.keys(b.game.orders).length, 0);
  await assert.rejects(() => b.command('order', { unit: u.id, kind: 'move', target }), /own army/);

  // Simulate phase: 1-hop LoS timeline & events
  await a.command('ready');
  await b.command('ready');
  await Promise.all([a.wait(() => a.game.phase === 'simulate'), b.wait(() => b.game.phase === 'simulate')]);
  assert.equal(a.game.players.find(p => p.id === a.session.player).resources.currency, beforeCurrency - movePrice);
  assert(a.game.timeline.length >= 5);
  assert.equal(a.game.timeline[0].units.find(v => v.id === u.id).territory, u.territory);
  assert.deepEqual(a.game.timeline.at(-1).units, a.game.units);
  assert(a.game.simulationDuration >= a.game.timeline.length * 3000);
  // Distant enemy units (Bravo's opposite corner) must not leak into Alpha's Simulate view
  assert(
    a.game.units.every(v => v.owner === a.session.player || a.game.territories[v.territory].visible),
    'Simulate leaked unit outside 1-hop LoS',
  );

  // Gather phase: untimed lock-in, Capital HQ income (+4 currency, +2 humans), lastEvents, recruit, feed, dedup
  const preGatherCurrency = a.game.players.find(p => p.id === a.session.player).resources.currency;
  const ownedBeforeGather = a.game.territories.filter(t => t.owner === a.session.player).length;
  await a.command('ready');
  await b.command('ready');
  await Promise.all([a.wait(() => a.game.phase === 'gather'), b.wait(() => b.game.phase === 'gather')]);
  assert.equal(a.game.deadline, 0, 'Gather phase must have no deadline');
  assert.equal(a.game.simulationDuration, 0, 'simulationDuration must reset to 0 after simulate');
  const postGatherCurrency = a.game.players.find(p => p.id === a.session.player).resources.currency;
  assert.equal(
    postGatherCurrency - preGatherCurrency,
    ownedBeforeGather * 8 + 4,
    'Gather income must include +4 Capital HQ currency bonus',
  );
  assert(Array.isArray(a.game.lastEvents) && a.game.lastEvents.length > 0, 'Last round report missing in Gather');

  const gatherSplit = a.game.units.find(v => v.owner === a.session.player && v.count >= 2);
  await a.command('split', { unit: gatherSplit.id, count: 1 });

  const territory = a.game.territories.find(
    t => t.owner === a.session.player && a.game.units.filter(v => v.territory === t.id).length < 3,
  );
  assert(territory);
  await a.command('recruit', { type: 'tank', count: 1, target: territory.id });
  const tank = a.game.units.find(x => x.owner === a.session.player && x.type === 'tank');
  assert(tank);
  assert(!tank.fed);
  await a.command('feed', { unit: tank.id, ration: 1.5 });
  await assert.rejects(() => a.command('feed', { unit: tank.id, ration: 1 }), /not been fed/);
  await assert.rejects(() => a.command('split', { unit: tank.id, count: 1 }), /unfed|smaller/);

  const duplicateID = crypto.randomUUID();
  const body = JSON.stringify({ id: duplicateID, action: 'recruit', type: 'artillery', count: 1, target: territory.id });
  const count = a.game.units.length;
  a.ws.send(body);
  await a.wait(() => a.messages.some(m => m.id === duplicateID));
  a.ws.send(body);
  await new Promise(r => setTimeout(r, 150));
  assert.equal(a.game.units.length, count + 1);
  // Fill sector to 6 stacks and verify 7th stack is rejected by 6-stack cap
  while (a.game.units.filter(x => x.territory === territory.id).length < 6) {
    await a.command('recruit', { type: 'infantry', count: 1, target: territory.id });
  }
  await assert.rejects(
    () => a.command('recruit', { type: 'infantry', count: 1, target: territory.id }),
    /Sector limit reached \(6 armies\)/,
  );

  // 5-minute disconnect grace + reconnect
  a.close();
  await b.wait(() => b.game.players.find(p => p.id === a.session.player)?.disconnectedUntil > 0);
  const graceDeadline = b.game.players.find(p => p.id === a.session.player).disconnectedUntil;
  assert(Math.abs(graceDeadline - Date.now() - 300000) < 5000);
  reconnect = new Commander();
  await reconnect.connect();
  await reconnect.command('resume', { code: a.session.code, token: a.session.token });
  assert.equal(reconnect.session.player, a.session.player);
  assert.equal(reconnect.game.phase, 'gather');
  assert.equal(reconnect.game.players.find(p => p.id === a.session.player).disconnectedUntil, 0);
  assert(reconnect.game.units.every(v => v.owner === a.session.player || reconnect.game.territories[v.territory].visible));

  await reconnect.command('ready');
  await b.command('ready');
  await reconnect.wait(() => reconnect.game.phase === 'plan' && reconnect.game.round === 2);

  // Confirmed surrender & cleanup
  const oldSession = { ...b.session };
  await b.command('leave');
  await b.wait(() => b.messages.some(m => m.type === 'left'));
  await reconnect.wait(() => reconnect.game.phase === 'finished');
  assert(!reconnect.game.players.some(p => p.id === oldSession.player));
  assert(reconnect.game.territories.every(t => t.owner !== oldSession.player));
  assert(reconnect.game.units.every(v => v.owner !== oldSession.player));
  await b.command('resume', { code: oldSession.code, token: oldSession.token }).then(
    () => {
      throw Error('Surrendered seat resumed');
    },
    err => assert.match(err.message, /Session expired/),
  );
  await b.command('create', { name: 'New campaign' });
  assert.notEqual(b.session.code, oldSession.code);
  await reconnect.command('leave');
  await reconnect.wait(() => reconnect.messages.some(m => m.type === 'left'));

  console.log(
    'PASS: two-player lobby, repeating draft, concealed timed capitals, 1-hop LoS across Plan/Simulate/Gather, Army Split, Capital HQ income, Last Round report, untimed Gather, deduplication, 5m reconnect grace, and surrender.',
  );
} finally {
  a.close();
  b.close();
  reconnect?.close();
}
