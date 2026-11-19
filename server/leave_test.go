package main

import "testing"

func TestLeaveDissolvesAndNeutralizes(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	a.Capital = 0
	g.Units = []*Unit{{ID: "a", Owner: a.ID, Type: "tank", Territory: 0, Count: 5, Health: 100, Ration: 1}, {ID: "b", Owner: b.ID, Type: "infantry", Territory: 12, Count: 3, Health: 100, Ration: 1}}
	g.Orders["a"] = Order{"a", "move", 1}
	g.recordFrame("Before")
	archive := g.Timeline
	if err := g.leave(a.ID); err != nil {
		t.Fatal(err)
	}
	if len(archive[0].Units) != 2 || archive[0].Owners[0] != a.ID {
		t.Fatal("Surrender mutated an already queued replay snapshot")
	}
	if g.player(a.ID) != nil || g.Host != b.ID || g.unit("a") != nil {
		t.Fatal("Player, host or armies not cleaned up")
	}
	if _, ok := g.Orders["a"]; ok {
		t.Fatal("Orders retained")
	}
	for _, tr := range g.Territories {
		if tr.Owner == a.ID {
			t.Fatal("Territory not neutralized")
		}
	}
	for _, f := range g.Timeline {
		for _, u := range f.Units {
			if u.Owner == a.ID {
				t.Fatal("Army retained in replay")
			}
		}
	}
	if g.Phase != "finished" {
		t.Fatal("Last active commander should win")
	}
	if err := g.leave(a.ID); err == nil {
		t.Fatal("Repeated surrender accepted")
	}
}
func TestLobbyLeaveAndColorReuse(t *testing.T) {
	g := newGame("LOBBY")
	a := g.addPlayer("A", false)
	b := g.addPlayer("B", false)
	g.leave(a.ID)
	c := g.addPlayer("C", false)
	if c.Color == b.Color || g.Host != b.ID {
		t.Fatal("Color collision or failed host transfer")
	}
	if g.Phase != "lobby" {
		t.Fatal("Lobby exit changed phase")
	}
}
func TestDraftLeaveRespectsFourTileLimit(t *testing.T) {
	for leaveAt := 0; leaveAt < 12; leaveAt++ {
		g := newGame("DRAFT")
		for i := 0; i < 3; i++ {
			g.addPlayer("Commander", false)
		}
		g.setPhase("draft", 25)
		for i := 0; i < leaveAt && g.Phase == "draft"; i++ {
			g.draftOne()
		}
		leaver := g.Players[1].ID
		g.leave(leaver)
		for steps := 0; steps < 12 && g.Phase == "draft"; steps++ {
			g.draftOne()
		}
		if g.Phase != "capital" {
			t.Fatalf("Draft stuck at leave point %d", leaveAt)
		}
		for _, p := range g.Players {
			n := 0
			for _, tr := range g.Territories {
				if tr.Owner == p.ID {
					n++
				}
			}
			if n != 4 {
				t.Fatalf("Player has %d tiles after draft exit", n)
			}
		}
	}
}
func TestCapitalLeaveAndRollLeave(t *testing.T) {
	g := newGame("CAP")
	a := g.addPlayer("A", false)
	b := g.addPlayer("B", false)
	c := g.addPlayer("C", false)
	g.setPhase("draft", 25)
	for g.Phase == "draft" {
		g.draftOne()
	}
	for _, p := range []*Player{a, b} {
		for _, tr := range g.Territories {
			if tr.Owner == p.ID {
				g.chooseCapital(p.ID, tr.ID)
				break
			}
		}
	}
	g.leave(c.ID)
	if g.Phase != "plan" || len(g.Units) != 8 {
		t.Fatal("Departed commander blocked capital selection")
	}
	r := newGame("ROLL")
	x := r.addPlayer("X", false)
	r.addPlayer("Y", false)
	r.addPlayer("Z", false)
	r.setPhase("roll", 2)
	r.rollNext()
	r.leave(x.ID)
	if r.RollIndex != 0 {
		t.Fatal("Roll cursor not corrected")
	}
	for r.Phase == "roll" {
		r.rollNext()
	}
	if r.Phase != "draft" {
		t.Fatal("Roll exit stalled setup")
	}
}
func TestLeaveAdvancesReadyCommanders(t *testing.T) {
	g := fixture()
	c := g.addPlayer("C", false)
	g.Territories[23].Owner = c.ID
	g.Phase = "gather"
	for _, p := range g.Players[:2] {
		p.Ready = true
	}
	g.leave(c.ID)
	if g.Phase != "plan" || g.Round != 2 {
		t.Fatal("Leaver blocked ready commanders")
	}
	g.Phase = "simulate"
	c = g.addPlayer("Returning test player", false)
	g.Territories[23].Owner = c.ID
	g.Units = append(g.Units, &Unit{ID: "c", Owner: c.ID, Count: 2, Health: 100, Territory: 23})
	g.Events = append(g.Events, Event{"old bombard", 23, 0, "bombard"})
	g.recordFrame("Old battle")
	g.leave(c.ID)
	if len(g.Timeline) != 1 || len(g.Timeline[0].Events) != 0 {
		t.Fatal("Surrender left phantom combat in replay")
	}
}

func TestBotCleanupAndMidPlanLeaveUnlocks(t *testing.T) {
	// 1. Leaving when only bots remain clears orphaned bots so the server can evict the room.
	g := newGame("BOTCLEAN")
	human := g.addPlayer("Human", false)
	g.addPlayer("Bot 1", true)
	g.addPlayer("Bot 2", true)
	if err := g.leave(human.ID); err != nil {
		t.Fatal(err)
	}
	if len(g.Players) != 0 || g.Phase != "finished" {
		t.Fatalf("Orphaned bots retained after last human left: players=%d phase=%s", len(g.Players), g.Phase)
	}

	// 2. Mid-Plan surrender in a 3-player match resets remaining humans' Ready locks so locked players can adapt.
	m := fixture()
	c := m.addPlayer("Charlie", false)
	m.Territories[23].Owner = c.ID
	m.Phase = "plan"
	m.Players[0].Ready = true
	if err := m.leave(c.ID); err != nil {
		t.Fatal(err)
	}
	if m.Phase != "plan" || m.Players[0].Ready {
		t.Fatal("Locked rival was not unlocked when a commander surrendered during Plan")
	}
}
