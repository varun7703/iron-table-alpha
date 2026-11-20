package main

import (
	"fmt"
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
	// Friendly-sector movement is free (0 currency).
	friendly := &Unit{ID: "f", Owner: p.ID, Type: "tank", Territory: 0, Count: 4, Health: 100, Ration: 1}
	// Hostile-sector movement (6 -> 12) costs 4 * 3 = 12 currency.
	u := &Unit{ID: "a", Owner: p.ID, Type: "tank", Territory: 6, Count: 4, Health: 100, Ration: 1}
	g.Units = []*Unit{friendly, u}
	if err := g.command(p.ID, Command{Action: "order", Unit: friendly.ID, Kind: "move", Target: 1}); err != nil {
		t.Fatalf("Free friendly movement rejected: %v", err)
	}
	if g.reserved(p.ID, "") != 0 {
		t.Fatalf("Friendly movement reserved currency: %d", g.reserved(p.ID, ""))
	}
	if g.command(p.ID, Command{Action: "order", Unit: u.ID, Kind: "move", Target: 12}) == nil {
		t.Fatal("Overspending accepted")
	}
	p.Resources.Currency = 20
	if err := g.command(p.ID, Command{Action: "order", Unit: u.ID, Kind: "move", Target: 12}); err != nil {
		t.Fatal(err)
	}
	p.Ready = true
	if g.command(p.ID, Command{Action: "cancel", Unit: u.ID}) == nil {
		t.Fatal("Locked orders changed")
	}
	g.resolve()
	if p.Resources.Currency != 8 || friendly.Territory != 1 || u.Territory != 12 {
		t.Fatalf("Movement charge or destination incorrect: currency=%d friendly=%d hostile=%d", p.Resources.Currency, friendly.Territory, u.Territory)
	}
}
func TestFogDoesNotLeak(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	// Tile 12 is 1-hop adjacent to Alpha's tile 6; tile 18 is 2 hops away (fogged).
	g.Units = []*Unit{
		{ID: "adj", Owner: b.ID, Type: "tank", Territory: 12, Count: 4, Health: 100, Ration: 1},
		{ID: "far", Owner: b.ID, Type: "tank", Territory: 18, Count: 8, Health: 100, Ration: 1},
	}
	g.Orders["adj"] = Order{"adj", "move", 6}
	g.Orders["far"] = Order{"far", "move", 19}
	v := g.view(a.ID)
	us := v["units"].([]*Unit)
	if len(us) != 1 || us[0].ID != "adj" {
		t.Fatalf("Expected only 1-hop adjacent enemy unit visible, got %d", len(us))
	}
	if len(v["orders"].(map[string]Order)) != 0 {
		t.Fatal("Enemy orders leaked during Plan")
	}
	ts := v["territories"].([]Territory)
	// Static terrain yields are always visible, while distant tile 18 hides ownership.
	if !ts[12].Visible || ts[12].Owner != b.ID || ts[12].Yield == 0 {
		t.Fatalf("1-hop adjacent sector should show owner & yield: %+v", ts[12])
	}
	if ts[18].Visible || ts[18].Owner != "" || ts[18].Neutral || ts[18].Yield == 0 {
		t.Fatalf("2-hop sector must hide owner/neutral while keeping static yield: %+v", ts[18])
	}
	g.Phase = "simulate"
	simUs := g.view(a.ID)["units"].([]*Unit)
	if len(simUs) != 1 || simUs[0].ID != "adj" {
		t.Fatalf("Simulation should apply 1-hop LoS (expected only adj), got %d", len(simUs))
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
	if err := g.command(a.ID, Command{Action: "recruit", Type: "tank", Count: 2, Target: 0}); err != nil {
		t.Fatal(err)
	}
	if a.Resources.Humans != before.Humans-6 || a.Resources.Iron != before.Iron-16 || a.Resources.Currency != before.Currency-16 {
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
	for g.Phase == "roll" {
		g.rollNext()
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
		if counts[p.ID] != 4 {
			t.Fatal("Draft unbalanced")
		}
	}
	for _, tr := range g.Territories {
		if tr.Owner == a.ID {
			g.chooseCapital(a.ID, tr.ID)
			break
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
		for step := 0; step < 65 && g.Phase != "finished"; step++ {
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

func TestStackSplitInvarianceAndChaffFix(t *testing.T) {
	// 1. Destination battle: 5 Tanks + 5x1 Inf (chaff) vs 8 Tanks must match merged 5 Tanks + 1x5 Inf vs 8 Tanks.
	gMerged := fixture()
	aM, bM := gMerged.Players[0], gMerged.Players[1]
	gMerged.Units = []*Unit{
		{ID: "at", Owner: aM.ID, Type: "tank", Territory: 6, Count: 5, Health: 100, Ration: 1},
		{ID: "ai", Owner: aM.ID, Type: "infantry", Territory: 6, Count: 5, Health: 100, Ration: 1},
		{ID: "bt", Owner: bM.ID, Type: "tank", Territory: 12, Count: 8, Health: 100, Ration: 1},
	}
	gMerged.Orders["at"] = Order{"at", "move", 12}
	gMerged.Orders["ai"] = Order{"ai", "move", 12}
	gMerged.resolve()

	gSplit := fixture()
	aS, bS := gSplit.Players[0], gSplit.Players[1]
	gSplit.Units = []*Unit{
		{ID: "at", Owner: aS.ID, Type: "tank", Territory: 6, Count: 5, Health: 100, Ration: 1},
		{ID: "bt", Owner: bS.ID, Type: "tank", Territory: 12, Count: 8, Health: 100, Ration: 1},
	}
	gSplit.Orders["at"] = Order{"at", "move", 12}
	for i := 0; i < 5; i++ {
		id := uid()
		gSplit.Units = append(gSplit.Units, &Unit{ID: id, Owner: aS.ID, Type: "infantry", Territory: 6, Count: 1, Health: 100, Ration: 1})
		gSplit.Orders[id] = Order{id, "move", 12}
	}
	gSplit.resolve()

	if gMerged.Territories[12].Owner != bM.ID || gSplit.Territories[12].Owner != bS.ID {
		t.Fatal("Chaff screen upset superior 8-tank defender")
	}
	if gSplit.unit("at").Count != gMerged.unit("at").Count || gSplit.unit("bt").Count != gMerged.unit("bt").Count {
		t.Fatalf("Split tank casualties diverged: split=%d/%d vs merged=%d/%d", gSplit.unit("at").Count, gSplit.unit("bt").Count, gMerged.unit("at").Count, gMerged.unit("bt").Count)
	}
	splitInf := 0
	for _, u := range gSplit.Units {
		if u.Owner == aS.ID && u.Type == "infantry" {
			splitInf += u.Count
		}
	}
	if splitInf != gMerged.unit("ai").Count {
		t.Fatalf("Split infantry casualties diverged: split=%d vs merged=%d", splitInf, gMerged.unit("ai").Count)
	}

	// 2. Border clash: 1x10 Inf vs 5x2 Inf crossing must mirror 1x10 vs 1x10 without Nx AOE multiplication.
	gBorder := fixture()
	pA, pB := gBorder.Players[0], gBorder.Players[1]
	gBorder.Units = []*Unit{{ID: "a10", Owner: pA.ID, Type: "infantry", Territory: 6, Count: 10, Health: 100, Ration: 1}}
	gBorder.Orders["a10"] = Order{"a10", "move", 12}
	for i := 0; i < 5; i++ {
		id := uid()
		gBorder.Units = append(gBorder.Units, &Unit{ID: id, Owner: pB.ID, Type: "infantry", Territory: 12, Count: 2, Health: 100, Ration: 1})
		gBorder.Orders[id] = Order{id, "move", 6}
	}
	gBorder.resolve()
	bTotal := 0
	for _, u := range gBorder.Units {
		if u.Owner == pB.ID {
			bTotal += u.Count
		}
	}
	if gBorder.unit("a10").Count != bTotal || bTotal != 8 {
		t.Fatalf("Symmetric border clash asymmetric due to splitting: A=%d, B=%d", gBorder.unit("a10").Count, bTotal)
	}
}

func TestDefenderHoldsOnTieAndNeutralEval(t *testing.T) {
	// Isolated incumbent defender tied in its sole territory must hold and survive.
	g := newGame("TIE01")
	a := g.addPlayer("Attacker", false)
	b := g.addPlayer("Defender", false)
	for i := range g.Territories {
		g.Territories[i].Owner = a.ID
	}
	g.Territories[0].Owner = b.ID
	g.Units = []*Unit{
		{ID: "atk", Owner: a.ID, Type: "infantry", Territory: 1, Count: 4, Health: 100, Ration: 1},
		{ID: "def", Owner: b.ID, Type: "infantry", Territory: 0, Count: 4, Health: 100, Ration: 1},
	}
	g.Orders = map[string]Order{"atk": {"atk", "move", 0}}
	g.resolve()
	if g.Territories[0].Owner != b.ID || g.unit("def") == nil || g.unit("def").Territory != 0 {
		t.Fatal("Incumbent defender lost or died on a tie")
	}
	if g.unit("atk") == nil || g.unit("atk").Territory != 1 {
		t.Fatal("Tied attacker did not retreat to origin")
	}

	// Tied attackers entering a neutral territory both retreat and leave it neutral.
	g2 := newGame("TIE02")
	p1 := g2.addPlayer("One", false)
	p2 := g2.addPlayer("Two", false)
	g2.Territories[0].Owner = p1.ID
	g2.Territories[2].Owner = p2.ID
	g2.Units = []*Unit{
		{ID: "u1", Owner: p1.ID, Type: "infantry", Territory: 0, Count: 4, Health: 100, Ration: 1},
		{ID: "u2", Owner: p2.ID, Type: "infantry", Territory: 2, Count: 4, Health: 100, Ration: 1},
	}
	g2.Orders = map[string]Order{"u1": {"u1", "move", 1}, "u2": {"u2", "move", 1}}
	g2.resolve()
	if g2.Territories[1].Owner != "" || g2.unit("u1").Territory != 0 || g2.unit("u2").Territory != 2 {
		t.Fatal("Neutral tie was not left neutral with mutual retreat")
	}

	// Unopposed capture emits an explicit capture event.
	g3 := newGame("CAPEV")
	c1 := g3.addPlayer("Vanguard", false)
	g3.addPlayer("Rival", false)
	g3.Territories[0].Owner = c1.ID
	g3.Units = []*Unit{{ID: "scout", Owner: c1.ID, Type: "infantry", Territory: 0, Count: 2, Health: 100, Ration: 1}}
	g3.Orders = map[string]Order{"scout": {"scout", "move", 1}}
	g3.resolve()
	foundCapture := false
	for _, ev := range g3.Events {
		if ev.Kind == "capture" && ev.To == 1 {
			foundCapture = true
		}
	}
	if !foundCapture {
		t.Fatal("Unopposed capture did not emit capture event")
	}
}

func TestWeightedReinforceSectorCapAndPriorityAutoFeed(t *testing.T) {
	g := fixture()
	g.Phase = "gather"
	p := g.Players[0]
	p.Resources = Resources{Humans: 50, Food: 15, Iron: 100, Currency: 200}
	g.Units = []*Unit{
		{ID: "inf", Owner: p.ID, Type: "infantry", Territory: 0, Count: 10, Health: 100, Ration: 1, Fed: false},
		{ID: "tnk", Owner: p.ID, Type: "tank", Territory: 0, Count: 5, Health: 20, Ration: 1, Fed: false},
	}
	if err := g.command(p.ID, Command{Action: "reinforce", Unit: "tnk", Count: 5}); err != nil {
		t.Fatal(err)
	}
	if g.unit("tnk").Count != 10 || g.unit("tnk").Health != 60 {
		t.Fatalf("Reinforce did not apply troop-weighted health: %+v", g.unit("tnk"))
	}
	// Sector cap of 4 separate stacks on recruit (2 already present -> 2 more allowed, 5th rejected).
	for i := 0; i < 2; i++ {
		if err := g.command(p.ID, Command{Action: "recruit", Type: "infantry", Count: 1, Target: 0}); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.command(p.ID, Command{Action: "recruit", Type: "infantry", Count: 1, Target: 0}); err == nil {
		t.Fatal("Accepted 5th separate army stack in one sector")
	}
	// Priority autoFeed: with 15 food for 10 Infantry + 10 Tanks (leaving only those two), Tanks get 1.0 and Infantry gets 0.5 (0 starvation).
	g.Units = g.Units[:2]
	g.autoFeed(p)
	if g.unit("tnk").Ration != 1 || g.unit("inf").Ration != .5 || p.Resources.Food != 0 {
		t.Fatalf("Priority autoFeed misallocated food: tank=%v inf=%v rem=%d", g.unit("tnk").Ration, g.unit("inf").Ration, p.Resources.Food)
	}
}

func TestRoundTwentyEndsAfterSimulateAndUnready(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	if err := g.command(a.ID, Command{Action: "ready"}); err != nil {
		t.Fatal(err)
	}
	if !a.Ready {
		t.Fatal("Expected ready")
	}
	if err := g.command(a.ID, Command{Action: "unready"}); err != nil {
		t.Fatal(err)
	}
	if a.Ready {
		t.Fatal("Unready did not clear lock")
	}
	// Gather ready should not prematurely consume food if unlocked before advance.
	g.Phase = "gather"
	g.Units = []*Unit{{ID: "u", Owner: a.ID, Type: "infantry", Territory: 0, Count: 4, Health: 100, Ration: 1, Fed: false}}
	foodBefore := a.Resources.Food
	_ = g.command(a.ID, Command{Action: "ready"})
	_ = g.command(a.ID, Command{Action: "unready"})
	if a.Resources.Food != foodBefore || g.unit("u").Fed {
		t.Fatal("Lock/unlock in Gather mutated food prematurely")
	}

	// Round 20 ends directly after Simulate without entering untimed Gather.
	g.Round = 20
	g.Phase = "plan"
	_ = g.command(a.ID, Command{Action: "ready"})
	_ = g.command(b.ID, Command{Action: "ready"})
	if g.Phase != "simulate" {
		t.Fatal("Expected simulate")
	}
	g.advance()
	if g.Phase != "finished" {
		t.Fatalf("Round 20 did not finish after simulate: phase=%s", g.Phase)
	}
}

func TestBombardmentSplitInvarianceAndMixedHealthApplyDamage(t *testing.T) {
	// Bombardment against 1x10 Infantry vs 10x1 Infantry must inflict identical losses.
	runBombard := func(stacks int) int {
		g := fixture()
		a, b := g.Players[0], g.Players[1]
		g.Territories[0].Owner = a.ID
		g.Territories[1].Owner = b.ID
		g.Units = []*Unit{{ID: "art", Owner: a.ID, Type: "artillery", Territory: 0, Count: 4, Health: 100, Ration: 1, Fed: true}}
		per := 10 / stacks
		for i := 0; i < stacks; i++ {
			g.Units = append(g.Units, &Unit{
				ID: fmt.Sprintf("def-%d", i), Owner: b.ID, Type: "infantry",
				Territory: 1, Count: per, Health: 100, Ration: 1, Fed: true,
			})
		}
		g.Phase = "plan"
		g.Orders = map[string]Order{"art": {Kind: "bombard", Target: 1}}
		g.resolve()
		rem := 0
		for _, u := range g.Units {
			if u.Owner == b.ID {
				rem += u.Count
			}
		}
		return rem
	}
	mergedRem := runBombard(1)
	splitRem := runBombard(10)
	if mergedRem != splitRem {
		t.Fatalf("Bombardment split exploit: 1x10 left %d troops vs 10x1 left %d troops", mergedRem, splitRem)
	}

	// Mixed-health applyDamage: a stack wiped by 0% HP must not steal pooled
	// remainder kills from healthy stacks nor cause extra over-kills.
	lowHP := &Unit{ID: "low", Owner: "p1", Type: "infantry", Territory: 0, Count: 1, Health: 5}
	healthy := &Unit{ID: "ok", Owner: "p1", Type: "infantry", Territory: 0, Count: 3, Health: 100}
	// Total damage = 15 (pooled floor = 1 kill). lowHP takes 5 dmg (HP -> 0, killed = 1).
	// healthy takes 10 dmg (baseLoss = 1, HP -> 80). Total already killed = 2 >= 1, so extra = 0.
	applyDamage([]*Unit{lowHP, healthy}, map[string]float64{"low": 5, "ok": 10})
	if lowHP.Count != 0 || healthy.Count != 2 {
		t.Fatalf("Mixed-health over-kill/under-kill: low=%d healthy=%d (want 0, 2)", lowHP.Count, healthy.Count)
	}

	// Second case: 3 stacks of 1 troop each take 8 damage (total = 24, pooled floor = 2 kills).
	// Stack 0 dies from low HP (1 kill); remaining 1 pooled kill must hit a living stack.
	s0 := &Unit{ID: "s0", Owner: "p1", Type: "infantry", Territory: 0, Count: 1, Health: 10}
	s1 := &Unit{ID: "s1", Owner: "p1", Type: "infantry", Territory: 0, Count: 1, Health: 100}
	s2 := &Unit{ID: "s2", Owner: "p1", Type: "infantry", Territory: 0, Count: 1, Health: 100}
	applyDamage([]*Unit{s0, s1, s2}, map[string]float64{"s0": 8, "s1": 8, "s2": 8})
	alive := s0.Count + s1.Count + s2.Count
	if s0.Count != 0 || alive != 1 {
		t.Fatalf("Mixed-health pooled remainder failed: s0=%d s1=%d s2=%d (want s0=0, total alive=1)", s0.Count, s1.Count, s2.Count)
	}
}
