package main

import (
	"testing"
	"time"
)

func fixture() *Game {
	g := newGame("TEST01")
	a := g.addPlayer("Alpha", false)
	b := g.addPlayer("Bravo", false)
	for i := range g.Territories {
		g.Territories[i].Owner = a.ID
		if i >= 12 {
			g.Territories[i].Owner = b.ID
		}
	}
	g.Round = 1
	g.setPhase("plan", 90)
	return g
}
func TestMapConnectivity(t *testing.T) {
	g := fixture()
	seen := map[int]bool{0: true}
	q := []int{0}
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		for _, v := range g.Territories[n].Neighbors {
			if !g.adjacent(v, n) {
				t.Fatal("Nonreciprocal edge")
			}
			if !seen[v] {
				seen[v] = true
				q = append(q, v)
			}
		}
	}
	if len(seen) != 24 {
		t.Fatal("Disconnected map")
	}
}
func TestMovementBudgetAndLock(t *testing.T) {
	g := fixture()
	p := g.Players[0]
	p.Resources.Currency = 10
	u := &Unit{ID: "a", Owner: p.ID, Type: "tank", Territory: 0, Count: 4, Health: 100, Ration: 1}
	g.Units = []*Unit{u}
	if g.command(p.ID, Command{Action: "order", Unit: u.ID, Kind: "move", Target: 1}) == nil {
		t.Fatal("Overspending accepted")
	}
	p.Resources.Currency = 20
	if err := g.command(p.ID, Command{Action: "order", Unit: u.ID, Kind: "move", Target: 1}); err != nil {
		t.Fatal(err)
	}
	p.Ready = true
	if g.command(p.ID, Command{Action: "cancel", Unit: u.ID}) == nil {
		t.Fatal("Locked orders changed")
	}
	g.resolve()
	if p.Resources.Currency != 8 || u.Territory != 1 {
		t.Fatal("Movement charge or destination incorrect")
	}
}
func TestFogDoesNotLeak(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	g.Units = []*Unit{{ID: "enemy", Owner: b.ID, Type: "tank", Territory: 12, Count: 8, Health: 100, Ration: 1}}
	g.Orders["enemy"] = Order{"enemy", "move", 6}
	v := g.view(a.ID)
	if len(v["units"].([]*Unit)) != 0 || len(v["orders"].(map[string]Order)) != 0 {
		t.Fatal("Enemy military leaked")
	}
	ts := v["territories"].([]Territory)
	if ts[12].Yield != 0 || ts[12].Owner != "" {
		t.Fatal("Enemy production leaked")
	}
	g.Phase = "simulate"
	if len(g.view(a.ID)["units"].([]*Unit)) != 1 {
		t.Fatal("Simulation not revealed")
	}
}
func TestMergeAndFeed(t *testing.T) {
	g := fixture()
	g.Phase = "gather"
	p := g.Players[0]
	g.Units = []*Unit{{"a", p.ID, "infantry", 0, 4, 50, 1, false}, {"b", p.ID, "infantry", 0, 6, 100, 1, false}}
	if e := g.command(p.ID, Command{Action: "merge", Unit: "a", Other: "b"}); e != nil {
		t.Fatal(e)
	}
	if len(g.Units) != 1 || g.Units[0].Count != 10 || g.Units[0].Health != 80 {
		t.Fatal("Merge did not conserve troops/health")
	}
	food := p.Resources.Food
	if e := g.command(p.ID, Command{Action: "feed", Unit: "a", Ration: 1.5}); e != nil {
		t.Fatal(e)
	}
	if p.Resources.Food != food-15 || g.Units[0].Health != 100 {
		t.Fatal("Food not applied")
	}
	if g.command(p.ID, Command{Action: "feed", Unit: "a", Ration: 1}) == nil {
		t.Fatal("Double feeding accepted")
	}
}
func TestArtilleryRangedAndRecruitment(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	g.Units = []*Unit{{"art", a.ID, "artillery", 6, 5, 100, 1, false}, {"enemy", b.ID, "infantry", 12, 5, 100, 1, false}}
	if e := g.command(a.ID, Command{Action: "order", Unit: "art", Kind: "bombard", Target: 12}); e != nil {
		t.Fatal(e)
	}
	g.resolve()
	if g.unit("art").Territory != 6 || g.Territories[12].Owner != b.ID {
		t.Fatal("Artillery moved or captured")
	}
	g.Phase = "gather"
	before := a.Resources
	g.command(a.ID, Command{Action: "recruit", Type: "tank", Count: 2, Target: 0})
	if a.Resources.Humans != before.Humans-2 || a.Resources.Iron != before.Iron-32 || a.Resources.Currency != before.Currency-8 {
		t.Fatal("Incorrect tank cost")
	}
}
func TestDraftAndPhaseCycle(t *testing.T) {
	g := newGame("TEST02")
	a := g.addPlayer("A", false)
	g.addPlayer("B", true)
	if e := g.command(a.ID, Command{Action: "start"}); e != nil {
		t.Fatal(e)
	}
	for g.Phase == "draft" {
		g.draftOne()
		g.autoDraft()
	}
	counts := map[string]int{}
	for _, tr := range g.Territories {
		counts[tr.Owner]++
	}
	for _, p := range g.Players {
		if counts[p.ID] != 12 {
			t.Fatal("Draft unbalanced")
		}
	}
	g.advance()
	if g.Phase != "simulate" {
		t.Fatal("Expected simulate")
	}
	g.advance()
	if g.Phase != "gather" {
		t.Fatal("Expected gather")
	}
	g.advance()
	if g.Round != 2 || g.Phase != "plan" {
		t.Fatal("Round not advanced")
	}
	g.Deadline = time.Now().Add(-time.Second).UnixMilli()
	g.tick()
	if g.Phase != "simulate" {
		t.Fatal("Timer did not advance")
	}
}

func TestViewIsFrozenForNetworkWriter(t *testing.T) {
	g := fixture()
	p := g.Players[0]
	g.Units = []*Unit{{ID: "a", Owner: p.ID, Type: "infantry", Count: 10, Health: 100, Ration: 1}}
	v := g.view(p.ID)
	g.Units[0].Count = 2
	if v["units"].([]*Unit)[0].Count != 10 {
		t.Fatal("Outgoing state aliases mutable engine state")
	}
}

func TestBorderCrossingAndFullCampaign(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	g.Units = []*Unit{{"a", a.ID, "infantry", 6, 10, 100, 1, false}, {"b", b.ID, "infantry", 12, 10, 100, 1, false}}
	g.Orders["a"] = Order{"a", "move", 12}
	g.Orders["b"] = Order{"b", "move", 6}
	g.resolve()
	if g.unit("a").Territory != 6 || g.unit("b").Territory != 12 {
		t.Fatal("Crossing armies passed through each other")
	}
	for trial := 0; trial < 40; trial++ {
		g = newGame("SOAK")
		for i := 0; i < 4; i++ {
			g.addPlayer("Bot", true)
		}
		g.Phase = "draft"
		g.autoDraft()
		for step := 0; step < 40 && g.Phase != "finished"; step++ {
			g.advance()
			for _, u := range g.Units {
				if u.Count <= 0 || u.Health <= 0 || u.Health > 100 || g.Territories[u.Territory].Owner != u.Owner {
					t.Fatalf("Invalid surviving army: %+v in phase %s", u, g.Phase)
				}
			}
			for _, p := range g.Players {
				if p.Resources.Currency < 0 || p.Resources.Food < 0 {
					t.Fatal("Negative resources")
				}
			}
		}
		if g.Phase != "finished" {
			t.Fatal("Campaign failed to end")
		}
	}
}
