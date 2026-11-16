package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	rng "math/rand/v2"
	"sort"
	"time"
)

type Resources struct {
	Humans   int `json:"humans"`
	Food     int `json:"food"`
	Iron     int `json:"iron"`
	Currency int `json:"currency"`
}
type Player struct {
	Eliminated bool      `json:"eliminated"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
	Roll       int       `json:"roll"`
	Ready      bool      `json:"ready"`
	Bot        bool      `json:"bot"`
	Resources  Resources `json:"resources"`
	Token      string    `json:"-"`
}
type Territory struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Owner     string  `json:"owner"`
	Resource  string  `json:"resource"`
	Yield     int     `json:"yield"`
	Humans    int     `json:"humans"`
	Currency  int     `json:"currency"`
	Neighbors []int   `json:"neighbors"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Points    string  `json:"points"`
}
type Unit struct {
	ID        string  `json:"id"`
	Owner     string  `json:"owner"`
	Type      string  `json:"type"`
	Territory int     `json:"territory"`
	Count     int     `json:"count"`
	Health    float64 `json:"health"`
	Ration    float64 `json:"ration"`
	Fed       bool    `json:"fed"`
}
type Order struct {
	Unit   string `json:"unit"`
	Kind   string `json:"kind"`
	Target int    `json:"target"`
}
type Event struct {
	Text string `json:"text"`
	From int    `json:"from"`
	To   int    `json:"to"`
	Kind string `json:"kind"`
}
type Game struct {
	Code        string           `json:"code"`
	Host        string           `json:"host"`
	Phase       string           `json:"phase"`
	Round       int              `json:"round"`
	Deadline    int64            `json:"deadline"`
	Players     []*Player        `json:"players"`
	Territories []Territory      `json:"territories"`
	Units       []*Unit          `json:"units"`
	Orders      map[string]Order `json:"orders"`
	Events      []Event          `json:"events"`
	Draft       int              `json:"draft"`
	Winner      string           `json:"winner"`
}
type Command struct {
	ID     string  `json:"id"`
	Action string  `json:"action"`
	Name   string  `json:"name"`
	Code   string  `json:"code"`
	Token  string  `json:"token"`
	Unit   string  `json:"unit"`
	Other  string  `json:"other"`
	Kind   string  `json:"kind"`
	Type   string  `json:"type"`
	Target int     `json:"target"`
	Count  int     `json:"count"`
	Ration float64 `json:"ration"`
}

var colors = []string{"#cd754e", "#6d9ea5", "#b9a264", "#9b89b0"}

func uid() string {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func newGame(code string) *Game {
	g := &Game{Code: code, Phase: "lobby", Orders: map[string]Order{}, Events: []Event{}, Units: []*Unit{}}
	g.Territories = makeMap()
	return g
}
func makeMap() []Territory {
	names := []string{"Northwatch", "Greyhaven", "Frostfield", "Highmere", "Eastguard", "Stormcape", "Westreach", "Oakridge", "Kingsfall", "Ironvale", "Redmoor", "Saltmarsh", "Lowhaven", "Greenfold", "Crossroads", "Ashford", "Blackstone", "Dawnport", "Southgate", "Wheatfield", "Emberhill", "Stonebridge", "Goldplain", "Seabrook"}
	// A shared, gently distorted vertex mesh keeps visual borders and adjacency aligned.
	type point struct{ x, y float64 }
	var mesh [5][7]point
	for r := 0; r < 5; r++ {
		for c := 0; c < 7; c++ {
			x := float64(c)*120 + 80
			y := float64(r)*116 + 80
			if c > 0 && c < 6 {
				x += float64((r*31+c*17)%35 - 17)
			}
			if r > 0 && r < 4 {
				y += float64((r*13+c*29)%31 - 15)
			}
			mesh[r][c] = point{x, y}
		}
	}
	out := []Territory{}
	for r := 0; r < 4; r++ {
		for c := 0; c < 6; c++ {
			i := r*6 + c
			p := []point{mesh[r][c], mesh[r][c+1], mesh[r+1][c+1], mesh[r+1][c]}
			s := ""
			x, y := 0.0, 0.0
			for _, v := range p {
				s += fmt.Sprintf("%.0f,%.0f ", v.x, v.y)
				x += v.x / 4
				y += v.y / 4
			}
			n := []int{}
			if c > 0 {
				n = append(n, i-1)
			}
			if c < 5 {
				n = append(n, i+1)
			}
			if r > 0 {
				n = append(n, i-6)
			}
			if r < 3 {
				n = append(n, i+6)
			}
			resource := "food"
			if (r+c)%2 == 1 {
				resource = "iron"
			}
			out = append(out, Territory{ID: i, Name: names[i], Resource: resource, Yield: 6 + rng.IntN(7), Humans: 3, Currency: 8, Neighbors: n, X: x, Y: y, Points: s})
		}
	}
	return out
}
func (g *Game) player(id string) *Player {
	for _, p := range g.Players {
		if p.ID == id {
			return p
		}
	}
	return nil
}
func (g *Game) unit(id string) *Unit {
	for _, u := range g.Units {
		if u.ID == id {
			return u
		}
	}
	return nil
}
func (g *Game) addPlayer(name string, bot bool) *Player {
	p := &Player{ID: uid(), Token: uid(), Name: name, Color: colors[len(g.Players)], Bot: bot, Resources: Resources{40, 120, 40, 160}}
	g.Players = append(g.Players, p)
	if g.Host == "" {
		g.Host = p.ID
	}
	return p
}
func (g *Game) setPhase(p string, seconds int) {
	g.Phase = p
	g.Deadline = 0
	if seconds > 0 {
		g.Deadline = time.Now().Add(time.Duration(seconds) * time.Second).UnixMilli()
	}
	for _, pl := range g.Players {
		pl.Ready = pl.Bot
	}
}
func (g *Game) adjacent(a, b int) bool {
	if a < 0 || a >= len(g.Territories) || b < 0 || b >= len(g.Territories) {
		return false
	}
	for _, n := range g.Territories[a].Neighbors {
		if n == b {
			return true
		}
	}
	return false
}
func rate(t string) int {
	switch t {
	case "tank":
		return 3
	case "artillery":
		return 2
	}
	return 1
}
func cost(t string, n int) Resources {
	switch t {
	case "tank":
		return Resources{n, n, 4 * n, 16 * n}
	case "artillery":
		return Resources{n, n, 2 * n, 8 * n}
	}
	return Resources{n, n, 0, 4 * n}
}
func spend(p *Player, c Resources) bool {
	r := &p.Resources
	if r.Humans < c.Humans || r.Iron < c.Iron || r.Currency < c.Currency {
		return false
	}
	r.Humans -= c.Humans
	r.Iron -= c.Iron
	r.Currency -= c.Currency
	return true
}
func (g *Game) reserved(id, except string) int {
	n := 0
	for key, o := range g.Orders {
		u := g.unit(key)
		if u != nil && u.Owner == id && key != except && o.Kind == "move" {
			n += u.Count * rate(u.Type)
		}
	}
	return n
}
func (g *Game) command(id string, c Command) error {
	p := g.player(id)
	if p == nil {
		return fmt.Errorf("Player not found")
	}
	bad := func(s string) error { return fmt.Errorf("%s", s) }
	if c.Action == "addBot" {
		if g.Phase != "lobby" || id != g.Host || len(g.Players) >= 4 {
			return bad("Cannot add commander")
		}
		g.addPlayer(fmt.Sprintf("Commander %d", len(g.Players)+1), true)
		return nil
	}
	if c.Action == "start" {
		if g.Phase != "lobby" || id != g.Host || len(g.Players) < 2 {
			return bad("Host needs at least two commanders")
		}
		rolled := map[int]bool{}
		for _, p := range g.Players {
			p.Roll = 1 + rng.IntN(6)
			for rolled[p.Roll] {
				p.Roll = 1 + rng.IntN(6)
			}
			rolled[p.Roll] = true
		}
		sort.SliceStable(g.Players, func(i, j int) bool { return g.Players[i].Roll > g.Players[j].Roll })
		g.setPhase("draft", 25)
		g.autoDraft()
		return nil
	}
	if c.Action == "claim" {
		if g.Phase != "draft" || g.Players[g.Draft%len(g.Players)].ID != id {
			return bad("Wait for your draft turn")
		}
		if c.Target < 0 || c.Target >= len(g.Territories) || g.Territories[c.Target].Owner != "" {
			return bad("Choose an unclaimed territory")
		}
		g.claim(c.Target, id)
		g.autoDraft()
		return nil
	}
	if c.Action == "ready" {
		if g.Phase != "plan" && g.Phase != "gather" && g.Phase != "simulate" {
			return bad("This phase cannot be locked")
		}
		if g.Phase == "gather" {
			g.autoFeed(p)
		}
		p.Ready = true
		if g.allReady() {
			g.advance()
		}
		return nil
	}
	if p.Ready {
		return bad("Orders are locked until the next phase")
	}
	if c.Action == "order" || c.Action == "cancel" {
		if g.Phase != "plan" {
			return bad("Orders belong in Plan")
		}
		u := g.unit(c.Unit)
		if u == nil || u.Owner != id {
			return bad("Select your own army")
		}
		if c.Action == "cancel" {
			delete(g.Orders, u.ID)
			return nil
		}
		if c.Kind != "move" && c.Kind != "bombard" && c.Kind != "defend" {
			return bad("Unknown order")
		}
		if c.Kind != "defend" && !g.adjacent(u.Territory, c.Target) {
			return bad("Target must share a border")
		}
		if c.Kind == "bombard" && u.Type != "artillery" {
			return bad("Only artillery can bombard")
		}
		if c.Kind == "move" && g.reserved(id, u.ID)+u.Count*rate(u.Type) > p.Resources.Currency {
			return bad("Insufficient currency for movement")
		}
		g.Orders[u.ID] = Order{u.ID, c.Kind, c.Target}
		return nil
	}
	if g.Phase != "gather" {
		return bad("This action belongs in Gather")
	}
	switch c.Action {
	case "recruit":
		if c.Type != "infantry" && c.Type != "tank" && c.Type != "artillery" {
			return bad("Unknown unit type")
		}
		if c.Count < 1 || c.Count > 100 || c.Target < 0 || c.Target >= len(g.Territories) || g.Territories[c.Target].Owner != id {
			return bad("Choose your territory and a valid troop count")
		}
		if !spend(p, cost(c.Type, c.Count)) {
			return bad("Not enough recruitment resources")
		}
		g.Units = append(g.Units, &Unit{uid(), id, c.Type, c.Target, c.Count, 100, 1, false})
	case "feed":
		u := g.unit(c.Unit)
		if u == nil || u.Owner != id || u.Fed {
			return bad("Choose an army that has not been fed")
		}
		if c.Ration != 0.5 && c.Ration != 1 && c.Ration != 1.5 {
			return bad("Choose minimum, normal, or extra rations")
		}
		n := int(math.Ceil(float64(u.Count) * c.Ration))
		if p.Resources.Food < n {
			return bad("Not enough food")
		}
		p.Resources.Food -= n
		u.Fed = true
		u.Ration = c.Ration
		if c.Ration == 1.5 {
			u.Health = math.Min(100, u.Health+20)
		}
	case "merge":
		a, b := g.unit(c.Unit), g.unit(c.Other)
		if a == nil || b == nil || a == b || a.Owner != id || b.Owner != id || a.Type != b.Type || a.Territory != b.Territory || a.Fed || b.Fed {
			return bad("Merge two unfed armies of the same type in one territory")
		}
		a.Health = (a.Health*float64(a.Count) + b.Health*float64(b.Count)) / float64(a.Count+b.Count)
		a.Count += b.Count
		g.remove(b.ID)
	case "reinforce":
		u := g.unit(c.Unit)
		if u == nil || u.Owner != id || u.Fed || c.Count < 1 || c.Count > 100 {
			return bad("Reinforce an unfed army")
		}
		if !spend(p, cost(u.Type, c.Count)) {
			return bad("Not enough resources")
		}
		u.Count += c.Count
	default:
		return bad("Unknown command")
	}
	return nil
}
func (g *Game) remove(id string) {
	for i, u := range g.Units {
		if u.ID == id {
			g.Units = append(g.Units[:i], g.Units[i+1:]...)
			return
		}
	}
}
func (g *Game) claim(t int, id string) {
	g.Territories[t].Owner = id
	g.Draft++
	g.Deadline = time.Now().Add(25 * time.Second).UnixMilli()
	if g.Draft == len(g.Territories) {
		for _, t := range g.Territories {
			g.Units = append(g.Units, &Unit{uid(), t.Owner, "infantry", t.ID, 4, 100, 1, false})
		}
		g.Round = 1
		g.setPhase("plan", 90)
	}
}
func (g *Game) autoDraft() {
	for g.Phase == "draft" && g.Players[g.Draft%len(g.Players)].Bot {
		g.draftOne()
	}
}
func (g *Game) draftOne() {
	free := []int{}
	for _, t := range g.Territories {
		if t.Owner == "" {
			free = append(free, t.ID)
		}
	}
	if len(free) > 0 {
		g.claim(free[rng.IntN(len(free))], g.Players[g.Draft%len(g.Players)].ID)
	}
}
func (g *Game) allReady() bool {
	for _, p := range g.Players {
		if !g.eliminated(p.ID) && !p.Ready {
			return false
		}
	}
	return true
}
func (g *Game) eliminated(id string) bool {
	if g.Phase == "lobby" || g.Phase == "draft" {
		return false
	}
	for _, t := range g.Territories {
		if t.Owner == id {
			return false
		}
	}
	for _, u := range g.Units {
		if u.Owner == id && u.Count > 0 {
			return false
		}
	}
	return true
}
func (g *Game) autoFeed(p *Player) {
	for _, u := range g.Units {
		if u.Owner != p.ID || u.Fed {
			continue
		}
		n := u.Count
		if p.Resources.Food >= n {
			p.Resources.Food -= n
			u.Ration = 1
		} else if p.Resources.Food >= int(math.Ceil(float64(n)/2)) {
			p.Resources.Food -= int(math.Ceil(float64(n) / 2))
			u.Ration = .5
		} else {
			u.Ration = 0
			u.Health = math.Max(1, u.Health-25)
		}
		u.Fed = true
	}
}
func (g *Game) advance() {
	switch g.Phase {
	case "plan":
		g.botOrders()
		g.resolve()
		g.setPhase("simulate", 12)
	case "simulate":
		g.setPhase("gather", 0)
		for _, t := range g.Territories {
			p := g.player(t.Owner)
			if p == nil {
				continue
			}
			p.Resources.Humans += t.Humans
			p.Resources.Currency += t.Currency
			if t.Resource == "food" {
				p.Resources.Food += t.Yield
			} else {
				p.Resources.Iron += t.Yield
			}
		}
		for _, u := range g.Units {
			u.Fed = false
		}
		for _, p := range g.Players {
			if p.Bot {
				g.autoFeed(p)
			}
		}
	case "gather":
		for _, p := range g.Players {
			g.autoFeed(p)
		}
		if g.Round >= 12 || g.singleOwner() {
			g.finish()
		} else {
			g.Round++
			g.Orders = map[string]Order{}
			g.Events = []Event{}
			g.setPhase("plan", 90)
		}
	}
}
func (g *Game) singleOwner() bool {
	owner := g.Territories[0].Owner
	for _, t := range g.Territories {
		if t.Owner != owner {
			return false
		}
	}
	return true
}
func (g *Game) finish() {
	g.Phase = "finished"
	g.Deadline = 0
	best := -1
	winners := []string{}
	for _, p := range g.Players {
		score := 0
		for _, t := range g.Territories {
			if t.Owner == p.ID {
				score++
			}
		}
		if score > best {
			best = score
			winners = []string{p.Name}
		} else if score == best {
			winners = append(winners, p.Name)
		}
	}
	g.Winner = fmt.Sprintf("%v · %d territories", winners, best)
}
func (g *Game) tick() {
	if (g.Phase == "plan" || g.Phase == "gather" || g.Phase == "simulate") && g.allReady() {
		g.advance()
		return
	}
	if g.Deadline > 0 && time.Now().UnixMilli() >= g.Deadline {
		if g.Phase == "draft" {
			g.draftOne()
			g.autoDraft()
		} else if g.Phase != "finished" {
			g.advance()
		}
	}
}
func (g *Game) botOrders() {
	for _, u := range g.Units {
		p := g.player(u.Owner)
		if !p.Bot {
			continue
		}
		for _, n := range g.Territories[u.Territory].Neighbors {
			if g.Territories[n].Owner != u.Owner {
				kind := "move"
				if u.Type == "artillery" {
					kind = "bombard"
				}
				_ = g.commandBot(u, kind, n)
				break
			}
		}
	}
}
func (g *Game) commandBot(u *Unit, kind string, n int) error {
	p := g.player(u.Owner)
	if kind == "move" && g.reserved(p.ID, "")+u.Count*rate(u.Type) > p.Resources.Currency {
		return nil
	}
	g.Orders[u.ID] = Order{u.ID, kind, n}
	return nil
}
func attack(u *Unit, other string, direct bool) float64 {
	base := 2.0
	mod := 1.0
	switch u.Type {
	case "tank":
		base = 7
		if other == "infantry" {
			mod = 1.4
		}
	case "artillery":
		base = 4
		if direct {
			base = 1
		} else if other == "infantry" {
			mod = 1.5
		} else if other == "tank" {
			mod = .7
		}
	}
	if other == "artillery" && u.Type != "artillery" {
		mod = 1.5
	}
	ration := 1.0
	if u.Ration < 1 {
		ration = .65
	}
	if u.Ration == 0 {
		ration = .3
	}
	return float64(u.Count) * base * (u.Health / 100) * mod * ration
}
func hurt(u *Unit, damage float64) {
	before := u.Count
	loss := int(math.Floor(damage / 10))
	u.Count = max(0, u.Count-loss)
	u.Health = math.Max(0, u.Health-damage/float64(max(1, before))*6)
	if u.Health <= 0 {
		u.Count = 0
	}
}
func (g *Game) resolve() {
	g.Events = []Event{}
	orig := map[string]int{}
	damage := map[string]float64{}
	for _, u := range g.Units {
		orig[u.ID] = u.Territory
		if o, ok := g.Orders[u.ID]; ok && o.Kind == "move" {
			g.player(u.Owner).Resources.Currency -= u.Count * rate(u.Type)
		}
	}
	for _, u := range g.Units {
		o, ok := g.Orders[u.ID]
		if !ok || o.Kind != "bombard" {
			continue
		}
		g.Events = append(g.Events, Event{fmt.Sprintf("Artillery bombards %s", g.Territories[o.Target].Name), u.Territory, o.Target, "bombard"})
		targets := 0
		for _, v := range g.Units {
			if v.Territory == o.Target && v.Owner != u.Owner {
				targets++
			}
		}
		for _, v := range g.Units {
			if v.Territory == o.Target && v.Owner != u.Owner {
				damage[v.ID] += attack(u, v.Type, false) / float64(max(1, targets))
			}
		}
	}
	for _, u := range g.Units {
		if d := damage[u.ID]; d > 0 {
			hurt(u, d)
		}
	}
	// Resolve opposing border crossings at their origins; surviving forces hold.
	blocked := map[string]bool{}
	damage = map[string]float64{}
	for i, a := range g.Units {
		oa, ok := g.Orders[a.ID]
		if !ok || oa.Kind != "move" || a.Count == 0 {
			continue
		}
		for _, b := range g.Units[i+1:] {
			ob, ok := g.Orders[b.ID]
			if ok && ob.Kind == "move" && a.Owner != b.Owner && oa.Target == b.Territory && ob.Target == a.Territory && b.Count > 0 {
				blocked[a.ID] = true
				blocked[b.ID] = true
				damage[a.ID] += attack(b, a.Type, true)
				damage[b.ID] += attack(a, b.Type, true)
				g.Events = append(g.Events, Event{"Opposing armies clash at the border", a.Territory, b.Territory, "battle"})
			}
		}
	}
	for _, u := range g.Units {
		if d := damage[u.ID]; d > 0 {
			hurt(u, d)
		}
	}
	for _, u := range g.Units {
		o, ok := g.Orders[u.ID]
		if ok && o.Kind == "move" {
			if !blocked[u.ID] && u.Count > 0 {
				g.Events = append(g.Events, Event{fmt.Sprintf("%s advances to %s", u.Type, g.Territories[o.Target].Name), u.Territory, o.Target, "move"})
				u.Territory = o.Target
			}
		}
	}
	stationed := map[int][]*Unit{}
	retreats := map[string]bool{}
	for _, u := range g.Units {
		if u.Count > 0 {
			stationed[u.Territory] = append(stationed[u.Territory], u)
		}
	}
	for _, t := range g.Territories {
		groups := map[string][]*Unit{}
		for _, u := range stationed[t.ID] {
			if u.Count > 0 && u.Territory == t.ID {
				groups[u.Owner] = append(groups[u.Owner], u)
			}
		}
		if len(groups) > 1 {
			losses := map[string]float64{}
			for _, a := range stationed[t.ID] {
				if a.Count == 0 || a.Territory != t.ID {
					continue
				}
				for owner, targets := range groups {
					if owner == a.Owner {
						continue
					}
					for _, b := range targets {
						losses[b.ID] += attack(a, b.Type, true) / float64(len(targets)*(len(groups)-1))
					}
				}
			}
			for _, u := range g.Units {
				if d := losses[u.ID]; d > 0 {
					hurt(u, d)
				}
			}
			g.Events = append(g.Events, Event{"Battle at " + t.Name, t.ID, t.ID, "battle"})
		}
		best := ""
		power := -1.0
		tie := false
		keys := []string{}
		for k := range groups {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, owner := range keys {
			strength := 0.0
			for _, u := range groups[owner] {
				if u.Count > 0 {
					strength += attack(u, "infantry", true)
				}
			}
			if strength > power {
				power = strength
				best = owner
				tie = false
			} else if strength == power {
				tie = true
			}
		}
		if tie || power <= 0 {
			best = ""
		}
		for owner, units := range groups {
			if owner == best {
				continue
			}
			for _, u := range units {
				if u.Count <= 0 {
					continue
				}
				retreats[u.ID] = true
			}
		}
		if best != "" {
			g.Territories[t.ID].Owner = best
		}
	}
	// Retreat only after every destination's battle and ownership are resolved.
	for _, u := range g.Units {
		if !retreats[u.ID] || u.Count <= 0 {
			continue
		}
		retreat := -1
		if orig[u.ID] != u.Territory && g.Territories[orig[u.ID]].Owner == u.Owner {
			retreat = orig[u.ID]
		} else {
			for _, n := range g.Territories[u.Territory].Neighbors {
				if g.Territories[n].Owner == u.Owner {
					retreat = n
					break
				}
			}
		}
		if retreat < 0 {
			u.Count = 0
		} else {
			u.Territory = retreat
		}
	}
	living := []*Unit{}
	for _, u := range g.Units {
		if u.Count > 0 {
			living = append(living, u)
		}
	}
	g.Units = living
}
