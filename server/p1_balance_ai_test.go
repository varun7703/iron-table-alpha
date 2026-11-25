package main

import (
	"fmt"
	"math"
	"testing"
)

func TestCombinedArmsTriangleAndUnitHP(t *testing.T) {
	// 1. Artillery (16 Iron = 4 pieces) counters Tanks (16 Iron = 2 pieces @ 20 HP): deals 24 dmg -> kills 1 Tank (50%).
	gArt := fixture()
	a1, b1 := gArt.Players[0], gArt.Players[1]
	gArt.Units = []*Unit{
		{ID: "art", Owner: a1.ID, Type: "artillery", Territory: 6, Count: 4, Health: 100, Ration: 1},
		{ID: "tnk", Owner: b1.ID, Type: "tank", Territory: 12, Count: 2, Health: 100, Ration: 1},
	}
	gArt.Orders["art"] = Order{"art", "bombard", 12}
	gArt.resolve()
	if gArt.unit("tnk").Count != 1 || math.Abs(gArt.unit("tnk").Health-64.0) > 0.01 {
		t.Fatalf("Artillery failed to counter equal-iron Tanks: %+v", gArt.unit("tnk"))
	}

	// 2. Tanks beat open-field Infantry: 2 Tanks beat 6 unentrenched Infantry.
	gOpen := fixture()
	a2, b2 := gOpen.Players[0], gOpen.Players[1]
	gOpen.Units = []*Unit{
		{ID: "tnk", Owner: a2.ID, Type: "tank", Territory: 6, Count: 2, Health: 100, Ration: 1},
		{ID: "inf", Owner: b2.ID, Type: "infantry", Territory: 12, Count: 6, Health: 100, Ration: 1},
	}
	gOpen.Orders["tnk"] = Order{"tnk", "move", 12}
	gOpen.resolve()
	if gOpen.Territories[12].Owner != a2.ID {
		t.Fatal("2 Tanks did not beat 6 open-field Infantry")
	}

	// 3. Entrenched Infantry holds off equal-manpower Tanks: 6 defending Infantry repel 2 Tanks.
	gDef := fixture()
	a3, b3 := gDef.Players[0], gDef.Players[1]
	gDef.Units = []*Unit{
		{ID: "tnk", Owner: a3.ID, Type: "tank", Territory: 6, Count: 2, Health: 100, Ration: 1},
		{ID: "inf", Owner: b3.ID, Type: "infantry", Territory: 12, Count: 6, Health: 100, Ration: 1},
	}
	gDef.Orders["tnk"] = Order{"tnk", "move", 12}
	gDef.Orders["inf"] = Order{"inf", "defend", 12}
	gDef.resolve()
	if gDef.Territories[12].Owner != b3.ID {
		t.Fatal("6 entrenched Infantry did not hold against 2 Tanks")
	}

	// 4. Infantry overruns undefended Artillery even after post-movement bombardment.
	gInf := fixture()
	a4, b4 := gInf.Players[0], gInf.Players[1]
	gInf.Units = []*Unit{
		{ID: "inf", Owner: a4.ID, Type: "infantry", Territory: 6, Count: 6, Health: 100, Ration: 1},
		{ID: "art", Owner: b4.ID, Type: "artillery", Territory: 12, Count: 4, Health: 100, Ration: 1},
	}
	gInf.Orders["inf"] = Order{"inf", "move", 12}
	gInf.Orders["art"] = Order{"art", "bombard", 6}
	gInf.resolve()
	if gInf.Territories[12].Owner != a4.ID {
		t.Fatal("6 Infantry failed to overrun 4 undefended Artillery")
	}

	// 5. HP-weighted damage distribution remains split-invariant for mixed stacks.
	runMixed := func(infStacks int) (int, int) {
		g := fixture()
		pa, pb := g.Players[0], g.Players[1]
		g.Units = []*Unit{
			{ID: "at", Owner: pa.ID, Type: "tank", Territory: 6, Count: 2, Health: 100, Ration: 1},
			{ID: "bt", Owner: pb.ID, Type: "tank", Territory: 12, Count: 5, Health: 100, Ration: 1},
		}
		g.Orders["at"] = Order{"at", "move", 12}
		per := 6 / infStacks
		for i := 0; i < infStacks; i++ {
			id := fmt.Sprintf("ai-%d", i)
			g.Units = append(g.Units, &Unit{ID: id, Owner: pa.ID, Type: "infantry", Territory: 6, Count: per, Health: 100, Ration: 1})
			g.Orders[id] = Order{id, "move", 12}
		}
		g.resolve()
		remTanks, remInf := 0, 0
		for _, u := range g.Units {
			if u.Owner == pa.ID {
				if u.Type == "tank" {
					remTanks += u.Count
				} else {
					remInf += u.Count
				}
			}
		}
		return remTanks, remInf
	}
	mt, mi := runMixed(1)
	st, si := runMixed(6)
	if mt != st || mi != si {
		t.Fatalf("Mixed-type HP-weighted split diverged: merged=(%dT,%dI) split=(%dT,%dI)", mt, mi, st, si)
	}
}

func TestBacklineArtilleryShellsIncomingAttackers(t *testing.T) {
	// Sector 0 (backline) -> adjacent to Sector 6 (frontline) -> adjacent to Sector 12 (enemy).
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	g.Units = []*Unit{
		{ID: "back-art", Owner: a.ID, Type: "artillery", Territory: 0, Count: 4, Health: 100, Ration: 1},
		{ID: "front-inf", Owner: a.ID, Type: "infantry", Territory: 6, Count: 6, Health: 100, Ration: 1},
		{ID: "inv-tnk", Owner: b.ID, Type: "tank", Territory: 12, Count: 2, Health: 100, Ration: 1},
	}
	g.Orders["back-art"] = Order{"back-art", "bombard", 6}
	g.Orders["front-inf"] = Order{"front-inf", "defend", 6}
	g.Orders["inv-tnk"] = Order{"inv-tnk", "move", 6}
	g.resolve()

	// Frame 2 is Movement (inv-tnk arrives at 6); Frame 3 is Bombardment (back-art shells inv-tnk at 6, 0 friendly fire).
	var bombFrame *BattleFrame
	for i := range g.Timeline {
		if g.Timeline[i].Label == "Bombardment resolved · simultaneous artillery damage" {
			bombFrame = &g.Timeline[i]
			break
		}
	}
	if bombFrame == nil {
		t.Fatal("Missing bombardment timeline frame")
	}
	var frontAfterBomb, invAfterBomb *Unit
	for i := range bombFrame.Units {
		u := &bombFrame.Units[i]
		if u.ID == "front-inf" {
			frontAfterBomb = u
		}
		if u.ID == "inv-tnk" {
			invAfterBomb = u
		}
	}
	if frontAfterBomb == nil || frontAfterBomb.Count != 6 || frontAfterBomb.Health != 100 {
		t.Fatalf("Backline artillery caused friendly fire: %+v", frontAfterBomb)
	}
	if invAfterBomb == nil || invAfterBomb.Count != 1 {
		t.Fatalf("Backline artillery did not shell incoming attackers after movement: %+v", invAfterBomb)
	}
	if g.Territories[6].Owner != a.ID {
		t.Fatal("Frontline garrison + backline artillery failed to hold sector 6")
	}
}

func TestDefendEntrenchmentAndBombardResistance(t *testing.T) {
	// Compare bombardment damage against unordered vs entrenched Infantry.
	runBombard := func(defend bool) (int, float64) {
		g := fixture()
		a, b := g.Players[0], g.Players[1]
		g.Units = []*Unit{
			{ID: "art", Owner: a.ID, Type: "artillery", Territory: 6, Count: 4, Health: 100, Ration: 1},
			{ID: "inf", Owner: b.ID, Type: "infantry", Territory: 12, Count: 6, Health: 100, Ration: 1},
		}
		g.Orders["art"] = Order{"art", "bombard", 12}
		if defend {
			g.Orders["inf"] = Order{"inf", "defend", 12}
		}
		g.resolve()
		u := g.unit("inf")
		return u.Count, u.Health
	}
	// Unentrenched: 4 Art * 4 * 1.2 = 19.2 dmg -> 1 kill, Health = 100 - 19.2/60*60 = 80.8.
	// Entrenched (-25%): 19.2 * 0.75 = 14.4 dmg -> 1 kill, Health = 100 - 14.4/60*60 = 85.6.
	cPlain, hPlain := runBombard(false)
	cDef, hDef := runBombard(true)
	if cPlain != 5 || math.Abs(hPlain-80.8) > 0.01 {
		t.Fatalf("Unexpected plain bombardment result: count=%d health=%.2f", cPlain, hPlain)
	}
	if cDef != 5 || math.Abs(hDef-85.6) > 0.01 {
		t.Fatalf("Unexpected entrenched bombardment resistance: count=%d health=%.2f", cDef, hDef)
	}

	// Verify command("order", defend) normalizes Target to unit's current territory.
	g := fixture()
	a := g.Players[0]
	g.Units = []*Unit{{ID: "u", Owner: a.ID, Type: "infantry", Territory: 3, Count: 4, Health: 100, Ration: 1}}
	if err := g.command(a.ID, Command{Action: "order", Unit: "u", Kind: "defend", Target: 99}); err != nil {
		t.Fatal(err)
	}
	if g.Orders["u"].Target != 3 {
		t.Fatalf("Defend order did not normalize target to unit territory: %+v", g.Orders["u"])
	}
}

func TestRepeatingDraftOrderAndMidDraftSurrender(t *testing.T) {
	cases := []struct {
		players int
		want    []int
	}{
		{2, []int{0, 1, 0, 1, 0, 1, 0, 1}},
		{3, []int{0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2}},
		{4, []int{0, 1, 2, 3, 0, 1, 2, 3, 0, 1, 2, 3, 0, 1, 2, 3}},
	}
	for _, tc := range cases {
		g := newGame("SNAKE")
		for i := 0; i < tc.players; i++ {
			g.addPlayer(fmt.Sprintf("P%d", i), false)
		}
		g.setPhase("draft", 25)
		got := []int{}
		for g.Phase == "draft" {
			cur := g.draftPlayer()
			idx := -1
			for i, p := range g.Players {
				if p.ID == cur.ID {
					idx = i
					break
				}
			}
			got = append(got, idx)
			g.draftOne()
		}
		if len(got) != len(tc.want) {
			t.Fatalf("%dp repeating draft length mismatch: got %v want %v", tc.players, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%dp repeating draft mismatch at pick %d: got %v want %v", tc.players, i, got, tc.want)
			}
		}
	}

	// Verify 4-player repeating draft remains valid under mid-draft surrender at any pick & seat index.
	for _, leaveSeat := range []int{0, 1, 3} {
		for leaveAt := 0; leaveAt < 16; leaveAt++ {
			g := newGame("SNKLV")
			for i := 0; i < 4; i++ {
				g.addPlayer(fmt.Sprintf("P%d", i), false)
			}
			g.setPhase("draft", 25)
			for i := 0; i < leaveAt && g.Phase == "draft"; i++ {
				g.draftOne()
			}
			if g.Phase != "draft" {
				continue
			}
			if err := g.leave(g.Players[leaveSeat].ID); err != nil {
				t.Fatalf("leaveSeat=%d leaveAt=%d failed: %v", leaveSeat, leaveAt, err)
			}
			for steps := 0; steps < 20 && g.Phase == "draft"; steps++ {
				g.draftOne()
			}
			if g.Phase != "capital" {
				t.Fatalf("4p repeating draft stalled after leaveSeat=%d leaveAt=%d (phase=%s)", leaveSeat, leaveAt, g.Phase)
			}
			for _, p := range g.Players {
				owned := 0
				for _, tr := range g.Territories {
					if tr.Owner == p.ID {
						owned++
					}
				}
				if owned != 4 {
					t.Fatalf("Player %s ended with %d tiles after leaveSeat=%d leaveAt=%d", p.Name, owned, leaveSeat, leaveAt)
				}
			}
		}
	}
}

func TestStartingStockpileAndFreeFriendlyMovement(t *testing.T) {
	g := newGame("ECON")
	a := g.addPlayer("Alpha", false)
	b := g.addPlayer("Bravo", false)
	want := Resources{Humans: 16, Food: 40, Iron: 16, Currency: 60}
	if a.Resources != want {
		t.Fatalf("Starting stockpile mismatch: got %+v want %+v", a.Resources, want)
	}
	g.Territories[0].Owner = a.ID
	g.Territories[1].Owner = a.ID
	g.Territories[2].Owner = "" // neutral
	g.Territories[6].Owner = b.ID
	g.Round = 1
	g.setPhase("plan", 90)
	a.Resources.Currency = 6

	uFriend := &Unit{ID: "f", Owner: a.ID, Type: "tank", Territory: 0, Count: 5, Health: 100, Ration: 1}
	uNeutral := &Unit{ID: "n", Owner: a.ID, Type: "infantry", Territory: 1, Count: 4, Health: 100, Ration: 1}
	g.Units = []*Unit{uFriend, uNeutral}

	// Friendly move costs 0 even for 5 tanks (would be 15 if paid).
	if err := g.command(a.ID, Command{Action: "order", Unit: "f", Kind: "move", Target: 1}); err != nil {
		t.Fatalf("Free friendly transfer failed: %v", err)
	}
	if g.reserved(a.ID, "") != 0 {
		t.Fatalf("Friendly transfer reserved %d currency", g.reserved(a.ID, ""))
	}
	// Neutral move costs 4 currency (4 infantry * 1).
	if err := g.command(a.ID, Command{Action: "order", Unit: "n", Kind: "move", Target: 2}); err != nil {
		t.Fatalf("Neutral move failed: %v", err)
	}
	if g.reserved(a.ID, "") != 4 {
		t.Fatalf("Neutral move reserved %d currency, want 4", g.reserved(a.ID, ""))
	}
	g.resolve()
	if a.Resources.Currency != 2 || uFriend.Territory != 1 || uNeutral.Territory != 2 {
		t.Fatalf("Unexpected post-resolve state: currency=%d friend=%d neutral=%d", a.Resources.Currency, uFriend.Territory, uNeutral.Territory)
	}
}

func TestBotSmartDraftGatherAndThreatAwareOrders(t *testing.T) {
	// 1. Bot drafts balanced Food + Iron contiguous cluster and picks a valid capital.
	gDraft := newGame("BDRAFT")
	b1 := gDraft.addPlayer("Bot 1", true)
	b2 := gDraft.addPlayer("Bot 2", true)
	gDraft.setPhase("draft", 25)
	gDraft.autoDraft()
	for _, bot := range []*Player{b1, b2} {
		food, iron, adjPairs := 0, 0, 0
		owned := map[int]bool{}
		for _, tr := range gDraft.Territories {
			if tr.Owner == bot.ID {
				owned[tr.ID] = true
				if tr.Resource == "food" {
					food++
				} else {
					iron++
				}
			}
		}
		for tid := range owned {
			for _, n := range gDraft.Territories[tid].Neighbors {
				if owned[n] {
					adjPairs++
				}
			}
		}
		if food == 0 || iron == 0 {
			t.Fatalf("Bot %s drafted unbalanced resources: food=%d iron=%d", bot.Name, food, iron)
		}
		if adjPairs == 0 {
			t.Fatalf("Bot %s drafted 4 completely disconnected sectors", bot.Name)
		}
		if !owned[bot.Capital] {
			t.Fatalf("Bot %s capital %d not owned", bot.Name, bot.Capital)
		}
	}

	// 2. Bot Gather recruits combined arms, merges duplicate unfed stacks, preserves currency reserve, and avoids starvation.
	gGather := fixture()
	bot := gGather.Players[0]
	bot.Bot = true
	enemy := gGather.Players[1]
	bot.Resources = Resources{Humans: 40, Food: 60, Iron: 40, Currency: 120}
	gGather.Units = []*Unit{
		{ID: "inf1", Owner: bot.ID, Type: "infantry", Territory: 6, Count: 4, Health: 60, Ration: 1, Fed: false},
		{ID: "inf2", Owner: bot.ID, Type: "infantry", Territory: 6, Count: 4, Health: 100, Ration: 1, Fed: false},
		{ID: "etnk", Owner: enemy.ID, Type: "tank", Territory: 12, Count: 3, Health: 100, Ration: 1, Fed: true},
	}
	gGather.botGather(bot)

	typesSeen := map[string]int{}
	infStacksAt6 := 0
	for _, u := range gGather.Units {
		if u.Owner != bot.ID {
			continue
		}
		typesSeen[u.Type] += u.Count
		if u.Territory == 6 && u.Type == "infantry" {
			infStacksAt6++
		}
		if u.Ration == 0 {
			t.Fatalf("Bot starved its own army during botGather: %+v", u)
		}
	}
	if infStacksAt6 != 1 {
		t.Fatalf("Bot failed to merge duplicate infantry stacks on tile 6: stacks=%d", infStacksAt6)
	}
	if typesSeen["artillery"] == 0 || typesSeen["tank"] == 0 {
		t.Fatalf("Bot did not recruit combined arms: %+v", typesSeen)
	}
	if bot.Resources.Currency < 36 {
		t.Fatalf("Bot violated 30%% currency movement reserve: remaining=%d", bot.Resources.Currency)
	}

	// 3. Bot Plan orders: entrenches outgunned front, bombards enemy tanks, pushes rear troops forward for free.
	gPlan := fixture()
	pBot := gPlan.Players[0]
	pBot.Bot = true
	pEnemy := gPlan.Players[1]
	gPlan.Units = []*Unit{
		{ID: "front-outgunned", Owner: pBot.ID, Type: "infantry", Territory: 6, Count: 4, Health: 100, Ration: 1},
		{ID: "back-art", Owner: pBot.ID, Type: "artillery", Territory: 0, Count: 3, Health: 100, Ration: 1},
		{ID: "rear-tank", Owner: pBot.ID, Type: "tank", Territory: 1, Count: 2, Health: 100, Ration: 1},
		{ID: "enemy-armour", Owner: pEnemy.ID, Type: "tank", Territory: 12, Count: 6, Health: 100, Ration: 1},
	}
	gPlan.botOrders()
	if o := gPlan.Orders["front-outgunned"]; o.Kind != "defend" {
		t.Fatalf("Outgunned frontline bot failed to entrench: %+v", o)
	}
	if o := gPlan.Orders["back-art"]; o.Kind != "bombard" || o.Target != 6 {
		t.Fatalf("Backline bot artillery did not provide defensive support fire on threatened sector 6: %+v", o)
	}
	if o := gPlan.Orders["rear-tank"]; o.Kind != "move" || gPlan.Territories[o.Target].Owner != pBot.ID {
		t.Fatalf("Rear bot tank did not advance along free friendly route: %+v", o)
	}

	// 4. Bot attacks adjacent enemy when projected strength >= 1.15x garrison.
	gAtk := fixture()
	bAtk := gAtk.Players[0]
	bAtk.Bot = true
	eDef := gAtk.Players[1]
	gAtk.Units = []*Unit{
		{ID: "strike", Owner: bAtk.ID, Type: "tank", Territory: 6, Count: 4, Health: 100, Ration: 1},
		{ID: "weak", Owner: eDef.ID, Type: "infantry", Territory: 12, Count: 4, Health: 100, Ration: 1},
	}
	gAtk.botOrders()
	if o := gAtk.Orders["strike"]; o.Kind != "move" || o.Target != 12 {
		t.Fatalf("Bot with overwhelming odds failed to attack sector 12: %+v", o)
	}

	// 5. Multi-round 4-bot campaign exercises real combined-arms recruitment (Tanks + Artillery + Infantry).
	gSim := newGame("BSIM")
	for i := 0; i < 4; i++ {
		gSim.addPlayer(fmt.Sprintf("Bot %d", i+1), true)
	}
	gSim.setPhase("draft", 25)
	gSim.autoDraft()
	seenTank, seenArt := false, false
	for step := 0; step < 18 && gSim.Phase != "finished"; step++ {
		gSim.advance()
		for _, u := range gSim.Units {
			if u.Type == "tank" {
				seenTank = true
			}
			if u.Type == "artillery" {
				seenArt = true
			}
		}
	}
	if !seenTank || !seenArt {
		t.Fatalf("Multi-round bot campaign failed to recruit combined arms: tank=%v artillery=%v", seenTank, seenArt)
	}
}

func TestAdversarialCombatAndOrderInvariants(t *testing.T) {
	// 1. CRIT-1: Overkill on border 6-12 must NOT bleed into healthy army clashing on border 6-7.
	gBorder := fixture()
	a, b := gBorder.Players[0], gBorder.Players[1]
	gBorder.Territories[7].Owner = b.ID
	gBorder.Units = []*Unit{
		{ID: "u1", Owner: a.ID, Type: "infantry", Territory: 6, Count: 1, Health: 100, Ration: 1},
		{ID: "u2", Owner: a.ID, Type: "infantry", Territory: 6, Count: 5, Health: 100, Ration: 1},
		{ID: "e1", Owner: b.ID, Type: "tank", Territory: 12, Count: 10, Health: 100, Ration: 1},
		{ID: "e2", Owner: b.ID, Type: "infantry", Territory: 7, Count: 1, Health: 100, Ration: 1},
	}
	gBorder.Orders["u1"] = Order{"u1", "move", 12}
	gBorder.Orders["u2"] = Order{"u2", "move", 7}
	gBorder.Orders["e1"] = Order{"e1", "move", 6}
	gBorder.Orders["e2"] = Order{"e2", "move", 6}
	gBorder.resolve()
	if gBorder.unit("u1") != nil {
		t.Fatal("u1 should have been wiped by 10 tanks")
	}
	if gBorder.unit("u2") == nil || gBorder.unit("u2").Count != 5 {
		t.Fatalf("Overkill from border 6-12 bled onto unrelated border 6-7 army: %+v", gBorder.unit("u2"))
	}

	// 2. HIGH-2: Stage 3 bombardment overkill on unentrenched movers must NOT bleed onto entrenched same-type garrison.
	gBomb := fixture()
	pa, pb := gBomb.Players[0], gBomb.Players[1]
	gBomb.Units = []*Unit{
		{ID: "art", Owner: pa.ID, Type: "artillery", Territory: 6, Count: 15, Health: 100, Ration: 1},
		{ID: "infMove", Owner: pb.ID, Type: "infantry", Territory: 13, Count: 5, Health: 100, Ration: 1},
		{ID: "infDef", Owner: pb.ID, Type: "infantry", Territory: 12, Count: 1, Health: 100, Ration: 1},
	}
	gBomb.Orders["art"] = Order{"art", "bombard", 12}
	gBomb.Orders["infMove"] = Order{"infMove", "move", 12}
	gBomb.Orders["infDef"] = Order{"infDef", "defend", 12}
	gBomb.resolve()
	if gBomb.unit("infDef") == nil || gBomb.unit("infDef").Count != 1 {
		t.Fatalf("Unentrenched bombardment overkill bled onto entrenched stack: %+v", gBomb.unit("infDef"))
	}

	// 3. HIGH-3: 1% HP zombie infantry cannot meatshield healthy tanks from enemy fire.
	gMeat := fixture()
	mA, mB := gMeat.Players[0], gMeat.Players[1]
	gMeat.Units = []*Unit{
		{ID: "atk", Owner: mA.ID, Type: "tank", Territory: 6, Count: 10, Health: 100, Ration: 1},
		{ID: "zombie", Owner: mB.ID, Type: "infantry", Territory: 12, Count: 10, Health: 1, Ration: 0},
		{ID: "shielded", Owner: mB.ID, Type: "tank", Territory: 12, Count: 2, Health: 100, Ration: 1},
	}
	gMeat.Orders["atk"] = Order{"atk", "move", 12}
	gMeat.resolve()
	if gMeat.unit("shielded") != nil {
		t.Fatalf("1%% HP infantry meatshielded healthy tanks from 10 attacking tanks: %+v", gMeat.unit("shielded"))
	}

	// 4. HIGH-4: 3-way tie above incumbent must leave incumbent's garrison alive in its own sector.
	gTie := newGame("TIE3")
	tA := gTie.addPlayer("Incumbent", false)
	tB := gTie.addPlayer("AttackerB", false)
	tC := gTie.addPlayer("AttackerC", false)
	for i := range gTie.Territories {
		gTie.Territories[i].Owner = tB.ID
	}
	gTie.Territories[0].Owner = tA.ID // isolated sole tile of A
	gTie.Territories[6].Owner = tC.ID
	gTie.Units = []*Unit{
		{ID: "a", Owner: tA.ID, Type: "infantry", Territory: 0, Count: 2, Health: 100, Ration: 1},
		{ID: "b", Owner: tB.ID, Type: "infantry", Territory: 1, Count: 6, Health: 100, Ration: 1},
		{ID: "c", Owner: tC.ID, Type: "infantry", Territory: 6, Count: 6, Health: 100, Ration: 1},
	}
	gTie.Orders["b"] = Order{"b", "move", 0}
	gTie.Orders["c"] = Order{"c", "move", 0}
	gTie.resolve()
	if gTie.Territories[0].Owner != tA.ID || gTie.unit("a") == nil || gTie.unit("a").Territory != 0 {
		t.Fatalf("3-way tie above incumbent deleted/evicted incumbent garrison: owner=%s unit=%+v", gTie.Territories[0].Owner, gTie.unit("a"))
	}
	if gTie.unit("b").Territory != 1 || gTie.unit("c").Territory != 6 {
		t.Fatalf("Tied attackers did not retreat: b=%d c=%d", gTie.unit("b").Territory, gTie.unit("c").Territory)
	}

	// 5. MED-1: Overrun artillery in Stage 2 is suppressed from bombarding in Stage 3; Artillery has 14 HP durability.
	gOver := fixture()
	oA, oB := gOver.Players[0], gOver.Players[1]
	gOver.Territories[18].Owner = oA.ID
	gOver.Units = []*Unit{
		{ID: "rush", Owner: oA.ID, Type: "infantry", Territory: 6, Count: 6, Health: 100, Ration: 1},
		{ID: "bystander", Owner: oA.ID, Type: "infantry", Territory: 18, Count: 4, Health: 100, Ration: 1},
		{ID: "exposed-art", Owner: oB.ID, Type: "artillery", Territory: 12, Count: 4, Health: 100, Ration: 1},
	}
	gOver.Orders["rush"] = Order{"rush", "move", 12}
	gOver.Orders["exposed-art"] = Order{"exposed-art", "bombard", 18}
	gOver.resolve()
	if gOver.unit("bystander").Count != 4 || gOver.unit("bystander").Health != 100 {
		t.Fatalf("Overrun artillery fired Stage 3 bombardment despite melee engagement: %+v", gOver.unit("bystander"))
	}

	gArt14 := fixture()
	aa, ab := gArt14.Players[0], gArt14.Players[1]
	gArt14.Units = []*Unit{
		{ID: "a1", Owner: aa.ID, Type: "artillery", Territory: 6, Count: 4, Health: 100, Ration: 1},
		{ID: "b1", Owner: ab.ID, Type: "artillery", Territory: 12, Count: 2, Health: 100, Ration: 1},
	}
	gArt14.Orders["a1"] = Order{"a1", "bombard", 12}
	gArt14.resolve()
	// 4 Art * 4 * 1.0 = 16 dmg -> 1 kill @ 14 HP!
	if gArt14.unit("b1").Count != 1 {
		t.Fatalf("Artillery 14 HP durability mismatch: %+v", gArt14.unit("b1"))
	}

	// 6. HIGH-1: Orders are cleared on simulate -> gather, on finish(), and on mid-simulate surrender.
	gOrd := fixture()
	p0, p1 := gOrd.Players[0], gOrd.Players[1]
	gOrd.Units = []*Unit{{ID: "m", Owner: p0.ID, Type: "infantry", Territory: 0, Count: 4, Health: 100, Ration: 1}}
	gOrd.Orders["m"] = Order{"m", "move", 1}
	gOrd.advance() // plan -> simulate
	if len(gOrd.view(p0.ID)["orders"].(map[string]Order)) != 1 {
		t.Fatal("Simulate should show committed orders")
	}
	gOrd.advance() // simulate -> gather
	if len(gOrd.view(p0.ID)["orders"].(map[string]Order)) != 0 {
		t.Fatal("Stale orders leaked into Gather view")
	}
	gOrd.Round = 20
	gOrd.Phase = "plan"
	gOrd.Orders["m"] = Order{"m", "defend", 1}
	gOrd.advance() // plan -> simulate
	gOrd.advance() // simulate -> finished
	if len(gOrd.view(p0.ID)["orders"].(map[string]Order)) != 0 || len(gOrd.view(p1.ID)["orders"].(map[string]Order)) != 0 {
		t.Fatal("Stale orders leaked into Finished view")
	}
}

func TestBotEconomySustainabilityAndTargeting(t *testing.T) {
	// 1. Frontline artillery prioritizes real adjacent enemy stack over speculative empty friendly sector
	// and prioritizes a large healthy army over a 1-troop wounded tank straggler.
	g := fixture()
	bot, enemy := g.Players[0], g.Players[1]
	bot.Bot = true
	g.Territories[7].Owner = enemy.ID
	g.Territories[13].Owner = enemy.ID
	g.Units = []*Unit{
		{ID: "art", Owner: bot.ID, Type: "artillery", Territory: 6, Count: 4, Health: 100, Ration: 1},
		{ID: "big-inf", Owner: enemy.ID, Type: "infantry", Territory: 12, Count: 12, Health: 100, Ration: 1},
		{ID: "straggler", Owner: enemy.ID, Type: "tank", Territory: 7, Count: 1, Health: 10, Ration: 1},
		{ID: "far-tank", Owner: enemy.ID, Type: "tank", Territory: 13, Count: 2, Health: 100, Ration: 1},
	}
	g.botOrders()
	if o := g.Orders["art"]; o.Kind != "bombard" || o.Target != 12 {
		t.Fatalf("Artillery chose wrong target (want 12 big-inf): %+v", o)
	}

	// 2. Frontline bot protects stationed artillery & sends only 1 cheap stack to undefended neutral tile.
	g2 := fixture()
	b2, e2 := g2.Players[0], g2.Players[1]
	b2.Bot = true
	g2.Territories[7].Owner = "" // neutral
	g2.Units = []*Unit{
		{ID: "art", Owner: b2.ID, Type: "artillery", Territory: 6, Count: 2, Health: 100, Ration: 1},
		{ID: "tnk", Owner: b2.ID, Type: "tank", Territory: 6, Count: 3, Health: 100, Ration: 1},
		{ID: "inf", Owner: b2.ID, Type: "infantry", Territory: 6, Count: 2, Health: 100, Ration: 1},
		{ID: "threat", Owner: e2.ID, Type: "tank", Territory: 12, Count: 4, Health: 100, Ration: 1},
	}
	g2.botOrders()
	if g2.Orders["tnk"].Kind != "defend" {
		t.Fatalf("Heavy tank abandoned threatened artillery instead of defending: %+v", g2.Orders["tnk"])
	}
	if g2.Orders["inf"].Kind != "move" || g2.Orders["inf"].Target != 7 {
		t.Fatalf("Cheapest infantry stack did not claim neutral 7: %+v", g2.Orders["inf"])
	}

	// 3. Multi-campaign sustainability: across 20 full 4-bot campaigns, bots recruit combined arms
	// without burning their starting stockpile into mass self-starvation.
	totalUnitTurns, starvingUnitTurns := 0, 0
	for trial := 0; trial < 20; trial++ {
		sim := newGame("SUSTAIN")
		for i := 0; i < 4; i++ {
			sim.addPlayer(fmt.Sprintf("Bot %d", i+1), true)
		}
		sim.setPhase("draft", 25)
		sim.autoDraft()
		for step := 0; step < 65 && sim.Phase != "finished"; step++ {
			sim.advance()
			if sim.Phase == "plan" {
				for _, u := range sim.Units {
					totalUnitTurns += u.Count
					if u.Ration == 0 {
						starvingUnitTurns += u.Count
					}
				}
			}
		}
	}
	starveRate := float64(starvingUnitTurns) / float64(max(1, totalUnitTurns))
	if starveRate > 0.05 {
		t.Fatalf("Bots self-starved excessively across 20 campaigns: %.2f%% (%d/%d)", starveRate*100, starvingUnitTurns, totalUnitTurns)
	}
}
