package main

import (
	"fmt"
	rng "math/rand/v2"
	"sort"
)

func (g *Game) rollNext() {
	if g.RollIndex < len(g.Players) {
		p := g.Players[g.RollIndex]
		p.Roll = 1 + rng.IntN(6)
		for {
			tie := false
			for i := 0; i < g.RollIndex; i++ {
				if g.Players[i].Roll == p.Roll {
					tie = true
				}
			}
			if !tie {
				break
			}
			p.Roll = 1 + rng.IntN(6)
		}
		g.RollIndex++
		g.setPhase("roll", 2)
		return
	}
	sort.SliceStable(g.Players, func(i, j int) bool { return g.Players[i].Roll > g.Players[j].Roll })
	g.setPhase("draft", 25)
	g.autoDraft()
}
func (g *Game) beginCapitals() {
	g.setPhase("capital", 0)
	for _, p := range g.Players {
		if p.Bot {
			if capID := g.pickBotCapital(p); capID >= 0 {
				p.Capital = capID
			}
		}
	}
	g.startIfCapitalsReady()
}
func (g *Game) startIfCapitalsReady() {
	for _, p := range g.Players {
		if p.Capital < 0 {
			return
		}
	}
	for _, t := range g.Territories {
		if t.Owner != "" {
			g.Units = append(g.Units, &Unit{uid(), t.Owner, "infantry", t.ID, 4, 100, 1, false})
		}
	}
	g.Round = 1
	g.setPhase("plan", 90)
}
func (g *Game) chooseCapital(id string, target int) error {
	p := g.player(id)
	if g.Phase != "capital" || p.Capital >= 0 {
		return fmt.Errorf("Capital selection is not available")
	}
	if target < 0 || target >= len(g.Territories) || g.Territories[target].Owner != id {
		return fmt.Errorf("Choose one of your four drafted territories")
	}
	p.Capital = target
	g.startIfCapitalsReady()
	return nil
}
