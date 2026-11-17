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
	checks := []struct {
		frame  int
		count  int
		health float64
	}{{0, 5, 100}, {1, 4, 86.56}, {2, 2, 55.06}}
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
	if math.Abs(attack(g.unit("a"), "tank", true)-7.7084) > .0001 {
		t.Fatal("Strength does not reflect damage")
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
