package main

import "testing"

func TestFourTilesAndCapital(t *testing.T) {
	g := newGame("SETUP")
	a := g.addPlayer("A", false)
	b := g.addPlayer("B", false)
	g.command(a.ID, Command{Action: "start"})
	if g.Phase != "roll" || a.Roll != 0 || b.Roll != 0 {
		t.Fatal("Rolls revealed early")
	}
	g.rollNext()
	if a.Roll == 0 || b.Roll != 0 {
		t.Fatal("Rolls not sequential")
	}
	g.rollNext()
	if a.Roll == b.Roll {
		t.Fatal("Ties unresolved")
	}
	g.rollNext()
	for g.Phase == "draft" {
		g.draftOne()
	}
	counts := map[string]int{}
	for _, tr := range g.Territories {
		counts[tr.Owner]++
	}
	if counts[a.ID] != 4 || counts[b.ID] != 4 || counts[""] != 16 {
		t.Fatal("Wrong ownership")
	}
	if len(g.Units) != 0 || g.Phase != "capital" {
		t.Fatal("Started before capitals")
	}
	neutral := -1
	for _, tr := range g.Territories {
		if tr.Owner == "" {
			neutral = tr.ID
			break
		}
	}
	if g.chooseCapital(a.ID, neutral) == nil {
		t.Fatal("Neutral capital accepted")
	}
	for _, p := range g.Players {
		for _, tr := range g.Territories {
			if tr.Owner == p.ID {
				if err := g.chooseCapital(p.ID, tr.ID); err != nil {
					t.Fatal(err)
				}
				break
			}
		}
	}
	if g.Phase != "plan" || len(g.Units) != 8 {
		t.Fatal("Wrong initial armies")
	}
	source := g.Territories[neutral].Neighbors[0]
	g.Territories[source].Owner = a.ID
	g.Units = []*Unit{{ID: "scout", Owner: a.ID, Type: "infantry", Territory: source, Count: 2, Health: 100, Ration: 1}}
	g.Orders = map[string]Order{"scout": {"scout", "move", neutral}}
	g.resolve()
	if g.Territories[neutral].Owner != a.ID {
		t.Fatal("Neutral capture failed")
	}
	g.Round = 19
	g.Phase = "gather"
	g.advance()
	if g.Round != 20 || g.Phase != "plan" {
		t.Fatal("Must play round twenty")
	}
	g.Phase = "gather"
	g.advance()
	if g.Phase != "finished" {
		t.Fatal("Campaign must end at twenty")
	}
}
