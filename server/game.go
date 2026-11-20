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
	VP                int       `json:"vp"`
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
	Visible   bool    `json:"visible"`
	IsCapital bool    `json:"isCapital"`
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
	RollIndex   int                     `json:"rollIndex"`
	Timeline    []BattleFrame           `json:"timeline"`
	Code        string                  `json:"code"`
	Host        string                  `json:"host"`
	Phase       string                  `json:"phase"`
	Round       int                     `json:"round"`
	Deadline    int64                   `json:"deadline"`
	Players     []*Player               `json:"players"`
	Territories []Territory             `json:"territories"`
	Units       []*Unit                 `json:"units"`
	Orders      map[string]Order        `json:"orders"`
	Events      []Event                 `json:"events"`
	LastEvents  []Event                 `json:"-"`
	LastSimVis  map[string]map[int]bool `json:"-"`
	Draft       int                     `json:"draft"`
	Winner      string                  `json:"winner"`
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

const (
	maxStacksPerSector      = 4
	capitalCurrencyBonus    = 4
	capitalHumansBonus      = 2
	capitalDefenseBonus     = 1.15
	capitalVP               = 2
	simulateDurationSeconds = 24
)

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
	p := &Player{Capital: -1, ID: uid(), Token: uid(), Name: name, Color: color, Bot: bot, Resources: Resources{Humans: 16, Food: 40, Iron: 16, Currency: 60}}
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
func unitHP(t string) float64 {
	switch t {
	case "tank":
		return 20
	case "artillery":
		return 14
	}
	return 10
}
func cost(t string, n int) Resources {
	switch t {
	case "tank":
		return Resources{Humans: 3 * n, Iron: 8 * n, Currency: 8 * n}
	case "artillery":
		return Resources{Humans: 2 * n, Iron: 4 * n, Currency: 4 * n}
	}
	return Resources{Humans: n, Currency: 4 * n}
}
func (g *Game) moveCost(u *Unit, target int) int {
	if target >= 0 && target < len(g.Territories) && g.Territories[target].Owner == u.Owner {
		return 0
	}
	return u.Count * rate(u.Type)
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
			n += g.moveCost(u, o.Target)
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
		dp := g.draftPlayer()
		if g.Phase != "draft" || dp == nil || dp.ID != id {
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
	if c.Action == "split" {
		if g.Phase != "gather" && g.Phase != "plan" {
			return bad("Split belongs in Plan or Gather")
		}
		u := g.unit(c.Unit)
		if u == nil || u.Owner != id {
			return bad("Select your own army")
		}
		if g.Phase == "gather" && u.Fed {
			return bad("Split an unfed army")
		}
		if c.Count < 1 || c.Count >= u.Count {
			return bad("Choose a split count smaller than the army")
		}
		if g.sectorStackCount(u.Territory) >= maxStacksPerSector {
			return bad(fmt.Sprintf("Sector limit reached (%d armies); merge or reinforce existing armies", maxStacksPerSector))
		}
		if g.Phase == "plan" {
			delete(g.Orders, u.ID)
		}
		u.Count -= c.Count
		g.Units = append(g.Units, &Unit{uid(), id, u.Type, u.Territory, c.Count, u.Health, u.Ration, u.Fed})
		return nil
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
		if c.Kind == "move" && g.reserved(id, u.ID)+g.moveCost(u, c.Target) > p.Resources.Currency {
			return bad("Insufficient currency for movement")
		}
		target := c.Target
		if c.Kind == "defend" {
			target = u.Territory
		}
		g.Orders[u.ID] = Order{u.ID, c.Kind, target}
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
		if g.sectorStackCount(c.Target) >= maxStacksPerSector {
			return bad(fmt.Sprintf("Sector limit reached (%d armies); merge or reinforce existing armies", maxStacksPerSector))
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
		mergeUnits(a, b)
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
func (g *Game) sectorStackCount(tid int) int {
	n := 0
	for _, u := range g.Units {
		if u.Territory == tid && u.Count > 0 {
			n++
		}
	}
	return n
}
func mergeUnits(a, b *Unit) {
	a.Health = (a.Health*float64(a.Count) + b.Health*float64(b.Count)) / float64(a.Count+b.Count)
	a.Count += b.Count
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
	if g.Draft >= 4*len(g.Players) {
		g.beginCapitals()
	}
}
func (g *Game) autoDraft() {
	for g.Phase == "draft" && g.draftPlayer() != nil && g.draftPlayer().Bot {
		g.draftOne()
	}
}
func (g *Game) draftOne() {
	p := g.draftPlayer()
	if p == nil {
		return
	}
	if pick := g.pickDraftTerritory(p); pick >= 0 {
		g.claim(pick, p.ID)
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
	if id == "" {
		return true
	}
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
		g.setPhase("simulate", simulateDurationSeconds)
	case "simulate":
		g.Timeline = nil
		if g.Round >= 20 || g.singleOwner() {
			g.finish()
			return
		}
		g.setPhase("gather", 120)
		g.Orders = map[string]Order{}
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
		for _, p := range g.Players {
			if p.Capital >= 0 && p.Capital < len(g.Territories) && g.Territories[p.Capital].Owner == p.ID {
				p.Resources.Currency += capitalCurrencyBonus
				p.Resources.Humans += capitalHumansBonus
			}
		}
		for _, u := range g.Units {
			u.Fed = false
		}
		// Snapshot pre-Gather unit visibility so bots in the same Gather tick cannot react to each other's new recruits.
		snapshot := make([]Unit, 0, len(g.Units))
		for _, u := range g.Units {
			if u.Count > 0 {
				snapshot = append(snapshot, *u)
			}
		}
		for _, p := range g.Players {
			if p.Bot {
				g.botGatherWithSnapshot(p, snapshot)
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
func (g *Game) isCapitalSector(tid int) bool {
	if tid < 0 || tid >= len(g.Territories) {
		return false
	}
	if g.Territories[tid].IsCapital {
		return true
	}
	for _, p := range g.Players {
		if p.Capital == tid {
			return true
		}
	}
	return false
}
func (g *Game) playerScore(id string) (vp, territories, capitals int) {
	for _, t := range g.Territories {
		if t.Owner == id {
			territories++
			if g.isCapitalSector(t.ID) {
				capitals++
			}
		}
	}
	return territories + capitalVP*capitals, territories, capitals
}
func (g *Game) finish() {
	g.Phase = "finished"
	g.Deadline = 0
	g.Orders = map[string]Order{}
	g.Timeline = nil
	bestVP, bestTerr, bestCaps := -1, -1, -1
	winners := []string{}
	for _, p := range g.Players {
		vp, terr, caps := g.playerScore(p.ID)
		if vp > bestVP || (vp == bestVP && terr > bestTerr) || (vp == bestVP && terr == bestTerr && caps > bestCaps) {
			bestVP, bestTerr, bestCaps = vp, terr, caps
			winners = []string{p.Name}
		} else if vp == bestVP && terr == bestTerr && caps == bestCaps {
			winners = append(winners, p.Name)
		}
	}
	g.Winner = fmt.Sprintf("%v · %d VP (%d territories, %d capitals)", winners, bestVP, bestTerr, bestCaps)
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
		} else if g.Phase == "capital" {
			g.autoPickCapitals()
		} else if g.Phase != "finished" {
			g.advance()
		}
	}
}
func isDefending(orders map[string]Order, u *Unit) bool {
	if orders == nil || u == nil {
		return false
	}
	o, ok := orders[u.ID]
	return ok && o.Kind == "defend"
}
func defendBonus(orders map[string]Order, u *Unit, direct bool) float64 {
	if direct && isDefending(orders, u) {
		return 1.2
	}
	return 1.0
}
func (g *Game) capitalBonus(u *Unit, territory int, direct bool) float64 {
	if !direct || territory < 0 || territory >= len(g.Territories) || u == nil {
		return 1.0
	}
	p := g.player(u.Owner)
	if p != nil && p.Capital == territory && g.Territories[territory].Owner == u.Owner {
		return capitalDefenseBonus
	}
	return 1.0
}
func attack(u *Unit, other string, direct bool) float64 {
	base := 2.0
	mod := 1.0
	switch u.Type {
	case "tank":
		base = 6
		if other == "infantry" {
			mod = 1.4
		}
	case "artillery":
		base = 4
		if direct {
			base = 1
		} else if other == "tank" {
			mod = 1.5
		} else if other == "infantry" {
			mod = 1.2
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
	applyDamageBucketed(units, damage, nil)
}

func applyDamageBucketed(units []*Unit, damage map[string]float64, bucket func(*Unit) int) {
	type groupKey struct {
		territory int
		owner     string
		kind      string
		bucket    int
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
		b := 0
		if bucket != nil {
			b = bucket(u)
		}
		k := groupKey{u.Territory, u.Owner, u.Type, b}
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		hp := unitHP(u.Type)
		baseLoss := int(math.Floor(d / hp))
		groups[k] = append(groups[k], entry{
			u:        u,
			before:   u.Count,
			baseLoss: baseLoss,
			dmg:      d,
			rem:      d - float64(baseLoss)*hp,
		})
	}
	for _, k := range order {
		items := groups[k]
		hp := unitHP(k.kind)
		totalDmg := 0.0
		sumKilled := 0
		for _, it := range items {
			// Cap pooled damage per stack at its lethal capacity so overkill on a wiped
			// stack cannot bleed into other surviving stacks in the same group.
			totalDmg += math.Min(it.dmg, float64(it.before)*hp)
			it.u.Count = max(0, it.before-it.baseLoss)
			it.u.Health = math.Max(0, it.u.Health-(it.dmg/(float64(max(1, it.before))*hp))*60)
			if it.u.Health <= 0 {
				it.u.Count = 0
			}
			sumKilled += it.before - it.u.Count
		}
		extra := int(math.Floor((totalDmg+1e-9)/hp)) - sumKilled
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
// by each target stack's effective HP (count * unitHP * health/100). Entrenched
// units gain +20% direct combat power and take 25% less bombardment damage; units
// defending their own capital gain +15% direct combat power in destination battles.
// Returns true if any hostile troops were hit.
func (g *Game) distributeDamage(a *Unit, targets []*Unit, territory int, direct bool, orders map[string]Order, out map[string]float64) bool {
	oppWeight := 0.0
	for _, b := range targets {
		if b.Owner != a.Owner && b.Count > 0 && b.Health > 0 && (territory < 0 || b.Territory == territory) {
			oppWeight += float64(b.Count) * unitHP(b.Type) * (b.Health / 100)
		}
	}
	if oppWeight <= 0 {
		return false
	}
	atkMult := defendBonus(orders, a, direct) * g.capitalBonus(a, territory, direct)
	for _, b := range targets {
		if b.Owner != a.Owner && b.Count > 0 && b.Health > 0 && (territory < 0 || b.Territory == territory) {
			share := float64(b.Count) * unitHP(b.Type) * (b.Health / 100) / oppWeight
			defMult := 1.0
			if !direct && isDefending(orders, b) {
				defMult = 0.75
			}
			out[b.ID] += attack(a, b.Type, direct) * atkMult * share * defMult
		}
	}
	return true
}

func (g *Game) resolve() {
	g.Events = []Event{}
	g.Timeline = []BattleFrame{}
	g.recordFrame("Orders revealed · armies before combat")
	orig := map[string]int{}
	for _, u := range g.Units {
		orig[u.ID] = u.Territory
		if o, ok := g.Orders[u.ID]; ok && o.Kind == "move" {
			if p := g.player(u.Owner); p != nil {
				p.Resources.Currency -= g.moveCost(u, o.Target)
			}
		}
	}
	// Stage 1: Resolve opposing border crossings per edge so unrelated borders never pool damage.
	blocked := map[string]bool{}
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
		edgeDamage := map[string]float64{}
		clashed := false
		for _, a := range sideX {
			if g.distributeDamage(a, sideY, -1, true, g.Orders, edgeDamage) {
				blocked[a.ID] = true
				clashed = true
			}
		}
		for _, b := range sideY {
			if g.distributeDamage(b, sideX, -1, true, g.Orders, edgeDamage) {
				blocked[b.ID] = true
			}
		}
		if clashed {
			edgeUnits := make([]*Unit, 0, len(sideX)+len(sideY))
			edgeUnits = append(edgeUnits, sideX...)
			edgeUnits = append(edgeUnits, sideY...)
			applyDamage(edgeUnits, edgeDamage)
			g.Events = append(g.Events, Event{"Opposing armies clash at the border", x, y, "battle"})
		}
	}
	g.recordFrame("Border clashes resolved · simultaneous return fire")
	// Stage 2: Advance unblocked movers to their destination sectors.
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
	// Stage 3: Bombardment resolves after movement so backline artillery shells incoming attackers.
	// Artillery overrun by hostile troops in its own sector during Stage 2 is locked in melee and suppressed.
	overrun := func(u *Unit) bool {
		for _, v := range g.Units {
			if v.Territory == u.Territory && v.Owner != u.Owner && v.Count > 0 {
				return true
			}
		}
		return false
	}
	damage := map[string]float64{}
	for _, u := range g.Units {
		o, ok := g.Orders[u.ID]
		if !ok || o.Kind != "bombard" || u.Count <= 0 || overrun(u) {
			continue
		}
		g.Events = append(g.Events, Event{fmt.Sprintf("Artillery bombards %s", g.Territories[o.Target].Name), u.Territory, o.Target, "bombard"})
		g.distributeDamage(u, g.Units, o.Target, false, g.Orders, damage)
	}
	applyDamageBucketed(g.Units, damage, func(u *Unit) int {
		if isDefending(g.Orders, u) {
			return 1
		}
		return 0
	})
	g.recordFrame("Bombardment resolved · simultaneous artillery damage")
	// Stage 4: Destination battles and territory ownership.
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
				g.distributeDamage(a, stationed[t.ID], t.ID, true, g.Orders, losses)
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
					strength += attack(u, "", true) * defendBonus(g.Orders, u, true) * g.capitalBonus(u, t.ID, true)
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
		// When a multi-way tie leaves territory ownership unchanged, the incumbent
		// garrison retains its sector while all tied attackers retreat.
		holder := best
		if holder == "" {
			holder = t.Owner
		}
		for owner, units := range groups {
			if owner == holder {
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
	g.LastEvents = append([]Event{}, g.Events...)
	g.LastSimVis = make(map[string]map[int]bool, len(g.Players))
	for _, p := range g.Players {
		g.LastSimVis[p.ID] = g.simulateVisibleSectors(p.ID)
	}
}
