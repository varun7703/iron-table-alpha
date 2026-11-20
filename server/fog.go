package main

import "strings"

func (g *Game) fullVisibility(id string) bool {
	if g.Phase == "lobby" || g.Phase == "roll" || g.Phase == "draft" || g.Phase == "capital" || g.Phase == "finished" {
		return true
	}
	if id == "" || g.eliminated(id) {
		return true
	}
	return false
}

func (g *Game) markOneHop(vis map[int]bool, tid int) {
	if tid < 0 || tid >= len(g.Territories) {
		return
	}
	vis[tid] = true
	for _, n := range g.Territories[tid].Neighbors {
		vis[n] = true
	}
}

// visibleSectors returns the set of territory IDs currently within 1-hop LoS
// of player id (owned sectors, sectors containing living units of id, and all
// immediate neighbors).
func (g *Game) visibleSectors(id string) map[int]bool {
	vis := make(map[int]bool, len(g.Territories))
	if g.fullVisibility(id) {
		for i := range g.Territories {
			vis[i] = true
		}
		return vis
	}
	for _, t := range g.Territories {
		if t.Owner == id {
			g.markOneHop(vis, t.ID)
		}
	}
	for _, u := range g.Units {
		if u.Owner == id && u.Count > 0 {
			g.markOneHop(vis, u.Territory)
		}
	}
	return vis
}

// simulateVisibleSectors computes the union of 1-hop LoS across all frames of
// the current simulation timeline so a commander sees every engagement and
// border around sectors they owned or fought in during resolution, without
// leaking distant battles across the continent.
func (g *Game) simulateVisibleSectors(id string) map[int]bool {
	vis := make(map[int]bool, len(g.Territories))
	if g.fullVisibility(id) {
		for i := range g.Territories {
			vis[i] = true
		}
		return vis
	}
	for _, f := range g.Timeline {
		for tid, owner := range f.Owners {
			if owner == id {
				g.markOneHop(vis, tid)
			}
		}
		for _, u := range f.Units {
			if u.Owner == id && u.Count > 0 {
				g.markOneHop(vis, u.Territory)
			}
		}
	}
	if len(g.Timeline) == 0 {
		return g.visibleSectors(id)
	}
	return vis
}

func eventVisible(e Event, vis map[int]bool) bool {
	if e.From == e.To {
		return e.To >= 0 && vis[e.To]
	}
	return e.From >= 0 && e.To >= 0 && vis[e.From] && vis[e.To]
}

func filterEvents(events []Event, vis map[int]bool) []Event {
	out := make([]Event, 0, len(events))
	for _, e := range events {
		if eventVisible(e, vis) {
			out = append(out, e)
		}
	}
	return out
}

// filterTimeline returns a player-scoped timeline with zero side-channel leaks:
// distant "Territory resolved · <Name>" frames outside vis are omitted entirely
// (no placeholder frames), and Destroyed lists only unit IDs that were visible
// in the player's immediately preceding frame and actually died.
func (g *Game) filterTimeline(id string, vis map[int]bool) []BattleFrame {
	out := make([]BattleFrame, 0, len(g.Timeline))
	const resolvedPrefix = "Territory resolved · "
	prevVisible := map[string]bool{}
	pendingDestroyed := map[string]bool{}
	for _, f := range g.Timeline {
		for _, deadID := range f.Destroyed {
			pendingDestroyed[deadID] = true
		}
		if strings.HasPrefix(f.Label, resolvedPrefix) {
			name := strings.TrimPrefix(f.Label, resolvedPrefix)
			hidden := false
			for _, t := range g.Territories {
				if t.Name == name && !vis[t.ID] {
					hidden = true
					break
				}
			}
			if hidden {
				continue
			}
		}
		owners := make([]string, len(f.Owners))
		for i, owner := range f.Owners {
			if vis[i] {
				owners[i] = owner
			}
		}
		units := make([]Unit, 0, len(f.Units))
		currVisible := make(map[string]bool, len(f.Units))
		for _, u := range f.Units {
			if u.Owner == id || vis[u.Territory] {
				units = append(units, u)
				currVisible[u.ID] = true
			}
		}
		destroyed := []string{}
		for deadID := range pendingDestroyed {
			if prevVisible[deadID] && !currVisible[deadID] {
				destroyed = append(destroyed, deadID)
			}
		}
		pendingDestroyed = map[string]bool{}
		prevVisible = currVisible
		out = append(out, BattleFrame{
			Label:     f.Label,
			Units:     units,
			Owners:    owners,
			Events:    filterEvents(f.Events, vis),
			Destroyed: destroyed,
		})
	}
	return out
}

func (g *Game) filterSimulateOrders(id string, vis map[int]bool) map[string]Order {
	orders := map[string]Order{}
	frame0Units := map[string]Unit{}
	if len(g.Timeline) > 0 {
		for _, u := range g.Timeline[0].Units {
			frame0Units[u.ID] = u
		}
	}
	for key, o := range g.Orders {
		if u, ok := frame0Units[key]; ok {
			if u.Owner == id || (vis[u.Territory] && vis[o.Target]) {
				orders[key] = o
			}
			continue
		}
		if u := g.unit(key); u != nil {
			if u.Owner == id || (vis[u.Territory] && vis[o.Target]) {
				orders[key] = o
			}
		}
	}
	return orders
}
