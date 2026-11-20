package main

type BattleFrame struct {
	Label     string   `json:"label"`
	Units     []Unit   `json:"units"`
	Owners    []string `json:"owners"`
	Events    []Event  `json:"events"`
	Destroyed []string `json:"destroyed,omitempty"`
}

func (g *Game) recordFrame(label string) {
	f := BattleFrame{Label: label, Units: []Unit{}, Owners: []string{}, Events: append([]Event{}, g.Events...)}
	alive := make(map[string]bool, len(g.Units))
	for _, u := range g.Units {
		if u.Count > 0 {
			f.Units = append(f.Units, *u)
			alive[u.ID] = true
		}
	}
	if len(g.Timeline) > 0 {
		prev := g.Timeline[len(g.Timeline)-1]
		for _, pu := range prev.Units {
			if !alive[pu.ID] {
				f.Destroyed = append(f.Destroyed, pu.ID)
			}
		}
	}
	for _, t := range g.Territories {
		f.Owners = append(f.Owners, t.Owner)
	}
	g.Timeline = append(g.Timeline, f)
}
