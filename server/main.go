package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type client struct {
	conn   *websocket.Conn
	game   *Game
	player string
	out    chan any
	seen   map[string]bool
}
type hub struct {
	mu      sync.Mutex
	games   map[string]*Game
	clients map[*client]bool
}

func (g *Game) view(id string) map[string]any {
	full := g.fullVisibility(id)
	vis := g.visibleSectors(id)
	if g.Phase == "simulate" {
		vis = g.simulateVisibleSectors(id)
	}
	ts := make([]Territory, len(g.Territories))
	copy(ts, g.Territories)
	for i := range ts {
		ts[i].Visible = vis[i]
		if vis[i] {
			ts[i].Neutral = ts[i].Owner == ""
			ts[i].IsCapital = g.isCapitalSector(i) && !(g.Phase == "capital" && ts[i].Owner != id)
		} else {
			ts[i].Neutral = false
			ts[i].Owner = ""
			ts[i].IsCapital = false
		}
	}
	us := []*Unit{}
	for _, u := range g.Units {
		if u.Owner == id || vis[u.Territory] {
			copyUnit := *u
			us = append(us, &copyUnit)
		}
	}
	ps := []Player{}
	for _, p := range g.Players {
		cp := *p
		cp.Eliminated = g.eliminated(p.ID)
		if p.ID != id {
			cp.Resources = Resources{}
			if g.Phase == "capital" || (cp.Capital >= 0 && !vis[cp.Capital]) {
				cp.Capital = -1
			}
		}
		if p.ID == id || full {
			vp, _, _ := g.playerScore(p.ID)
			cp.VP = vp
		}
		ps = append(ps, cp)
	}
	orders := map[string]Order{}
	if g.Phase == "simulate" {
		orders = g.filterSimulateOrders(id, vis)
	} else {
		for key, o := range g.Orders {
			u := g.unit(key)
			if u != nil && u.Owner == id {
				orders[key] = o
			}
		}
	}
	events := []Event{}
	timeline := []BattleFrame{}
	if g.Phase == "simulate" {
		events = filterEvents(g.Events, vis)
		timeline = g.filterTimeline(id, vis)
	} else if g.Phase == "finished" {
		events = append([]Event{}, g.Events...)
	}
	lastEvents := []Event{}
	if (g.Phase == "gather" || g.Phase == "plan") && len(g.LastEvents) > 0 {
		if full {
			lastEvents = append([]Event{}, g.LastEvents...)
		} else if lv, ok := g.LastSimVis[id]; ok {
			lastEvents = filterEvents(g.LastEvents, lv)
		}
	}
	turn := ""
	if g.Phase == "draft" {
		if dp := g.draftPlayer(); dp != nil {
			turn = dp.ID
		}
	}
	simDuration := 0
	if g.Phase == "simulate" {
		simDuration = max(12, len(timeline)*3) * 1000
	}
	return map[string]any{
		"rollIndex":          g.RollIndex,
		"timeline":           timeline,
		"simulationDuration": simDuration,
		"code":               g.Code,
		"host":               g.Host,
		"phase":              g.Phase,
		"round":              g.Round,
		"deadline":           g.Deadline,
		"players":            ps,
		"territories":        ts,
		"units":              us,
		"orders":             orders,
		"events":             events,
		"lastEvents":         lastEvents,
		"turn":               turn,
		"winner":             g.Winner,
		"reserved":           g.reserved(id, ""),
	}
}
func (h *hub) send(c *client, v any) {
	select {
	case c.out <- v:
	default:
		go c.conn.Close(websocket.StatusPolicyViolation, "Client too slow")
	}
}
func (h *hub) broadcast(g *Game) {
	for c := range h.clients {
		if c.game == g {
			h.send(c, map[string]any{"type": "state", "game": g.view(c.player)})
		}
	}
}
func (h *hub) handle(c *client, cmd Command) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cmd.ID == "" || len(cmd.ID) > 100 {
		h.send(c, map[string]any{"type": "error", "message": "Command ID required"})
		return
	}
	if c.seen[cmd.ID] {
		h.send(c, map[string]any{"type": "ack", "id": cmd.ID})
		return
	}
	if len(c.seen) > 10000 {
		c.seen = map[string]bool{}
	}
	c.seen[cmd.ID] = true
	var err error
	if cmd.Action == "leave" && c.game != nil {
		g, id := c.game, c.player
		if err := g.leave(id); err != nil {
			h.send(c, map[string]any{"type": "error", "message": err.Error(), "id": cmd.ID})
			return
		}

		for peer := range h.clients {
			if peer.game == g && peer.player == id {
				peer.game = nil
				peer.player = ""
				h.send(peer, map[string]any{"type": "left"})
			}
		}
		h.broadcast(g)
		if len(g.Players) == 0 {
			delete(h.games, g.Code)
		}
		h.send(c, map[string]any{"type": "ack", "id": cmd.ID})
		return
	}
	if c.game == nil {
		switch cmd.Action {
		case "create", "join", "resume":
			var g *Game
			var p *Player
			if cmd.Action == "create" {
				name := strings.TrimSpace(cmd.Name)
				if len(name) < 1 || len(name) > 24 {
					err = fmt.Errorf("Enter a name of 1–24 characters")
				} else {
					code := strings.ToUpper(uid()[:6])
					for h.games[code] != nil {
						code = strings.ToUpper(uid()[:6])
					}
					g = newGame(code)
					p = g.addPlayer(name, false)
					h.games[code] = g
				}
			} else {
				g = h.games[strings.ToUpper(strings.TrimSpace(cmd.Code))]
				if g == nil {
					err = fmt.Errorf("Lobby not found")
				} else if cmd.Action == "resume" {
					for _, candidate := range g.Players {
						if candidate.Token == cmd.Token && cmd.Token != "" {
							p = candidate
						}
					}
					if p == nil {
						err = fmt.Errorf("Session expired")
					}
				} else if g.Phase != "lobby" || len(g.Players) >= 4 {
					err = fmt.Errorf("Lobby is full or match has started")
				} else {
					name := strings.TrimSpace(cmd.Name)
					if len(name) < 1 || len(name) > 24 {
						err = fmt.Errorf("Enter a name of 1–24 characters")
					} else {
						p = g.addPlayer(name, false)
					}
				}
			}
			if err == nil {
				for old := range h.clients {
					if old != c && old.game == g && old.player == p.ID {
						old.game = nil
						go old.conn.Close(websocket.StatusNormalClosure, "Session opened elsewhere")
					}
				}
				c.game = g
				c.player = p.ID
				p.DisconnectedUntil = 0

				h.send(c, map[string]any{"type": "session", "code": g.Code, "player": p.ID, "token": p.Token})
				h.broadcast(g)
			}
		default:
			err = fmt.Errorf("Join a lobby first")
		}
	} else {
		err = c.game.command(c.player, cmd)
		if err == nil {

			h.broadcast(c.game)
		}
	}
	if err != nil {
		h.send(c, map[string]any{"type": "error", "message": err.Error(), "id": cmd.ID})
	} else {
		h.send(c, map[string]any{"type": "ack", "id": cmd.ID})
	}
}
func (h *hub) serveWS(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r, os.Getenv("ALLOWED_ORIGINS")) {
		http.Error(w, "Origin is not allowed", http.StatusForbidden)
		return
	}
	// Host-only patterns are derived from the exact origin already validated above.
	options := &websocket.AcceptOptions{}
	if origin, err := url.Parse(r.Header.Get("Origin")); err == nil && origin.Host != "" {
		options.OriginPatterns = []string{origin.Host}
	}
	conn, err := websocket.Accept(w, r, options)
	if err != nil {
		return
	}
	conn.SetReadLimit(8192)
	c := &client{conn: conn, out: make(chan any, 64), seen: map[string]bool{}}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer conn.CloseNow()
	defer func() { h.mu.Lock(); delete(h.clients, c); h.disconnected(c, time.Now()); h.mu.Unlock() }()
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingCtx, done := context.WithTimeout(ctx, 10*time.Second)
				err := conn.Ping(pingCtx)
				done()
				if err != nil {
					cancel()
					conn.CloseNow()
					return
				}
			}
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case v := <-c.out:
				wctx, done := context.WithTimeout(ctx, 5*time.Second)
				err := wsjson.Write(wctx, conn, v)
				done()
				if err != nil {
					cancel()
					conn.CloseNow()
					return
				}
			}
		}
	}()
	for {
		var cmd Command
		if err := wsjson.Read(ctx, conn, &cmd); err != nil {
			return
		}
		h.handle(c, cmd)
	}
}
func main() {
	h := &hub{games: map[string]*Game{}, clients: map[*client]bool{}}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			h.mu.Lock()
			now := time.Now()
			h.expireDisconnected(now)
			for _, g := range h.games {
				phase, draft, roll := g.Phase, g.Draft, g.RollIndex
				g.tick()
				if g.Phase != phase || g.Draft != draft || g.RollIndex != roll {

					h.broadcast(g)
				}
			}
			h.mu.Unlock()
		}
	}()
	http.HandleFunc("/ws", h.serveWS)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "engine": "go"})
	})
	if _, err := os.Stat("dist/index.html"); err == nil {
		http.Handle("/", http.FileServer(http.Dir("dist")))
	} else {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"service": "Iron Table game server", "health": "/health", "websocket": "/ws"})
		})
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Iron Table listening at http://localhost:%s", port)
	server := &http.Server{Addr: ":" + port, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second}
	log.Fatal(server.ListenAndServe())
}

// Same-origin remains available locally; split hosting requires explicit HTTPS origins.
// Non-browser clients may omit Origin. Tokens and ownership checks still apply.
func originAllowed(r *http.Request, configured string) bool {
	raw := r.Header.Get("Origin")
	if raw == "" {
		return true
	}
	origin, err := url.Parse(raw)
	if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" {
		return false
	}
	if strings.EqualFold(origin.Host, r.Host) {
		return true
	}
	for _, allowed := range strings.Split(configured, ",") {
		if strings.EqualFold(strings.TrimSpace(allowed), origin.Scheme+"://"+origin.Host) {
			return true
		}
	}
	return false
}
