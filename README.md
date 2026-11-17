# Iron Table — Alpha 0.1

A browser multiplayer tabletop strategy alpha. Go owns every match, React renders an interactive SVG battlefield, and standard WebSockets connect up to four commanders.

## Run

Windows: `powershell -ExecutionPolicy Bypass -File ./start.ps1`

Requires Node.js 22+ and Go 1.24+. The workspace-local Go installation at `.tools/go` is used if present. Open http://localhost:8080. Create a lobby and share its code. Other players must open the **same server URL**, then enter that code. On a LAN, use the host computer's address and allow the port through your firewall if needed. Internet play requires deployment to a persistent Go-capable host with HTTPS/WebSocket support; this alpha is not publicly deployed.

For frontend development, run `npm run dev` alongside the Go server. Vite proxies `/ws` to port 8080. For browser-only asset builds, run `npm run build`.

## Play

Create a room, invite another player or add bot commanders, and start. Server dice determine draft order. Claim territories in repeating A → B → C → D order. Each claimed territory starts with four infantry.

- **Plan (90s):** select an army and an adjacent destination. Move/attack or, for artillery, bombard. Hold/defend is available. Lock when done; everyone ready ends the phase early.
- **Simulate (up to 12s):** all orders resolve and the full map is revealed. A playback progress bar reveals field dispatches and highlights the current engagement. All active commanders can lock in to skip the remaining playback.
- **Gather (no time limit):** territory income is credited at entry. Recruit, reinforce, merge matching unfed armies, and allocate rations. Unfed armies receive automatic normal/minimum rations on lock when affordable. Every active commander must lock in to advance the round; there is no Gather deadline.
- Most territories after 20 rounds wins; ties are shared. Owning the whole map also ends the campaign.

Lock-in buttons and the active commander readiness count appear beside the timer in the sticky top bar. Eliminated commanders (no territory and no surviving armies) do not count toward phase readiness. Plan retains its 90-second limit but advances immediately when all active commanders lock in.

## Alpha rule defaults

These fill decisions that were still open in the design conversation; they are adjustable, not final balance commitments.

| Rule | Default |
| --- | --- |
| Map | 24 connected, hand-shaped quadrilateral sectors on a shared vertex mesh |
| Production | Each sector yields 3 humans + 8 currency and fixed-at-match-start random 6–12 food or iron |
| Starting stock | 40 humans / 120 food / 40 iron / 160 currency |
| Infantry | 1 human + 4 currency per troop |
| Artillery | 1 human + 2 currency + 4 iron per piece |
| Tank | 1 human + 4 currency + 8 iron per piece |
| Movement | One adjacent territory; currency cost per troop/piece is infantry 1, artillery 2, tank 3 |
| Artillery | Adjacent bombardment, no movement that round, cannot capture at range |
| Normal food | 1 food per troop/piece |
| Minimum food | 0.5 per troop/piece, rounded up; 65% attack |
| Extra food | 1.5 per troop/piece, rounded up; +20 health, capped at 100 |
| Starvation | -25 health (minimum 1), 30% attack |
| Merging | Same owner/type/territory, both unfed; troop-weighted health |
| Reinforcement | Same resource cost per added troop/piece as recruitment; preserves stack health |
| Captured production | Available in the immediately following Gather |

Artillery and tank recruitment now emphasize iron: artillery costs 2 currency + 4 iron and tanks cost 4 currency + 8 iron, plus one human each. This replaces the earlier currency-based 4:2:1 cost target.

Combat is deterministic for the alpha: troop count × base attack × health fraction × ration modifier × matchup. Base attack is infantry 2, tank 7, ranged artillery 4, direct artillery 1. Tanks receive ×1.4 against infantry; infantry/tanks receive ×1.5 against artillery; ranged artillery receives ×1.5 against infantry and ×0.7 against tanks. Damage is applied simultaneously within each encounter; each 10 damage kills one troop and damage also reduces stack health. Border-crossing opponents clash and hold their origins. Ranged fire resolves first, then border clashes, movements, and destination battles. Destination losers retreat after all ownership resolves, or are removed if no friendly neighboring destination exists. Tied forces retreat and ownership stays unchanged.

## Architecture and limits

- `server/game.go`: authoritative rules, generation, commands, timers, and combat.
- `server/main.go`: HTTP/WebSocket lobby server and player-specific state filtering.
- `src/main.tsx`: React interface, SVG territory/army assets, orders, recruitment and feeding.
- Match state is **in memory**. Server restart loses lobbies; SQLite persistence is deferred.
- Reconnect uses a private token in browser session storage. Refresh restores the same seat while the server remains running. A tab has its own session; use separate browsers/devices to play multiple humans.
- Accepted command IDs are deduplicated within a connection. The client does not automatically replay mutations across reconnects; it requests a fresh authoritative state.
- Server serializes mutations under a mutex. Independent lobby actors and horizontal scaling are deferred.
- During Plan/Gather enemy ownership, units, orders and yields are omitted; geography remains visible as targetable outlines. Full information is revealed during Simulate. Previously revealed information cannot be made secret again.
- Bot commanders draft, attack adjacent enemies and auto-feed. They do not yet recruit intelligently.
- No accounts, diplomacy, public matchmaking, map editor, audio, or database persistence yet.
- Visual assets are original code-generated SVG/CSS. Fonts request Google Fonts with local fallbacks.

## Verify

`npm run build`

`go test ./...` from `server/` (or use `.tools/go/bin/go.exe -C server test ./...` with workspace-local `GOPATH` and `GOCACHE`).

With the server running: `npm run test:network`. This drives two real WebSocket clients through drafting, private orders, simulation, gathering, recruitment, food, deduplication and reconnect.

## Deploy both services on Render

Use the same GitHub repository for two separate services. Deploy the backend first, then the frontend.

### 1. Backend — Web Service

- Repository: `varun7703/iron-table-alpha`, branch `main`.
- Name: `iron-table-server` (or another available name).
- Language/runtime: **Go**.
- Root directory: `server`.
- Build command: `go build -tags netgo -ldflags '-s -w' -o app .`
- Start command: `./app`
- Health check path: `/health`.
- Keep one instance: matches are held in memory and cannot span instances.
- Deploy, then copy the actual HTTPS service URL. Visiting `/health` should return `{"engine":"go","status":"ok"}` (key order may vary).

The backend reads Render's `PORT` automatically. There is no need to set it yourself. `render.yaml` also supplies these backend settings for Blueprint deployment, but the manual Web Service form works independently.

### 2. Frontend — Static Site

- Create **New → Static Site** and select the same repository and `main` branch.
- Name: `iron-table` (or another available name).
- Root directory: leave blank (repository root).
- Build command: `npm ci && npm run build`
- Publish directory: `dist`.
- Environment variable `NODE_VERSION`: `24`.
- Environment variable `VITE_GAME_SERVER_URL`: the backend's actual HTTPS URL, for example `https://YOUR-BACKEND.onrender.com`.
- Deploy, then copy the frontend's actual HTTPS URL.

### 3. Allow the frontend to connect

In the backend's Environment settings, set `ALLOWED_ORIGINS` to the frontend's exact origin, for example `https://YOUR-FRONTEND.onrender.com`, with **no trailing slash**. Save and redeploy the backend. Multiple explicitly allowed origins may be comma-separated.

Open the frontend, confirm **COMMAND LINK ONLINE**, create a lobby, and have another browser/device join using the room code. The browser automatically connects to the backend using `wss://…/ws`. Do not use the backend URL as the playable website.

The `VITE_GAME_SERVER_URL` value is included at frontend build time. If it changes, redeploy/rebuild the Static Site. The `netlify.toml` file is only for optional Netlify hosting and is ignored by Render.

### Alpha hosting limits

Restarting, redeploying, or replacing the backend loses active matches because persistence is not implemented. Free Render web services can spin down after inactivity; the first connection may need time to wake the backend. The client retries with backoff. Use a paid always-on instance when uninterrupted sessions matter, but in-memory matches still reset on backend replacement. No paid plan is required by this repository's configuration.

## Combat inspector and replay

Army cards show current direct combat strength, matchup values, and separate ranged values for artillery. Health, troop count, and ration changes update these values immediately. During Simulate, the map and battle inspector use immutable server snapshots in resolution order: initial armies, simultaneous bombardment, simultaneous border clashes, movement, individual territory resolutions, then retreats/final ownership. Playback lasts at least 12 seconds, with three seconds per snapshot, and can still be skipped by all active commanders locking in. Hidden phases never transmit replay snapshots.

## Four-territory setup and capitals

The server reveals one commander’s dice result every two seconds, rerolling ties. Highest roll drafts first. Players claim four tiles each in repeating order; remaining tiles are neutral, yield no player income until captured, and have no garrisons. Each commander then confirms one of their own four tiles as capital. Bots choose automatically. Capital selection has no timer. Starting infantry is placed only on drafted tiles. Capitals are markers without special income, defense, or elimination rules yet. Normal movement into a neutral tile captures it during Simulate, subject to competing arrivals. Matches run for 20 rounds.
