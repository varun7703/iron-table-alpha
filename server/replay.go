package main

type BattleFrame struct {
	Label  string   `json:"label"`
	Units  []Unit   `json:"units"`
	Owners []string `json:"owners"`
	Events []Event  `json:"events"`
}

func (g *Game) recordFrame(label string) {
	f := BattleFrame{Label: label, Units: []Unit{}, Owners: []string{}, Events: append([]Event{}, g.Events...)}
	for _, u := range g.Units {
		if u.Count > 0 {
			f.Units = append(f.Units, *u)
		}
	}
	for _, t := range g.Territories {
		f.Owners = append(f.Owners, t.Owner)
	}
	g.Timeline = append(g.Timeline, f)
}
