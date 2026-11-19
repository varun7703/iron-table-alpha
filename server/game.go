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
	DisconnectedUntil int64     `json:"disconnectedUntil"`
	Capital           int       `json:"capital"`
	Eliminated        bool      `json:"eliminated"`
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Color             string    `json:"color"`
	Roll              int       `json:"roll"`
	Ready             bool      `json:"ready"`
	Bot               bool      `json:"bot"`
	Resources         Resources `json:"resources"`
	Token             string    `json:"-"`
}
type Territory struct {
	Neutral   bool    `json:"neutral"`
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
	RollIndex   int              `json:"rollIndex"`
	Timeline    []BattleFrame    `json:"timeline"`
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
	color := colors[0]
	for _, candidate := range colors {
		used := false
		for _, existing := range g.Players {
			if existing.Color == candidate {
				used = true
			}
		}
		if !used {
			color = candidate
			break
		}
	}
	p := &Player{Capital: -1, ID: uid(), Token: uid(), Name: name, Color: color, Bot: bot, Resources: Resources{40, 120, 40, 160}}
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
		return Resources{n, 0, 8 * n, 4 * n}
	case "artillery":
		return Resources{n, 0, 4 * n, 2 * n}
	}
	return Resources{n, 0, 0, 4 * n}
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
		g.setPhase("roll", 2)
		return nil
	}
	if c.Action == "capital" {
		return g.chooseCapital(id, c.Target)
	}
	if c.Action == "claim" {
		if g.Phase != "draft" || g.draftPlayer().ID != id {
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
		p.Ready = true
		if g.allReady() {
			g.advance()
		}
		return nil
	}
	if c.Action == "unready" {
		if g.Phase != "plan" && g.Phase != "gather" && g.Phase != "simulate" {
			return bad("This phase cannot be unlocked")
		}
		p.Ready = false
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
		stacks := 0
		for _, u := range g.Units {
			if u.Territory == c.Target && u.Count > 0 {
				stacks++
			}
		}
		if stacks >= 6 {
			return bad("Sector limit reached (6 armies); merge or reinforce existing armies")
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
		u.Health = (u.Health*float64(u.Count) + 100*float64(c.Count)) / float64(u.Count+c.Count)
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
	if g.Draft == 4*len(g.Players) {
		g.beginCapitals()
	}
}
func (g *Game) autoDraft() {
	for g.Phase == "draft" && g.draftPlayer() != nil && g.draftPlayer().Bot {
		g.draftOne()
	}
}
func (g *Game) draftOne() {
	if g.draftPlayer() == nil {
		return
	}
	free := []int{}
	for _, t := range g.Territories {
		if t.Owner == "" {
			free = append(free, t.ID)
		}
	}
	if len(free) > 0 {
		g.claim(free[rng.IntN(len(free))], g.draftPlayer().ID)
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
	if g.Phase == "lobby" || g.Phase == "draft" || g.Phase == "roll" || g.Phase == "capital" {
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
func unitPriority(t string) int {
	switch t {
	case "tank":
		return 3
	case "artillery":
		return 2
	}
	return 1
}
func (g *Game) autoFeed(p *Player) {
	unfed := []*Unit{}
	for _, u := range g.Units {
		if u.Owner == p.ID && !u.Fed && u.Count > 0 {
			unfed = append(unfed, u)
		}
	}
	sort.SliceStable(unfed, func(i, j int) bool {
		pi, pj := unitPriority(unfed[i].Type), unitPriority(unfed[j].Type)
		if pi != pj {
			return pi > pj
		}
		return unfed[i].Count > unfed[j].Count
	})
	for _, u := range unfed {
		minFood := int(math.Ceil(float64(u.Count) / 2))
		if p.Resources.Food >= minFood {
			p.Resources.Food -= minFood
			u.Ration = .5
		} else {
			u.Ration = 0
			u.Health = math.Max(1, u.Health-25)
		}
		u.Fed = true
	}
	for _, u := range unfed {
		if u.Ration == .5 {
			extra := u.Count - int(math.Ceil(float64(u.Count)/2))
			if p.Resources.Food >= extra {
				p.Resources.Food -= extra
				u.Ration = 1
			}
		}
	}
}
func (g *Game) advance() {
	switch g.Phase {
	case "plan":
		g.botOrders()
		g.resolve()
		g.setPhase("simulate", max(12, len(g.Timeline)*3))
	case "simulate":
		if g.Round >= 20 || g.singleOwner() {
			g.finish()
			return
		}
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
		if g.Round >= 20 || g.singleOwner() {
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
	if owner == "" {
		return false
	}
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
		if g.Phase == "roll" {
			g.rollNext()
		} else if g.Phase == "draft" {
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

// applyDamage applies accumulated damage to units while pooling fractional
// troop losses across same-sector, same-owner, same-type stacks so splitting
// armies into smaller stacks neither reduces nor amplifies casualties.
func applyDamage(units []*Unit, damage map[string]float64) {
	type groupKey struct {
		territory int
		owner     string
		kind      string
	}
	type entry struct {
		u        *Unit
		before   int
		baseLoss int
		dmg      float64
		rem      float64
	}
	groups := map[groupKey][]entry{}
	order := []groupKey{}
	for _, u := range units {
		d := damage[u.ID]
		if d <= 0 || u.Count <= 0 {
			continue
		}
		k := groupKey{u.Territory, u.Owner, u.Type}
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		baseLoss := int(math.Floor(d / 10))
		groups[k] = append(groups[k], entry{
			u:        u,
			before:   u.Count,
			baseLoss: baseLoss,
			dmg:      d,
			rem:      d - float64(baseLoss*10),
		})
	}
	for _, k := range order {
		items := groups[k]
		totalDmg := 0.0
		sumKilled := 0
		for _, it := range items {
			totalDmg += it.dmg
			it.u.Count = max(0, it.before-it.baseLoss)
			it.u.Health = math.Max(0, it.u.Health-it.dmg/float64(max(1, it.before))*6)
			if it.u.Health <= 0 {
				it.u.Count = 0
			}
			sumKilled += it.before - it.u.Count
		}
		extra := int(math.Floor((totalDmg+1e-9)/10)) - sumKilled
		if extra > 0 {
			idx := make([]int, len(items))
			for i := range idx {
				idx[i] = i
			}
			sort.SliceStable(idx, func(i, j int) bool {
				ri, rj := items[idx[i]].rem, items[idx[j]].rem
				if math.Abs(ri-rj) > 1e-9 {
					return ri > rj
				}
				return items[idx[i]].u.Count > items[idx[j]].u.Count
			})
			for extra > 0 {
				progressed := false
				for _, pos := range idx {
					if extra <= 0 {
						break
					}
					if items[pos].u.Count > 0 {
						items[pos].u.Count--
						extra--
						progressed = true
					}
				}
				if !progressed {
					break
				}
			}
		}
	}
}

// distributeDamage spreads attacker a's damage across hostile living targets in
// territory (or across all hostile living targets when territory < 0), weighted
// by each target stack's troop count. Returns true if any hostile troops were hit.
func distributeDamage(a *Unit, targets []*Unit, territory int, direct bool, out map[string]float64) bool {
	oppTroops := 0
	for _, b := range targets {
		if b.Owner != a.Owner && b.Count > 0 && (territory < 0 || b.Territory == territory) {
			oppTroops += b.Count
		}
	}
	if oppTroops <= 0 {
		return false
	}
	for _, b := range targets {
		if b.Owner != a.Owner && b.Count > 0 && (territory < 0 || b.Territory == territory) {
			out[b.ID] += attack(a, b.Type, direct) * float64(b.Count) / float64(oppTroops)
		}
	}
	return true
}

func (g *Game) resolve() {
	g.Events = []Event{}
	g.Timeline = []BattleFrame{}
	g.recordFrame("Orders revealed · armies before combat")
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
		if !ok || o.Kind != "bombard" || u.Count <= 0 {
			continue
		}
		g.Events = append(g.Events, Event{fmt.Sprintf("Artillery bombards %s", g.Territories[o.Target].Name), u.Territory, o.Target, "bombard"})
		distributeDamage(u, g.Units, o.Target, false, damage)
	}
	applyDamage(g.Units, damage)
	g.recordFrame("Bombardment resolved · simultaneous artillery damage")
	// Resolve opposing border crossings at their origins; surviving forces hold.
	blocked := map[string]bool{}
	damage = map[string]float64{}
	type edge struct{ from, to int }
	crossing := map[edge][]*Unit{}
	edgePairs := [][2]int{}
	seenPair := map[[2]int]bool{}
	for _, u := range g.Units {
		o, ok := g.Orders[u.ID]
		if !ok || o.Kind != "move" || u.Count <= 0 {
			continue
		}
		crossing[edge{u.Territory, o.Target}] = append(crossing[edge{u.Territory, o.Target}], u)
		pair := [2]int{min(u.Territory, o.Target), max(u.Territory, o.Target)}
		if !seenPair[pair] {
			seenPair[pair] = true
			edgePairs = append(edgePairs, pair)
		}
	}
	for _, pair := range edgePairs {
		x, y := pair[0], pair[1]
		sideX, sideY := crossing[edge{x, y}], crossing[edge{y, x}]
		if len(sideX) == 0 || len(sideY) == 0 {
			continue
		}
		clashed := false
		for _, a := range sideX {
			if distributeDamage(a, sideY, -1, true, damage) {
				blocked[a.ID] = true
				clashed = true
			}
		}
		for _, b := range sideY {
			if distributeDamage(b, sideX, -1, true, damage) {
				blocked[b.ID] = true
			}
		}
		if clashed {
			g.Events = append(g.Events, Event{"Opposing armies clash at the border", x, y, "battle"})
		}
	}
	applyDamage(g.Units, damage)
	g.recordFrame("Border clashes resolved · simultaneous return fire")
	for _, u := range g.Units {
		o, ok := g.Orders[u.ID]
		if ok && o.Kind == "move" {
			if !blocked[u.ID] && u.Count > 0 {
				g.Events = append(g.Events, Event{fmt.Sprintf("%s advances to %s", u.Type, g.Territories[o.Target].Name), u.Territory, o.Target, "move"})
				u.Territory = o.Target
			}
		}
	}
	g.recordFrame("Movement resolved · armies reach their destinations")
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
				if a.Count <= 0 || a.Territory != t.ID {
					continue
				}
				distributeDamage(a, stationed[t.ID], t.ID, true, losses)
			}
			applyDamage(stationed[t.ID], losses)
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
		strengths := map[string]float64{}
		for _, owner := range keys {
			strength := 0.0
			for _, u := range groups[owner] {
				if u.Count > 0 {
					strength += attack(u, "", true)
				}
			}
			strengths[owner] = strength
			if strength > power+1e-9 {
				power = strength
				best = owner
				tie = false
			} else if math.Abs(strength-power) <= 1e-9 {
				tie = true
			}
		}
		if power <= 0 {
			best = ""
		} else if tie {
			if incumbent, ok := strengths[t.Owner]; ok && math.Abs(incumbent-power) <= 1e-9 {
				best = t.Owner
			} else {
				best = ""
			}
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
			if len(groups) == 1 && best != t.Owner {
				if p := g.player(best); p != nil {
					g.Events = append(g.Events, Event{fmt.Sprintf("%s captures %s", p.Name, t.Name), t.ID, t.ID, "capture"})
				}
			}
			g.Territories[t.ID].Owner = best
		}
		if len(groups) > 1 || (best != "" && best != t.Owner) {
			g.recordFrame("Territory resolved · " + t.Name)
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
	g.recordFrame("Retreats and captures complete · final battlefield")
}
