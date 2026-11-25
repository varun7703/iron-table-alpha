package main

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestPhaseSpecificFogOfWar(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	a.Capital, b.Capital = 0, 23
	g.Units = []*Unit{
		{ID: "own", Owner: a.ID, Type: "infantry", Territory: 6, Count: 4, Health: 100, Ration: 1},
		{ID: "near", Owner: b.ID, Type: "infantry", Territory: 12, Count: 4, Health: 100, Ration: 1},
		{ID: "far", Owner: b.ID, Type: "tank", Territory: 22, Count: 3, Health: 100, Ration: 1},
	}
	g.Phase = "capital"
	if g.view(a.ID)["players"].([]Player)[1].Capital != -1 {
		t.Fatal("Rival capital pick exposed during setup")
	}
	g.Phase = "plan"
	plan := g.view(a.ID)
	tiles := plan["territories"].([]Territory)
	if !tiles[6].Visible || !tiles[12].Visible || tiles[23].Visible {
		t.Fatal("Plan must reveal own and adjacent tiles only")
	}
	if len(plan["units"].([]*Unit)) != 2 {
		t.Fatal("Plan must show adjacent army and hide distant army")
	}
	if plan["players"].([]Player)[1].Capital != -1 {
		t.Fatal("Plan exposed distant capital")
	}
	// Every player's view uses its own territory boundary.
	for _, p := range g.Players {
		expected := map[int]bool{}
		for _, tile := range g.Territories {
			if tile.Owner == p.ID {
				g.markOneHop(expected, tile.ID)
			}
		}
		for _, tile := range g.view(p.ID)["territories"].([]Territory) {
			if tile.Visible != expected[tile.ID] {
				t.Fatalf("Plan visibility mismatch for %s sector %d", p.Name, tile.ID)
			}
		}
	}
	g.Territories[23].Owner = ""
	g.Orders["far"] = Order{"far", "move", 23}
	g.resolve()
	g.setPhase("simulate", simulateDurationSeconds)
	sim := g.view(a.ID)
	for _, tile := range sim["territories"].([]Territory) {
		if !tile.Visible {
			t.Fatal("Simulate hid a tile")
		}
	}
	if len(sim["units"].([]*Unit)) != len(g.Units) {
		t.Fatal("Simulate hid armies")
	}
	if _, ok := sim["orders"].(map[string]Order)["far"]; !ok {
		t.Fatal("Simulate hid a distant order")
	}
	if len(sim["timeline"].([]BattleFrame)) != len(g.Timeline) {
		t.Fatal("Simulate omitted battle frames")
	}
	if len(sim["events"].([]Event)) != len(g.Events) {
		t.Fatal("Simulate omitted battle events")
	}
	if sim["players"].([]Player)[1].Capital != 23 {
		t.Fatal("Simulate hid distant capital")
	}
	g.advance()
	for _, p := range g.Players {
		gather := g.view(p.ID)
		for _, tile := range gather["territories"].([]Territory) {
			if tile.Visible != (g.Territories[tile.ID].Owner == p.ID) {
				t.Fatalf("Gather visibility mismatch for %s sector %d", p.Name, tile.ID)
			}
			if !tile.Visible && (tile.Owner != "" || tile.IsCapital || tile.Neutral) {
				t.Fatal("Gather exposed hidden territory details")
			}
		}
		for _, u := range gather["units"].([]*Unit) {
			if u.Owner != p.ID {
				t.Fatal("Gather exposed enemy army")
			}
		}
		if len(gather["lastEvents"].([]Event)) != len(g.LastEvents) {
			t.Fatal("Last Round report lost a battle revealed in Simulate")
		}
		if len(gather["timeline"].([]BattleFrame)) != 0 {
			t.Fatal("Gather retained live replay")
		}
	}
	// Returning to Plan restores adjacent visibility after Gather.
	g.Phase = "plan"
	if !g.view(a.ID)["territories"].([]Territory)[12].Visible {
		t.Fatal("Plan did not restore adjacent visibility")
	}
	spec := g.addPlayer("Spectator", false)
	for _, tile := range g.view(spec.ID)["territories"].([]Territory) {
		if !tile.Visible {
			t.Fatal("Eliminated spectator must see full map")
		}
	}
}

func TestP2MeaningfulCapitalsIncomeDefenseAndVP(t *testing.T) {
	// 1. Capital HQ Income (+4 Currency, +2 Humans) only while holding own capital.
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	a.Capital = 0 // owned by Alpha
	b.Capital = 1 // owned by Alpha (Bravo lost its capital!)
	aBefore := a.Resources
	bBefore := b.Resources
	g.Phase = "simulate"
	g.advance() // -> gather (collects income)
	// Alpha owns 12 tiles (each +8C/+3H) + own capital (+4C/+2H) = +100C / +38H.
	// Bravo owns 12 tiles (each +8C/+3H) + 0 own capital = +96C / +36H.
	if a.Resources.Currency-aBefore.Currency != 12*8+capitalCurrencyBonus || a.Resources.Humans-aBefore.Humans != 12*3+capitalHumansBonus {
		t.Fatalf("Alpha capital HQ income wrong: +C=%d +H=%d", a.Resources.Currency-aBefore.Currency, a.Resources.Humans-aBefore.Humans)
	}
	if b.Resources.Currency-bBefore.Currency != 12*8 || b.Resources.Humans-bBefore.Humans != 12*3 {
		t.Fatalf("Bravo should not receive HQ income while capital is lost: +C=%d +H=%d", b.Resources.Currency-bBefore.Currency, b.Resources.Humans-bBefore.Humans)
	}

	// 2. Capital Defense (+15% direct combat in destination battle, stacking with +20% defend = 1.38x; NOT in border clash).
	gCap := fixture()
	aC, bC := gCap.Players[0], gCap.Players[1]
	bC.Capital = 12
	gCap.Units = []*Unit{
		{ID: "atk", Owner: aC.ID, Type: "infantry", Territory: 6, Count: 5, Health: 100, Ration: 1},
		{ID: "def", Owner: bC.ID, Type: "infantry", Territory: 12, Count: 5, Health: 100, Ration: 1},
	}
	gCap.Orders["atk"] = Order{"atk", "move", 12}
	gCap.Orders["def"] = Order{"def", "defend", 12}
	gCap.resolve()
	// Defender deals 5 * 2 * 1.2 * 1.15 = 13.8 dmg to attacker -> 1 kill (10 HP), health = 100 - (13.8/50)*60 = 83.44.
	// Attacker deals 5 * 2 = 10.0 dmg to defender -> 1 kill (10 HP), health = 100 - (10/50)*60 = 88.0.
	atk := gCap.unit("atk")
	def := gCap.unit("def")
	if atk == nil || def == nil {
		t.Fatalf("Expected both stacks surviving: atk=%+v def=%+v", atk, def)
	}
	if atk.Count != 4 || math.Abs(atk.Health-83.44) > 0.01 {
		t.Fatalf("Capital + entrench damage to attacker mismatch: %+v", atk)
	}
	if def.Count != 4 || math.Abs(def.Health-88.0) > 0.01 {
		t.Fatalf("Attacker damage to capital defender mismatch: %+v", def)
	}
	if gCap.Territories[12].Owner != bC.ID {
		t.Fatal("Capital defender should retain sector 12")
	}

	// Border clash out of own capital must NOT grant capital defense bonus.
	gBorder := fixture()
	aB, bB := gBorder.Players[0], gBorder.Players[1]
	aB.Capital = 6
	gBorder.Units = []*Unit{
		{ID: "a", Owner: aB.ID, Type: "infantry", Territory: 6, Count: 5, Health: 100, Ration: 1},
		{ID: "b", Owner: bB.ID, Type: "infantry", Territory: 12, Count: 5, Health: 100, Ration: 1},
	}
	gBorder.Orders["a"] = Order{"a", "move", 12}
	gBorder.Orders["b"] = Order{"b", "move", 6}
	gBorder.resolve()
	if math.Abs(gBorder.unit("a").Health-gBorder.unit("b").Health) > 1e-6 || gBorder.unit("a").Count != gBorder.unit("b").Count {
		t.Fatalf("Border clash granted asymmetric capital bonus: a=%+v b=%+v", gBorder.unit("a"), gBorder.unit("b"))
	}

	// 3. VP Scoring: Territories + 2 * Capitals Controlled (including captured enemy capitals).
	gVP := fixture()
	p1, p2 := gVP.Players[0], gVP.Players[1]
	p1.Capital = 0  // owned by p1
	p2.Capital = 11 // owned by p1! So p1 holds 2 capitals (11 tiles + 4 VP = 15 VP) while p2 holds 13 tiles + 0 capitals = 13 VP!
	gVP.Territories[10].Owner = p2.ID
	vp1, t1, c1 := gVP.playerScore(p1.ID)
	vp2, t2, c2 := gVP.playerScore(p2.ID)
	if t1 != 11 || c1 != 2 || vp1 != 15 {
		t.Fatalf("p1 score wrong: vp=%d terr=%d caps=%d", vp1, t1, c1)
	}
	if t2 != 13 || c2 != 0 || vp2 != 13 {
		t.Fatalf("p2 score wrong: vp=%d terr=%d caps=%d", vp2, t2, c2)
	}
	gVP.finish()
	if !strings.Contains(gVP.Winner, p1.Name) || !strings.Contains(gVP.Winner, "15 VP") {
		t.Fatalf("Expected p1 to win via capital VP, got %q", gVP.Winner)
	}
}

func TestP2GatherOnlySplitAndSixStackCap(t *testing.T) {
	g := fixture()
	a := g.Players[0]
	g.Units = []*Unit{{ID: "u1", Owner: a.ID, Type: "infantry", Territory: 6, Count: 12, Health: 75, Ration: 0.5}}
	if err := g.command(a.ID, Command{Action: "order", Unit: "u1", Kind: "move", Target: 12}); err != nil {
		t.Fatal(err)
	}
	if err := g.command(a.ID, Command{Action: "split", Unit: "u1", Count: 2}); err == nil {
		t.Fatal("Allowed split during Plan")
	}
	if len(g.Units) != 1 || g.Units[0].Count != 12 || g.reserved(a.ID, "") != 12 {
		t.Fatal("Rejected Plan split mutated army or order")
	}
	g.Phase = "gather"
	g.Orders = map[string]Order{}
	for _, count := range []int{0, -1, 12} {
		if err := g.command(a.ID, Command{Action: "split", Unit: "u1", Count: count}); err == nil {
			t.Fatalf("Accepted invalid split count %d", count)
		}
	}
	for i := 1; i < 6; i++ {
		if err := g.command(a.ID, Command{Action: "split", Unit: "u1", Count: 1}); err != nil {
			t.Fatalf("Split to stack %d failed: %v", i+1, err)
		}
	}
	if len(g.Units) != 6 || g.Units[0].Count != 7 {
		t.Fatal("Unexpected split counts")
	}
	for _, u := range g.Units {
		if u.Health != 75 || u.Ration != 0.5 || u.Type != "infantry" {
			t.Fatalf("Split changed stats: %+v", u)
		}
	}
	if err := g.command(a.ID, Command{Action: "split", Unit: "u1", Count: 1}); err == nil {
		t.Fatal("Allowed seventh stack via split")
	}
	if err := g.command(a.ID, Command{Action: "recruit", Type: "infantry", Target: 6, Count: 1}); err == nil {
		t.Fatal("Allowed seventh stack via recruit")
	}
	if err := g.command(a.ID, Command{Action: "merge", Unit: g.Units[4].ID, Other: g.Units[5].ID}); err != nil {
		t.Fatal(err)
	}
	if err := g.command(a.ID, Command{Action: "feed", Unit: "u1", Ration: 1}); err != nil {
		t.Fatal(err)
	}
	if err := g.command(a.ID, Command{Action: "split", Unit: "u1", Count: 1}); err == nil {
		t.Fatal("Allowed splitting fed army")
	}
}

func TestP2CapitalAndGatherAFKTimers(t *testing.T) {
	g := newGame("AFK01")
	a := g.addPlayer("Alpha", false)
	b := g.addPlayer("Bravo", false)
	g.Phase = "draft"
	for i := 0; i < 4; i++ {
		g.Territories[i].Owner = a.ID
		g.Territories[12+i].Owner = b.ID
	}
	g.Draft = 8
	g.beginCapitals()
	if g.Phase != "capital" || g.Deadline <= time.Now().UnixMilli() {
		t.Fatalf("Expected 45s capital deadline, phase=%s deadline=%d", g.Phase, g.Deadline)
	}
	// Alpha picks manually; Bravo is AFK -> deadline expires -> Bravo auto-assigned capital and game starts Plan.
	if err := g.chooseCapital(a.ID, 0); err != nil {
		t.Fatal(err)
	}
	if g.Phase != "capital" {
		t.Fatal("Started before Bravo picked or timed out")
	}
	g.Deadline = time.Now().Add(-time.Second).UnixMilli()
	g.tick()
	if g.Phase != "plan" || g.Round != 1 {
		t.Fatalf("Capital timeout did not auto-assign and advance to Plan: phase=%s round=%d", g.Phase, g.Round)
	}
	if b.Capital < 12 || b.Capital > 15 {
		t.Fatalf("Bravo auto-assigned capital out of owned range: %d", b.Capital)
	}

	// Gather has no deadline and waits for every active commander to lock in.
	g.Phase = "simulate"
	g.advance()
	if g.Phase != "gather" || g.Deadline != 0 {
		t.Fatalf("Gather must be untimed: phase=%s deadline=%d", g.Phase, g.Deadline)
	}
	g.tick()
	if g.Phase != "gather" {
		t.Fatal("Gather advanced without lock-in")
	}
	if err := g.command(a.ID, Command{Action: "ready"}); err != nil {
		t.Fatal(err)
	}
	g.tick()
	if g.Phase != "gather" {
		t.Fatal("Gather advanced before all players locked in")
	}
	if err := g.command(b.ID, Command{Action: "ready"}); err != nil {
		t.Fatal(err)
	}
	g.tick()
	if g.Phase != "plan" || g.Round != 2 {
		t.Fatal("Gather did not advance after all players locked in")
	}
	for _, u := range g.Units {
		if !u.Fed {
			t.Fatalf("Unit not auto-fed on Gather completion: %+v", u)
		}
	}

}

func TestP2AdversarialHardeningRegressions(t *testing.T) {
	// 1. Simulate includes distant captures; movement must not be marked as destruction.
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	// Alpha owns 0..11; Bravo owns 12..17; 18..23 neutral (2+ hops from Alpha).
	for i := 18; i < 24; i++ {
		g.Territories[i].Owner = ""
	}
	g.Units = []*Unit{
		{ID: "bRetreatIntoFog", Owner: b.ID, Type: "infantry", Territory: 12, Count: 4, Health: 100, Ration: 1},
		{ID: "bFarCap1", Owner: b.ID, Type: "infantry", Territory: 13, Count: 4, Health: 100, Ration: 1},
		{ID: "bFarCap2", Owner: b.ID, Type: "infantry", Territory: 14, Count: 4, Health: 100, Ration: 1},
	}
	// bRetreatIntoFog moves from visible 12 -> fogged 18; bFarCap1/2 capture fogged 19 & 20.
	g.Orders["bRetreatIntoFog"] = Order{"bRetreatIntoFog", "move", 18}
	g.Orders["bFarCap1"] = Order{"bFarCap1", "move", 19}
	g.Orders["bFarCap2"] = Order{"bFarCap2", "move", 20}
	g.resolve()
	g.setPhase("simulate", simulateDurationSeconds)

	vA := g.view(a.ID)
	tlA := vA["timeline"].([]BattleFrame)
	// All simulation frames are visible, including distant captures.
	if len(tlA) != len(g.Timeline) {
		t.Fatal("Simulate omitted distant frames")
	}
	if vA["simulationDuration"].(int) != max(12, len(g.Timeline)*3)*1000 {
		t.Fatal("Playback duration does not match full timeline")
	}
	for _, fr := range tlA {
		for _, deadID := range fr.Destroyed {
			if deadID == "bRetreatIntoFog" {
				t.Fatal("Enemy unit moving into fog was falsely marked Destroyed")
			}
		}
	}
	// Advance to gather -> Timeline cleared and simulationDuration == 0.
	g.advance()
	if len(g.Timeline) != 0 || g.view(a.ID)["simulationDuration"].(int) != 0 {
		t.Fatalf("Timeline/simulationDuration leaked after simulate: len=%d dur=%v", len(g.Timeline), g.view(a.ID)["simulationDuration"])
	}

	// 2. Unit actually killed in visible sector IS reported in BattleFrame.Destroyed.
	gKill := fixture()
	aK, bK := gKill.Players[0], gKill.Players[1]
	gKill.Units = []*Unit{
		{ID: "aTank", Owner: aK.ID, Type: "tank", Territory: 6, Count: 6, Health: 100, Ration: 1},
		{ID: "bTiny", Owner: bK.ID, Type: "infantry", Territory: 12, Count: 1, Health: 100, Ration: 1},
	}
	gKill.Orders["aTank"] = Order{"aTank", "move", 12}
	gKill.resolve()
	gKill.setPhase("simulate", simulateDurationSeconds)
	tlK := gKill.view(aK.ID)["timeline"].([]BattleFrame)
	foundDestroyed := false
	for _, fr := range tlK {
		for _, id := range fr.Destroyed {
			if id == "bTiny" {
				foundDestroyed = true
			}
		}
	}
	if !foundDestroyed {
		t.Fatal("Visible destroyed unit missing from BattleFrame.Destroyed")
	}

	// 3. Conquered capital VP survives original owner's surrender / disconnect expiry.
	g3 := newGame("CAPVP")
	pA := g3.addPlayer("Alpha", false)
	pB := g3.addPlayer("Bravo", false)
	pC := g3.addPlayer("Charlie", false)
	for i := 0; i < 8; i++ {
		g3.Territories[i].Owner = pA.ID
	}
	for i := 8; i < 18; i++ {
		g3.Territories[i].Owner = pB.ID
	}
	for i := 18; i < 24; i++ {
		g3.Territories[i].Owner = pC.ID
	}
	g3.Phase = "capital"
	if err := g3.chooseCapital(pA.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := g3.chooseCapital(pB.ID, 8); err != nil {
		t.Fatal(err)
	}
	if err := g3.chooseCapital(pC.ID, 23); err != nil {
		t.Fatal(err)
	}
	// Alpha conquers Charlie's capital at 23 (Alpha: 9 tiles + 2 caps = 13 VP; Bravo: 10 tiles + 1 cap = 12 VP).
	g3.Territories[23].Owner = pA.ID
	if err := g3.leave(pC.ID); err != nil {
		t.Fatal(err)
	}
	vpA, terrA, capsA := g3.playerScore(pA.ID)
	if vpA != 13 || terrA != 9 || capsA != 2 || !g3.Territories[23].IsCapital {
		t.Fatalf("Conquered capital VP lost on original owner surrender: vp=%d terr=%d caps=%d isCap=%v", vpA, terrA, capsA, g3.Territories[23].IsCapital)
	}
	// Verify rival VP is concealed during plan and revealed on finished.
	psPlan := g3.view(pA.ID)["players"].([]Player)
	for _, pl := range psPlan {
		if pl.ID == pB.ID && pl.VP != 0 {
			t.Fatalf("Rival VP leaked during Plan: %d", pl.VP)
		}
	}

	// 4. Bot AI capital awareness: never abandon own threatened capital; prioritize undefended enemy capital;
	// sectorThreat must not apply 1.15x defensive capital bonus to outgoing enemy threat.
	gBot := newGame("BOTCAP")
	bot := gBot.addPlayer("Bot", true)
	e1 := gBot.addPlayer("Enemy1", false)
	e2 := gBot.addPlayer("Enemy2", false)
	for i := range gBot.Territories {
		gBot.Territories[i].Owner = e2.ID
	}
	gBot.Territories[7].Owner = bot.ID
	bot.Capital = 7
	gBot.Territories[7].IsCapital = true
	gBot.Territories[1].Owner = e1.ID
	gBot.Territories[13].Owner = e2.ID
	e2.Capital = 13
	gBot.Territories[13].IsCapital = true
	gBot.Units = []*Unit{
		{ID: "botCap", Owner: bot.ID, Type: "infantry", Territory: 7, Count: 6, Health: 100, Ration: 1},
		{ID: "e1Weak", Owner: e1.ID, Type: "infantry", Territory: 1, Count: 2, Health: 100, Ration: 1},
		{ID: "e2Strong", Owner: e2.ID, Type: "infantry", Territory: 13, Count: 10, Health: 100, Ration: 1},
	}
	gBot.Phase = "plan"
	// sectorThreat on 7 from e1 (4) + e2 (20, NOT 23) = 24.
	if math.Abs(gBot.sectorThreat(7, bot.ID)-24.0) > 1e-6 {
		t.Fatalf("sectorThreat inflated enemy capital offensive power: got %.2f, want 24.00", gBot.sectorThreat(7, bot.ID))
	}
	gBot.botOrders()
	if ord := gBot.Orders["botCap"]; ord.Kind != "defend" {
		t.Fatalf("Bot abandoned its own threatened capital to attack weak neighbor: %+v", ord)
	}

	// Bot seizes undefended enemy capital before fighting weak garrisoned non-capital.
	gFreeCap := newGame("FREECAP")
	bot2 := gFreeCap.addPlayer("Bot", true)
	foe := gFreeCap.addPlayer("Foe", false)
	for i := range gFreeCap.Territories {
		gFreeCap.Territories[i].Owner = bot2.ID
	}
	bot2.Capital = 0
	gFreeCap.Territories[0].IsCapital = true
	gFreeCap.Territories[1].Owner = foe.ID
	foe.Capital = 1
	gFreeCap.Territories[1].IsCapital = true
	gFreeCap.Territories[13].Owner = foe.ID
	gFreeCap.Units = []*Unit{
		{ID: "bStack", Owner: bot2.ID, Type: "infantry", Territory: 7, Count: 6, Health: 100, Ration: 1},
		{ID: "fWeak", Owner: foe.ID, Type: "infantry", Territory: 13, Count: 1, Health: 100, Ration: 1},
	}
	gFreeCap.Phase = "plan"
	gFreeCap.botOrders()
	if ord := gFreeCap.Orders["bStack"]; ord.Kind != "move" || ord.Target != 1 {
		t.Fatalf("Bot failed to prioritize undefended enemy capital at 1: %+v", ord)
	}

	// 5. Mid-simulate surrender resets Ready so exit frame isn't skipped immediately.
	gSim := newGame("SIMEXIT")
	sA := gSim.addPlayer("A", false)
	sB := gSim.addPlayer("B", false)
	sC := gSim.addPlayer("C", false)
	gSim.Territories[0].Owner = sA.ID
	gSim.Territories[1].Owner = sB.ID
	gSim.Territories[2].Owner = sC.ID
	gSim.Phase = "simulate"
	sA.Ready = true
	sB.Ready = true
	if err := gSim.leave(sC.ID); err != nil {
		t.Fatal(err)
	}
	if gSim.Phase != "simulate" || sA.Ready || sB.Ready {
		t.Fatalf("Mid-simulate surrender skipped exit frame or kept Ready: phase=%s readyA=%v readyB=%v", gSim.Phase, sA.Ready, sB.Ready)
	}

	// 6. Invalid create does not leak empty game into hub, and nil/empty guards work.
	h := &hub{games: map[string]*Game{}, clients: map[*client]bool{}}
	c := &client{out: make(chan any, 8), seen: map[string]bool{}}
	h.handle(c, Command{ID: "bad-create", Action: "create", Name: "   "})
	if len(h.games) != 0 {
		t.Fatalf("Invalid create leaked empty game into hub: %d", len(h.games))
	}
	if !g3.eliminated("") {
		t.Fatal("eliminated(\"\") must be true")
	}
	if err := g3.chooseCapital("missing", 0); err == nil {
		t.Fatal("chooseCapital did not reject unknown player")
	}
}
