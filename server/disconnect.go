package main

import "time"

const disconnectGrace = 5 * time.Minute

func (h *hub) playerConnected(g *Game, id string) bool {
	for c := range h.clients {
		if c.game == g && c.player == id {
			return true
		}
	}
	return false
}

// Called under the hub lock after a socket has been removed.
func (h *hub) disconnected(c *client, now time.Time) {
	if c.game == nil || h.playerConnected(c.game, c.player) {
		return
	}
	p := c.game.player(c.player)
	if p == nil {
		return
	}
	if p.DisconnectedUntil == 0 {
		p.DisconnectedUntil = now.Add(disconnectGrace).UnixMilli()
	}
	h.broadcast(c.game)
}

// Room lifetime follows player seats, never time since the last command.
func (h *hub) expireDisconnected(now time.Time) {
	for code, g := range h.games {
		players := append([]*Player{}, g.Players...)
		for _, p := range players {
			if p.DisconnectedUntil == 0 {
				continue
			}
			if h.playerConnected(g, p.ID) {
				p.DisconnectedUntil = 0
				continue
			}
			if now.UnixMilli() < p.DisconnectedUntil {
				continue
			}
			if g.player(p.ID) != nil {
				_ = g.leave(p.ID)
				h.broadcast(g)
			}
		}
		if len(g.Players) == 0 {
			delete(h.games, code)
		}
	}
}
