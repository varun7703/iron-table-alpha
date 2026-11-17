package main

import "testing"

func TestUpdatedEquipmentCosts(t *testing.T) {
	for _, tc := range []struct {
		kind           string
		currency, iron int
	}{{"artillery", 2, 4}, {"tank", 4, 8}, {"infantry", 4, 0}} {
		c := cost(tc.kind, 3)
		if c.Humans != 3 || c.Currency != tc.currency*3 || c.Iron != tc.iron*3 {
			t.Fatalf("Wrong %s cost: %+v", tc.kind, c)
		}
		g := fixture()
		g.Phase = "gather"
		p := g.Players[0]
		g.Units = []*Unit{{ID: "army", Owner: p.ID, Type: tc.kind, Territory: 0, Count: 1, Health: 100, Ration: 1}}
		before := p.Resources
		if err := g.command(p.ID, Command{Action: "reinforce", Unit: "army", Count: 1}); err != nil {
			t.Fatal(err)
		}
		if p.Resources.Currency != before.Currency-tc.currency || p.Resources.Iron != before.Iron-tc.iron {
			t.Fatal("Reinforcement cost mismatch")
		}
	}
}
