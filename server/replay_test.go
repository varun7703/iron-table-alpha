package main

import (
	"math"
	"reflect"
	"testing"
)

func TestBombardmentAndBorderReplay(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	g.Units = []*Unit{{"a", a.ID, "tank", 6, 5, 100, 1, false}, {"b", b.ID, "tank", 12, 3, 100, 1, false}, {"art", b.ID, "artillery", 12, 4, 100, 1, false}}
	g.Orders["a"] = Order{"a", "move", 12}
	g.Orders["b"] = Order{"b", "move", 6}
	g.Orders["art"] = Order{"art", "bombard", 6}
	g.resolve()
	if len(g.Timeline) < 5 {
		t.Fatal("Missing resolution stages")
	}
	// Order of stages: 0 Orders revealed, 1 Border clashes, 2 Movement, 3 Bombardment, 4 Retreats/final.
	// Border clash: a (5 tanks) takes 3*6=18 dmg (0 kills @ 20 HP, Health 100 - 18/100*60 = 89.20); both blocked.
	// Bombardment: art (4 artillery @ x1.5 vs tank = 24 dmg) hits a -> 1 kill @ 20 HP, Health 89.20 - 24/100*60 = 74.80.
	checks := []struct {
		frame  int
		count  int
		health float64
	}{{0, 5, 100}, {1, 5, 89.2}, {2, 5, 89.2}, {3, 4, 74.8}}
	for _, c := range checks {
		u := g.Timeline[c.frame].Units[0]
		if u.Count != c.count || math.Abs(u.Health-c.health) > .0001 {
			t.Fatalf("Wrong replay state at %d: %+v", c.frame, u)
		}
	}
	final := g.Timeline[len(g.Timeline)-1]
	if len(final.Units) != len(g.Units) {
		t.Fatal("Final replay differs from game")
	}
	for i, u := range g.Units {
		if !reflect.DeepEqual(*u, final.Units[i]) {
			t.Fatal("Final unit mismatch")
		}
	}
	if math.Abs(attack(g.unit("a"), "tank", true)-17.952) > .0001 {
		t.Fatalf("Strength does not reflect damage: got %.4f", attack(g.unit("a"), "tank", true))
	}
	g.unit("a").Health = 1
	if g.Timeline[0].Units[0].Health != 100 {
		t.Fatal("Replay aliases live state")
	}
	g.Phase = "gather"
	if len(g.view(a.ID)["timeline"].([]BattleFrame)) != 0 {
		t.Fatal("Replay leaked through fog")
	}
}
