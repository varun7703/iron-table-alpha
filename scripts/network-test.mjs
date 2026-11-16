import assert from 'node:assert/strict';
const base=process.env.GAME_URL||'http://127.0.0.1:8080';
class Commander {
 constructor(){this.ws=new WebSocket(base.replace('http','ws')+'/ws');this.messages=[];this.listeners=[];this.ws.addEventListener('message',e=>{const m=JSON.parse(e.data);this.messages.push(m);if(m.type==='state')this.game=m.game;if(m.type==='session')this.session=m;for(const f of this.listeners)f()})}
 async connect(){await new Promise((resolve,reject)=>{this.ws.addEventListener('open',resolve,{once:true});this.ws.addEventListener('error',reject,{once:true})})}
 wait(test,timeout=5000){if(test())return Promise.resolve();return new Promise((resolve,reject)=>{const timer=setTimeout(()=>{this.listeners=this.listeners.filter(x=>x!==check);reject(new Error('Timed out waiting for server state'))},timeout);const check=()=>{if(test()){clearTimeout(timer);this.listeners=this.listeners.filter(x=>x!==check);resolve()}};this.listeners.push(check)})}
 async command(action,data={}){const id=crypto.randomUUID();this.ws.send(JSON.stringify({id,action,...data}));await this.wait(()=>this.messages.some(m=>m.id===id));const error=this.messages.find(m=>m.type==='error'&&m.id===id);if(error)throw Error(error.message);return id}
 close(){this.ws.close()}
}
const a=new Commander(),b=new Commander();let reconnect;
try{
 await Promise.all([a.connect(),b.connect()]);await a.command('create',{name:'Network Alpha'});await b.command('join',{name:'Network Bravo',code:a.session.code});await a.wait(()=>a.game.players.length===2);assert.equal(b.game.players.length,2);
 await a.command('start');await b.wait(()=>b.game.phase==='draft');
 for(let i=0;i<24;i++){const current=a.game.turn===a.session.player?a:b;await current.wait(()=>current.game.phase==='draft');const t=current.game.territories.find(t=>!t.owner);await current.command('claim',{target:t.id});await Promise.all([a.wait(()=>a.game.phase!=='draft'||a.game.turn!==current.session.player),b.wait(()=>b.game.phase!=='draft'||b.game.turn!==current.session.player)])}
 await Promise.all([a.wait(()=>a.game.phase==='plan'),b.wait(()=>b.game.phase==='plan')]);assert(a.game.units.every(u=>u.owner===a.session.player));assert(b.game.units.every(u=>u.owner===b.session.player));assert.equal(a.game.units.length,12);
 const u=a.game.units[0],target=a.game.territories[u.territory].neighbors[0],before=a.game.players.find(p=>p.id===a.session.player).resources.currency;
 await a.command('order',{unit:u.id,kind:'move',target});assert.equal(a.game.reserved,4);assert.equal(Object.keys(b.game.orders).length,0);
 await assert.rejects(()=>b.command('order',{unit:u.id,kind:'move',target}),/own army/);
 await a.command('ready');await b.command('ready');await a.wait(()=>a.game.phase==='simulate');assert(a.game.units.some(u=>u.owner===b.session.player));assert.equal(a.game.players.find(p=>p.id===a.session.player).resources.currency,before-4);
 await a.wait(()=>a.game.phase==='gather',18000);await b.wait(()=>b.game.phase==='gather');const territory=a.game.territories.find(t=>t.owner===a.session.player);await a.command('recruit',{type:'tank',count:1,target:territory.id});const tank=a.game.units.find(u=>u.type==='tank');assert(tank);assert(!tank.fed);await a.command('feed',{unit:tank.id,ration:1.5});await assert.rejects(()=>a.command('feed',{unit:tank.id,ration:1}),/not been fed/);
 const duplicateID=crypto.randomUUID(),body=JSON.stringify({id:duplicateID,action:'recruit',type:'artillery',count:1,target:territory.id});const count=a.game.units.length;a.ws.send(body);await a.wait(()=>a.messages.some(m=>m.id===duplicateID));a.ws.send(body);await new Promise(r=>setTimeout(r,150));assert.equal(a.game.units.length,count+1);
 reconnect=new Commander();await reconnect.connect();await reconnect.command('resume',{code:a.session.code,token:a.session.token});assert.equal(reconnect.session.player,a.session.player);assert.equal(reconnect.game.phase,'gather');assert(reconnect.game.units.every(u=>u.owner===a.session.player));
 await reconnect.command('ready');await b.command('ready');await reconnect.wait(()=>reconnect.game.phase==='plan'&&reconnect.game.round===2);
 console.log('PASS: two-player lobby, 24-pick draft, private orders, ownership validation, simultaneous simulation, currency charge, gather production, recruitment, food, duplicate command protection, reconnect, and round progression.');
}finally{a.close();b.close();reconnect?.close()}
