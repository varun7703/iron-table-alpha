package main

import (
	"fmt"
	"time"
)

// Explicit surrender only: network disconnects keep their reconnectable seat.
func (g *Game) leave(id string) error {
	index := -1
	for i, p := range g.Players {
		if p.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("Player not found")
	}
	name := g.Players[index].Name
	setup := g.Phase == "lobby" || g.Phase == "roll" || g.Phase == "draft" || g.Phase == "capital"
	unitIDs := map[string]bool{}
	living := []*Unit{}
	for _, u := range g.Units {
		if u.Owner == id {
			unitIDs[u.ID] = true
		} else {
			living = append(living, u)
		}
	}
	g.Units = living
	for key := range g.Orders {
		if unitIDs[key] {
			delete(g.Orders, key)
		}
	}
	for i := range g.Territories {
		if g.Territories[i].Owner == id {
			g.Territories[i].Owner = ""
		}
	}
	// Do not replay a surrendered player's armies or ownership on remaining clients.
	timeline := make([]BattleFrame, 0, len(g.Timeline))
	for _, original := range g.Timeline {
		f := original
		f.Owners = append([]string{}, original.Owners...)
		units := []Unit{}
		for _, u := range f.Units {
			if u.Owner == id {
				unitIDs[u.ID] = true
			} else {
				units = append(units, u)
			}
		}
		f.Units = units
		for j, owner := range f.Owners {
			if owner == id {
				f.Owners[j] = ""
			}
		}
		timeline = append(timeline, f)
	}
	g.Timeline = timeline
	for key := range g.Orders {
		if unitIDs[key] {
			delete(g.Orders, key)
		}
	}
	g.Players = append(g.Players[:index], g.Players[index+1:]...)
	if index < g.RollIndex {
		g.RollIndex--
	}
	if g.Host == id {
		g.Host = ""
		for _, p := range g.Players {
			if g.Host == "" || !p.Bot {
				g.Host = p.ID
				if !p.Bot {
					break
				}
			}
		}
	}
	if len(g.Players) == 0 {
		g.Phase = "finished"
		g.Deadline = 0
		g.Winner = "No commanders remaining"
		return nil
	}
	if setup && len(g.Players) < 2 {
		for i := range g.Territories {
			g.Territories[i].Owner = ""
		}
		g.Units = []*Unit{}
		g.Orders = map[string]Order{}
		g.Draft = 0
		g.RollIndex = 0
		for _, p := range g.Players {
			p.Capital = -1
			p.Roll = 0
		}
		g.setPhase("lobby", 0)
		return nil
	}
	if setup {
		switch g.Phase {
		case "draft":
			g.Draft = 0
			for _, t := range g.Territories {
				if t.Owner != "" {
					g.Draft++
				}
			}
			if g.Draft == 4*len(g.Players) {
				g.beginCapitals()
			} else {
				g.setPhase("draft", 25)
				g.autoDraft()
			}
		case "capital":
			g.startIfCapitalsReady()
		}
		return nil
	}
	if g.Phase == "finished" {
		return nil
	}
	if g.Phase == "simulate" {
		g.Timeline = []BattleFrame{}
		g.Events = []Event{}
		g.recordFrame(name + " exited · armies dissolved and territories neutralized")
		g.Deadline = time.Now().Add(12 * time.Second).UnixMilli()
	}
	active := 0
	for _, p := range g.Players {
		if !g.eliminated(p.ID) {
			active++
		}
	}
	if active == 0 {
		g.Phase = "finished"
		g.Deadline = 0
		g.Winner = "No active commanders remaining"
	} else if active == 1 {
		g.finish()
	} else if g.allReady() {
		g.advance()
	}
	return nil
}

func (g *Game) draftPlayer() *Player {
	if len(g.Players) == 0 {
		return nil
	}
	for offset := 0; offset < len(g.Players); offset++ {
		p := g.Players[(g.Draft+offset)%len(g.Players)]
		owned := 0
		for _, t := range g.Territories {
			if t.Owner == p.ID {
				owned++
			}
		}
		if owned < 4 {
			return p
		}
	}
	return nil
}
