# Iron Table — Alpha 0.1

A browser multiplayer tabletop strategy alpha. Go owns every match, React renders an interactive SVG battlefield, and standard WebSockets connect up to four commanders.

## Run

Windows: `powershell -ExecutionPolicy Bypass -File ./start.ps1`

Requires Node.js 22+ and Go 1.24+. The workspace-local Go installation at `.tools/go` is used if present. Open http://localhost:8080. Create a lobby and share its code. Other players must open the **same server URL**, then enter that code. On a LAN, use the host computer's address and allow the port through your firewall if needed. Internet play requires deployment to a persistent Go-capable host with HTTPS/WebSocket support; this alpha is not publicly deployed.

For frontend development, run `npm run dev` alongside the Go server. Vite proxies `/ws` to port 8080. For browser-only asset builds, run `npm run build`.

## Play

Create a room, invite another player or add bot commanders, and start. Server dice determine draft order. Claim territories in snake order (`A → B → C → D → D → C → B → A`). Each claimed territory starts with four infantry.

- **Plan (90s):** select an army and an adjacent destination. Friendly-sector transfers are free; attacks/neutral advances spend currency. Artillery defaults to Bombard across one border (firing after movement). Hold & defend entrenches the army (+20% direct combat, −25% bombardment taken). You can also split stacks (up to 4 stacks per sector). Lock when done; everyone ready ends the phase early.
- **Simulate (at least 12s, 3s per frame):** all orders resolve within 1-hop Line of Sight. A playback scrubber lets you pause or step through frames while field dispatches highlight each engagement. All active commanders can lock in to skip the remaining playback.
- **Gather (120s):** territory income plus Capital HQ bonuses (+4 currency, +2 humans while held) are credited at entry. Recruit, reinforce, split, merge matching unfed armies (max 4 stacks per sector), and allocate rations. Review the Last Round report tab anytime. Unfed armies auto-feed when all commanders lock in or the timer expires.
- Highest **Victory Points (`Territories + 2 × Capitals Controlled`)** after 20 rounds wins; ties break on territories then capitals. Owning the whole map also ends the campaign.

Lock-in buttons and the active commander readiness count appear beside the timer in the sticky top bar. Eliminated commanders (no territory and no surviving armies) do not count toward phase readiness. Plan retains its 90-second limit but advances immediately when all active commanders lock in.

## Alpha rule defaults

These fill decisions that were still open in the design conversation; they are adjustable, not final balance commitments.

| Rule | Default |
| --- | --- |
| Map | 24 connected, hand-shaped quadrilateral sectors on a shared vertex mesh |
| Production | Each sector yields 3 humans + 8 currency and fixed-at-match-start random 6–12 food or iron |
| Starting stock | 16 humans / 40 food / 16 iron / 60 currency |
| Infantry | 1 human + 4 currency per troop (10 HP) |
| Artillery | 2 humans + 4 currency + 4 iron per piece (14 HP) |
| Tank | 3 humans + 8 currency + 8 iron per piece (20 HP) |
| Movement | One adjacent territory; free between friendly sectors; neutral/enemy advance costs infantry 1, artillery 2, tank 3 currency per unit |
| Artillery | Adjacent bombardment resolving after movement, no movement that round, cannot capture at range |
| Hold & defend | Explicit entrenchment order: +20% direct combat strength and −25% incoming bombardment damage |
| Normal food | 1 food per troop/piece |
| Minimum food | 0.5 per troop/piece, rounded up; 65% attack |
| Extra food | 1.5 per troop/piece, rounded up; +20 health, capped at 100 |
| Starvation | -25 health (minimum 1), 30% attack |
| Merging & Splitting | Merge same owner/type/territory (both unfed, weighted health); split in Gather (unfed) or Plan (clears order); max 4 stacks per sector |
| Reinforcement | Same resource cost per added troop/piece as recruitment; troop-weighted health (new troops enter at 100); up to 4 armies per sector |
| Capitals | 45s concealed pick; holding your own capital grants +4 currency, +2 humans, and +15% direct defense in destination battles; each controlled capital awards +2 VP |
| Captured production | Available in the immediately following Gather |

Combined-arms combat is deterministic: troop count × base attack × health fraction × ration modifier × matchup × entrenchment. Base attack is infantry 2, tank 6, ranged artillery 4, direct artillery 1. Tanks receive ×1.4 against infantry; infantry/tanks receive ×1.5 against artillery; ranged artillery receives ×1.5 against tanks and ×1.2 against infantry. Damage is HP-weighted across opposing stacks and applied simultaneously within each encounter; each 10 / 14 / 20 damage kills one infantry / artillery / tank and scales stack health proportionally. Border-crossing opponents clash and hold their origins. Border clashes resolve first, then movement, then artillery bombardment (allowing backline artillery to shell incoming attackers; exposed artillery overrun by enemies in Stage 2 is suppressed in melee), then destination battles and retreats. Destination losers retreat after all ownership resolves, or are removed if no friendly neighboring destination exists. In a tie, an incumbent defending garrison holds its sector; tied attackers in an unowned or third-party sector retreat and ownership stays unchanged. Campaigns end immediately after Simulate when round 20 completes or one commander holds the entire continent.

## Architecture and limits

- `server/game.go`: authoritative rules, generation, commands, timers, and combat.
- `server/bot.go`: scored snake drafting, capital selection, combined-arms Gather management, and threat-aware Plan orders.
- `server/fog.go`: 1-hop Line-of-Sight visibility filtering across Plan, Simulate, and Gather, plus Last Round report filtering.
- `server/main.go`: HTTP/WebSocket lobby server and player-specific state filtering.
- `src/`: modular React interface (`src/App.tsx`, `src/combat.ts`, `src/types.ts`, `src/components/*`), SVG territory/army assets, orders, playback scrubber, and recruitment/feeding controls.
- Match state is **in memory**. Server restart loses lobbies; SQLite persistence is deferred.
- Reconnect uses a private token in browser session storage. Refresh restores the same seat while the server remains running. A tab has its own session; use separate browsers/devices to play multiple humans.
- Accepted command IDs are deduplicated within a connection. The client does not automatically replay mutations across reconnects; it requests a fresh authoritative state.
- Server serializes mutations under a mutex. Independent lobby actors and horizontal scaling are deferred.
- 1-hop Line of Sight applies across Plan, Simulate, and Gather: static terrain yields stay visible everywhere, while enemy ownership, capitals, units, orders, and simulation dispatches are only transmitted for sectors within 1 border of your territories or armies. Eliminated spectators and finished campaigns receive full visibility.
- Bot commanders score contiguous/balanced drafts, recruit combined arms with currency and food reserves, heal wounded stacks, bombard armor/threats, entrench outnumbered fronts, and push rear armies forward.
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

Army cards show current direct combat strength, matchup values, entrenchment status, and separate ranged values for artillery. Health, troop count, and ration changes update these values immediately. During Simulate, the map and battle inspector use immutable server snapshots in resolution order: initial armies, simultaneous border clashes, movement, simultaneous bombardment, individual territory resolutions, then retreats/final ownership. Playback lasts at least 12 seconds, with three seconds per snapshot, and can still be skipped by all active commanders locking in. Hidden phases never transmit replay snapshots.

## Four-territory setup and capitals

The server reveals one commander’s dice result every two seconds, rerolling ties. Highest roll drafts first. Players claim four tiles each in snake order (`A → B → C → D → D → C → B → A`); remaining tiles are neutral, yield no player income until captured, and have no garrisons. Each commander then confirms one of their own four tiles as capital within 45 seconds (rival picks stay concealed until Round 1 starts; bots and AFK timeouts pick their most sheltered tile automatically). Holding your own capital yields +4 currency and +2 humans each Gather and grants +15% direct combat defense in destination battles (+38% stacked with Hold & defend). Each controlled capital (yours or captured) adds +2 VP to your final score. Starting infantry is placed only on drafted tiles. Normal movement into a neutral tile captures it during Simulate, subject to competing arrivals. Matches run for 20 rounds.

## Integrated gameplay animations

The eight approved effects run in gameplay: interpolated army movement, artillery arcs/recoil flash, simultaneous tank and infantry volleys, impact-timed health/casualties, retreat routes, territory color/flag capture, and Gather production traveling to the player’s capital stockpile. Combat calculations remain server-authoritative. Use Reduced motion during Simulate for stepped movement and simpler impacts. Recruitment and merging have no animations. Capital stockpile travel is a visual presentation of global resources, not a new supply restriction; if the capital is occupied, income still arrives but the capital delivery animation is omitted.

## Exit round / leave lobby

The header exit button opens a confirmation. Confirming surrenders the entire match: the server removes the player and capital, dissolves all their armies, cancels their orders, and neutralizes their territory. Remaining players are notified immediately through their next state; the departed player returns to room creation and cannot resume the surrendered seat. Temporary WebSocket disconnects and refreshes preserve the seat for reconnect. Host control transfers to a remaining human where possible. Departures during setup cannot block rolls, drafting, or capital selection; if fewer than two players remain in setup, it resets to the lobby. During a match the last active commander wins. A departure during Simulate replaces playback with a fresh battlefield snapshot to avoid showing dissolved units again.

## Disconnect grace period

Idle-command room eviction is disabled. Losing a WebSocket connection starts a five-minute server-side timer for that player; reconnecting to the same seat cancels it. Remaining commanders see the offline countdown. At expiry the server removes only the disconnected player using surrender rules: dissolve armies, cancel orders, neutralize territories, remove the capital and readiness requirement. A room with a connected human is never disbanded for inactivity. When all human seats have expired or explicitly exited, the empty/bot-only room is removed. Server restarts still reset in-memory matches.
