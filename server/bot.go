package main

import (
	"math"
	rng "math/rand/v2"
	"sort"
)

func (g *Game) pickDraftTerritory(p *Player) int {
	if p == nil {
		return -1
	}
	foodTiles, ironTiles := 0, 0
	for _, t := range g.Territories {
		if t.Owner == p.ID {
			if t.Resource == "food" {
				foodTiles++
			} else {
				ironTiles++
			}
		}
	}
	bestID := -1
	bestScore := -1e9
	for _, t := range g.Territories {
		if t.Owner != "" {
			continue
		}
		score := float64(t.Yield) + 0.3*float64(len(t.Neighbors))
		if t.Resource == "food" && foodTiles < ironTiles {
			score += 2.5 * float64(ironTiles-foodTiles)
			if foodTiles == 0 {
				score += 2.5
			}
		} else if t.Resource == "iron" && ironTiles < foodTiles {
			score += 2.5 * float64(foodTiles-ironTiles)
			if ironTiles == 0 {
				score += 2.5
			}
		}
		ownAdj, enemyAdj := 0, 0
		for _, n := range t.Neighbors {
			owner := g.Territories[n].Owner
			if owner == p.ID {
				ownAdj++
			} else if owner != "" {
				enemyAdj++
			}
		}
		if ownAdj > 0 {
			score += 4.5 + 2.0*float64(ownAdj-1)
		}
		if enemyAdj == 0 {
			score += 1.0
		}
		score += rng.Float64() * 0.5
		if score > bestScore {
			bestScore = score
			bestID = t.ID
		}
	}
	return bestID
}

func (g *Game) pickBotCapital(p *Player) int {
	if p == nil {
		return -1
	}
	bestID := -1
	bestScore := -1e9
	for _, t := range g.Territories {
		if t.Owner != p.ID {
			continue
		}
		ownAdj, enemyAdj := 0, 0
		for _, n := range t.Neighbors {
			owner := g.Territories[n].Owner
			if owner == p.ID {
				ownAdj++
			} else if owner != "" {
				enemyAdj++
			}
		}
		score := 4.0*float64(ownAdj) - 2.0*float64(enemyAdj) + 0.5*float64(t.Yield)
		if score > bestScore {
			bestScore = score
			bestID = t.ID
		}
	}
	return bestID
}

func enemyPowerInSlice(units []Unit, tid int, owner string) float64 {
	total := 0.0
	for i := range units {
		u := &units[i]
		if u.Territory == tid && u.Owner != owner && u.Count > 0 {
			total += attack(u, "", true)
		}
	}
	return total
}

func enemyHasTanksInSlice(units []Unit, tid int, owner string) bool {
	for i := range units {
		u := &units[i]
		if u.Territory == tid && u.Owner != owner && u.Type == "tank" && u.Count > 0 {
			return true
		}
	}
	return false
}

func (g *Game) enemyPowerAt(tid int, owner string) float64 {
	total := 0.0
	for _, u := range g.Units {
		if u.Territory == tid && u.Owner != owner && u.Count > 0 {
			total += attack(u, "", true)
		}
	}
	return total
}

func (g *Game) enemyHasTanksAt(tid int, owner string) bool {
	for _, u := range g.Units {
		if u.Territory == tid && u.Owner != owner && u.Type == "tank" && u.Count > 0 {
			return true
		}
	}
	return false
}

// sectorThreat sums hostile combat power across all adjacent enemy-owned sectors
// that could converge on tid in the upcoming turn.
func (g *Game) sectorThreat(tid int, owner string) float64 {
	total := 0.0
	for _, n := range g.Territories[tid].Neighbors {
		if g.Territories[n].Owner != "" && g.Territories[n].Owner != owner {
			total += g.enemyPowerAt(n, owner)
		}
	}
	return total
}

func (g *Game) sectorOwnPower(tid int, owner string) float64 {
	total := 0.0
	for _, u := range g.Units {
		if u.Territory == tid && u.Owner == owner && u.Count > 0 {
			total += attack(u, "", true)
		}
	}
	return total
}

func (g *Game) sectorHasArtillery(tid int, owner string) bool {
	for _, u := range g.Units {
		if u.Territory == tid && u.Owner == owner && u.Type == "artillery" && u.Count > 0 {
			return true
		}
	}
	return false
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

func (g *Game) expectedMatchupPower(stacks []*Unit, targetTile int, owner string) float64 {
	oppWeight := 0.0
	for _, v := range g.Units {
		if v.Territory == targetTile && v.Owner != owner && v.Count > 0 && v.Health > 0 {
			oppWeight += float64(v.Count) * unitHP(v.Type) * (v.Health / 100)
		}
	}
	total := 0.0
	if oppWeight <= 0 {
		for _, u := range stacks {
			total += attack(u, "", true)
		}
		return total
	}
	for _, u := range stacks {
		for _, v := range g.Units {
			if v.Territory == targetTile && v.Owner != owner && v.Count > 0 && v.Health > 0 {
				share := float64(v.Count) * unitHP(v.Type) * (v.Health / 100) / oppWeight
				total += attack(u, v.Type, true) * share
			}
		}
	}
	return total
}

func (g *Game) botGather(p *Player) {
	snapshot := make([]Unit, 0, len(g.Units))
	for _, u := range g.Units {
		if u.Count > 0 {
			snapshot = append(snapshot, *u)
		}
	}
	g.botGatherWithSnapshot(p, snapshot)
}

func (g *Game) botGatherWithSnapshot(p *Player, snapshot []Unit) {
	// 1. Merge duplicate unfed stacks of the same type in each territory.
	type stackKey struct {
		tid  int
		kind string
	}
	primary := map[stackKey]*Unit{}
	toRemove := []string{}
	for _, u := range g.Units {
		if u.Owner != p.ID || u.Fed || u.Count <= 0 {
			continue
		}
		k := stackKey{u.Territory, u.Type}
		if first, ok := primary[k]; ok {
			mergeUnits(first, u)
			toRemove = append(toRemove, u.ID)
		} else {
			primary[k] = u
		}
	}
	for _, id := range toRemove {
		g.remove(id)
	}

	// 2. Classify owned sectors and compute recurring Food/Iron income.
	type scoredTile struct {
		id    int
		score float64
	}
	owned := []int{}
	isFront := map[int]bool{}
	frontTiles := []scoredTile{}
	expandTiles := []scoredTile{}
	foodIncome, ironIncome := 0, 0
	nearbyEnemyTanks := false
	for _, t := range g.Territories {
		if t.Owner != p.ID {
			continue
		}
		owned = append(owned, t.ID)
		if t.Resource == "food" {
			foodIncome += t.Yield
		} else {
			ironIncome += t.Yield
		}
		adjEnemyPower := 0.0
		hasEnemy, hasNeutral := false, false
		bestNeutralYield := 0.0
		for _, n := range t.Neighbors {
			nt := g.Territories[n]
			if nt.Owner == "" {
				hasNeutral = true
				if float64(nt.Yield) > bestNeutralYield {
					bestNeutralYield = float64(nt.Yield)
				}
			} else if nt.Owner != p.ID {
				hasEnemy = true
				adjEnemyPower += enemyPowerInSlice(snapshot, n, p.ID)
				if enemyHasTanksInSlice(snapshot, n, p.ID) {
					nearbyEnemyTanks = true
				}
			}
		}
		if hasEnemy {
			isFront[t.ID] = true
			score := adjEnemyPower - 0.4*g.sectorOwnPower(t.ID, p.ID) + 0.3*float64(t.Yield)
			frontTiles = append(frontTiles, scoredTile{t.ID, score})
		}
		if hasNeutral {
			score := bestNeutralYield + 0.2*float64(t.Yield) - 0.1*g.sectorOwnPower(t.ID, p.ID)
			expandTiles = append(expandTiles, scoredTile{t.ID, score})
		}
	}
	if len(owned) == 0 {
		g.autoFeed(p)
		return
	}

	backTiles := []scoredTile{}
	for _, tid := range owned {
		if isFront[tid] {
			continue
		}
		frontAdj := 0
		frontThreat := 0.0
		for _, n := range g.Territories[tid].Neighbors {
			if isFront[n] {
				frontAdj++
				for _, nn := range g.Territories[n].Neighbors {
					if g.Territories[nn].Owner != "" && g.Territories[nn].Owner != p.ID {
						frontThreat += enemyPowerInSlice(snapshot, nn, p.ID)
					}
				}
			}
		}
		if frontAdj > 0 {
			backTiles = append(backTiles, scoredTile{tid, frontThreat + 4.0*float64(frontAdj)})
		}
	}

	sortDesc := func(s []scoredTile) {
		sort.SliceStable(s, func(i, j int) bool {
			if math.Abs(s[i].score-s[j].score) > 1e-9 {
				return s[i].score > s[j].score
			}
			return s[i].id < s[j].id
		})
	}
	sortDesc(frontTiles)
	sortDesc(expandTiles)
	sortDesc(backTiles)

	fallback := owned[0]
	if p.Capital >= 0 && p.Capital < len(g.Territories) && g.Territories[p.Capital].Owner == p.ID {
		fallback = p.Capital
	}

	buildTargets := func(kind string) []int {
		seen := map[int]bool{}
		out := []int{}
		add := func(id int) {
			if id >= 0 && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
		if kind == "artillery" {
			for _, st := range backTiles {
				add(st.id)
			}
			for _, st := range frontTiles {
				add(st.id)
			}
			for _, st := range expandTiles {
				add(st.id)
			}
		} else {
			for _, st := range frontTiles {
				add(st.id)
			}
			for _, st := range expandTiles {
				add(st.id)
			}
		}
		add(fallback)
		for _, id := range owned {
			add(id)
		}
		return out
	}

	// 3. Recruit combined arms within a sustainable Food ceiling and Currency reserve.
	infCount, artCount, tankCount := 0, 0, 0
	for _, u := range g.Units {
		if u.Owner != p.ID || u.Count <= 0 {
			continue
		}
		switch u.Type {
		case "tank":
			tankCount += u.Count
		case "artillery":
			artCount += u.Count
		default:
			infCount += u.Count
		}
	}
	totalTroops := infCount + artCount + tankCount
	moveReserve := max(12, p.Resources.Currency*30/100)
	// Base recruitment cap on recurring food income + modest reserve buffer so bots
	// don't burn their starting stockpile and starve on Rounds 3-4.
	bankedSurplus := max(0, p.Resources.Food-totalTroops)
	sustainableCap := foodIncome + min(6, bankedSurplus/4)
	if sustainableCap > p.Resources.Food {
		sustainableCap = p.Resources.Food
	}
	foodHeadroom := max(0, sustainableCap-totalTroops)

	recruitOne := func(kind string) bool {
		if foodHeadroom < 1 {
			return false
		}
		c := cost(kind, 1)
		if p.Resources.Humans < c.Humans || p.Resources.Iron < c.Iron || p.Resources.Currency-c.Currency < moveReserve {
			return false
		}
		for _, tid := range buildTargets(kind) {
			for _, u := range g.Units {
				if u.Owner == p.ID && u.Territory == tid && u.Type == kind && !u.Fed && u.Count > 0 && u.Count < 100 {
					if !spend(p, c) {
						return false
					}
					u.Health = (u.Health*float64(u.Count) + 100) / float64(u.Count+1)
					u.Count++
					foodHeadroom--
					totalTroops++
					return true
				}
			}
			if g.sectorStackCount(tid) < 6 {
				if !spend(p, c) {
					return false
				}
				g.Units = append(g.Units, &Unit{uid(), p.ID, kind, tid, 1, 100, 1, false})
				foodHeadroom--
				totalTroops++
				return true
			}
		}
		return false
	}

	for step := 0; step < 24 && foodHeadroom > 0; step++ {
		needInf := infCount < (tankCount+artCount)*2
		artTarget := float64(totalTroops) / 4.0
		if nearbyEnemyTanks {
			artTarget = float64(totalTroops) / 3.0
		}
		tankTarget := float64(totalTroops) / 5.0
		artDeficit := artTarget - float64(artCount)
		tankDeficit := tankTarget - float64(tankCount)
		// Save 4-7 iron for a Tank when Tanks are more needed than Artillery and iron is flowing.
		saveIronForTank := tankDeficit > artDeficit && p.Resources.Iron >= 4 && p.Resources.Iron < 8 && ironIncome >= 4
		var prefs []string
		switch {
		case needInf:
			if saveIronForTank {
				prefs = []string{"infantry", "tank"}
			} else {
				prefs = []string{"infantry", "artillery", "tank"}
			}
		case artDeficit > 0 && artDeficit >= tankDeficit:
			prefs = []string{"artillery", "tank", "infantry"}
		case tankDeficit > 0:
			if saveIronForTank {
				prefs = []string{"tank", "infantry"}
			} else {
				prefs = []string{"tank", "artillery", "infantry"}
			}
		default:
			prefs = []string{"tank", "artillery", "infantry"}
		}
		bought := false
		for _, kind := range prefs {
			if recruitOne(kind) {
				switch kind {
				case "tank":
					tankCount++
				case "artillery":
					artCount++
				default:
					infCount++
				}
				bought = true
				break
			}
		}
		if !bought {
			break
		}
	}

	// 4. Heal wounded priority stacks with Extra rations when surplus food safely allows.
	unfed := []*Unit{}
	remainingNormal := 0
	for _, u := range g.Units {
		if u.Owner == p.ID && !u.Fed && u.Count > 0 {
			unfed = append(unfed, u)
			remainingNormal += u.Count
		}
	}
	sort.SliceStable(unfed, func(i, j int) bool {
		pi, pj := unitPriority(unfed[i].Type), unitPriority(unfed[j].Type)
		if pi != pj {
			return pi > pj
		}
		return unfed[i].Health < unfed[j].Health
	})
	for _, u := range unfed {
		if u.Health >= 85 {
			continue
		}
		extraFood := int(math.Ceil(float64(u.Count) * 1.5))
		surcharge := extraFood - u.Count
		if p.Resources.Food >= remainingNormal+surcharge {
			p.Resources.Food -= extraFood
			remainingNormal -= u.Count
			u.Fed = true
			u.Ration = 1.5
			u.Health = math.Min(100, u.Health+20)
		}
	}

	// 5. Feed all remaining unfed stacks via priority two-pass autoFeed.
	g.autoFeed(p)
}

func (g *Game) distToFront(owner string, enemyOnly bool) []int {
	dist := make([]int, len(g.Territories))
	for i := range dist {
		dist[i] = -1
	}
	q := []int{}
	for _, t := range g.Territories {
		if t.Owner != owner {
			continue
		}
		for _, n := range t.Neighbors {
			no := g.Territories[n].Owner
			if no != owner && (!enemyOnly || no != "") {
				dist[t.ID] = 0
				q = append(q, t.ID)
				break
			}
		}
	}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, n := range g.Territories[cur].Neighbors {
			if g.Territories[n].Owner == owner && dist[n] < 0 {
				dist[n] = dist[cur] + 1
				q = append(q, n)
			}
		}
	}
	return dist
}

func (g *Game) botOrders() {
	for _, p := range g.Players {
		if !p.Bot || g.eliminated(p.ID) {
			continue
		}
		distAny := g.distToFront(p.ID, false)
		distEnemy := g.distToFront(p.ID, true)
		totalTroops := 0
		for _, u := range g.Units {
			if u.Owner == p.ID && u.Count > 0 {
				totalTroops += u.Count
			}
		}
		artSupport := map[int]float64{}

		// Pass 1: Artillery orders (real enemy targets by capped effective damage, or backline support).
		for _, u := range g.Units {
			if u.Owner != p.ID || u.Count <= 0 || u.Type != "artillery" {
				continue
			}
			bestTarget, bestScore := -1, 0.0
			for _, n := range g.Territories[u.Territory].Neighbors {
				nt := g.Territories[n]
				score := 0.0
				if nt.Owner != p.ID {
					oppWeight := 0.0
					for _, v := range g.Units {
						if v.Territory == n && v.Owner != p.ID && v.Count > 0 && v.Health > 0 {
							oppWeight += float64(v.Count) * unitHP(v.Type) * (v.Health / 100)
						}
					}
					if oppWeight > 0 {
						// Score by actual effective damage absorbed without overkill so large stacks
						// outrank lone wounded stragglers while Tanks remain top-priority targets.
						for _, v := range g.Units {
							if v.Territory == n && v.Owner != p.ID && v.Count > 0 && v.Health > 0 {
								capHP := float64(v.Count) * unitHP(v.Type) * (v.Health / 100)
								raw := attack(u, v.Type, false) * (capHP / oppWeight)
								score += math.Min(raw, capHP)
							}
						}
					}
				} else if threat := g.sectorThreat(n, p.ID); threat > 0 && (g.sectorOwnPower(n, p.ID) > 0 || n == p.Capital) {
					// Speculative backline fire support onto a defended friendly border sector
					// scores conservatively (0.55x) so real adjacent enemy armies take priority.
					targetType := "infantry"
					for _, nn := range nt.Neighbors {
						if g.enemyHasTanksAt(nn, p.ID) {
							targetType = "tank"
							break
						}
					}
					score = math.Min(attack(u, targetType, false), threat) * 0.55
				}
				if score > bestScore {
					bestScore = score
					bestTarget = n
				}
			}
			if bestTarget >= 0 {
				g.commandBot(u, "bombard", bestTarget)
				artSupport[bestTarget] += bestScore
				continue
			}
			// Advance rear artillery toward 1 hop behind an active enemy front (not stranded behind neutrals).
			artDist := distEnemy
			if artDist[u.Territory] < 0 {
				artDist = distAny
			}
			if artDist[u.Territory] >= 2 {
				bestStep, bestStepDist := -1, artDist[u.Territory]
				for _, n := range g.Territories[u.Territory].Neighbors {
					if g.Territories[n].Owner == p.ID && artDist[n] >= 0 && artDist[n] < bestStepDist {
						bestStepDist = artDist[n]
						bestStep = n
					}
				}
				if bestStep >= 0 && g.commandBot(u, "move", bestStep) {
					continue
				}
			}
			g.commandBot(u, "defend", u.Territory)
		}

		// Group non-artillery combat stacks by territory.
		frontGroups := map[int][]*Unit{}
		frontIDs := []int{}
		rearUnits := []*Unit{}
		for _, u := range g.Units {
			if u.Owner != p.ID || u.Count <= 0 || u.Type == "artillery" {
				continue
			}
			if distAny[u.Territory] == 0 {
				if _, ok := frontGroups[u.Territory]; !ok {
					frontIDs = append(frontIDs, u.Territory)
				}
				frontGroups[u.Territory] = append(frontGroups[u.Territory], u)
			} else {
				rearUnits = append(rearUnits, u)
			}
		}

		// Prioritize enemy-bordering fronts over neutral-only borders so attacks aren't starved of currency.
		sort.SliceStable(frontIDs, func(i, j int) bool {
			ei := distEnemy[frontIDs[i]] == 0
			ej := distEnemy[frontIDs[j]] == 0
			if ei != ej {
				return ei
			}
			ti, tj := g.sectorThreat(frontIDs[i], p.ID), g.sectorThreat(frontIDs[j], p.ID)
			if math.Abs(ti-tj) > 1e-9 {
				return ti > tj
			}
			return frontIDs[i] < frontIDs[j]
		})

		claimedUndefended := map[int]bool{}

		// Pass 2: Frontline combat coordination (attack >= 1.25x matchup odds, safe single-stack expansion, or entrench).
		for _, tid := range frontIDs {
			stacks := frontGroups[tid]
			sort.SliceStable(stacks, func(i, j int) bool {
				return attack(stacks[i], "", true) > attack(stacks[j], "", true)
			})
			ownPower := 0.0
			for _, u := range stacks {
				ownPower += attack(u, "", true)
			}
			threat := g.sectorThreat(tid, p.ID)
			protectArt := threat > 0 && g.sectorHasArtillery(tid, p.ID)

			bestEnemy, bestEnemyScore := -1, -1e9
			bestEmptyEnemy, bestEmptyEnemyScore := -1, -1e9
			bestNeutral, bestNeutralScore := -1, -1e9
			for _, n := range g.Territories[tid].Neighbors {
				nt := g.Territories[n]
				if nt.Owner == p.ID {
					continue
				}
				if nt.Owner == "" {
					if claimedUndefended[n] {
						continue
					}
					sc := float64(nt.Yield)
					if nt.Resource == "food" && p.Resources.Food < totalTroops*2 {
						sc += 4.0
					}
					if sc > bestNeutralScore {
						bestNeutralScore = sc
						bestNeutral = n
					}
					continue
				}
				ePower := g.enemyPowerAt(n, p.ID)
				if ePower == 0 {
					if claimedUndefended[n] {
						continue
					}
					sc := float64(nt.Yield) + 6.0
					if sc > bestEmptyEnemyScore {
						bestEmptyEnemyScore = sc
						bestEmptyEnemy = n
					}
					continue
				}
				offense := g.expectedMatchupPower(stacks, n, p.ID) + artSupport[n]*0.35
				// Require 1.25x matchup-adjusted advantage to beat potential +20% entrenchment.
				if offense >= ePower*1.25 {
					sc := (offense - ePower) + float64(nt.Yield)*0.8
					if sc > bestEnemyScore {
						bestEnemyScore = sc
						bestEnemy = n
					}
				}
			}

			// 2a. Decisive attack on a garrisoned enemy sector.
			if bestEnemy >= 0 {
				ePower := g.enemyPowerAt(bestEnemy, p.ID)
				// If protecting stationed artillery from other enemies, keep at least 1 garrison stack.
				attackPool := stacks
				if protectArt && len(stacks) > 1 {
					attackPool = stacks[:len(stacks)-1]
				} else if protectArt && threat > ePower*1.05 {
					attackPool = nil
				}
				affordable := []*Unit{}
				remCurrency := p.Resources.Currency - g.reserved(p.ID, "")
				for _, u := range attackPool {
					mc := g.moveCost(u, bestEnemy)
					if mc <= remCurrency {
						remCurrency -= mc
						affordable = append(affordable, u)
					}
				}
				if len(affordable) > 0 && g.expectedMatchupPower(affordable, bestEnemy, p.ID)+artSupport[bestEnemy]*0.35 >= ePower*1.25 {
					sent := map[string]bool{}
					for _, u := range affordable {
						if g.commandBot(u, "move", bestEnemy) {
							sent[u.ID] = true
						}
					}
					for _, u := range stacks {
						if !sent[u.ID] {
							g.commandBot(u, "defend", tid)
						}
					}
					continue
				}
			}

			// 2b. Opportunistic single-stack capture of an undefended enemy or neutral tile.
			freeTarget := bestEmptyEnemy
			if freeTarget < 0 {
				freeTarget = bestNeutral
			}
			if freeTarget >= 0 {
				// Never abandon stationed artillery or a heavily threatened sector with our only stack.
				if len(stacks) == 1 && (protectArt || threat > ownPower*0.8) {
					g.commandBot(stacks[0], "defend", tid)
					continue
				}
				// Send only the smallest/cheapest affordable stack to claim undefended tiles;
				// keep heavier stacks entrenched on the front line.
				claimIdx := -1
				for i := len(stacks) - 1; i >= 0; i-- {
					if g.commandBot(stacks[i], "move", freeTarget) {
						claimIdx = i
						claimedUndefended[freeTarget] = true
						break
					}
				}
				for i, u := range stacks {
					if i != claimIdx {
						g.commandBot(u, "defend", tid)
					}
				}
				continue
			}

			// 2c. Hold & entrench when outgunned or lacking favorable attacks.
			for _, u := range stacks {
				g.commandBot(u, "defend", tid)
			}
		}

		// Pass 3: Rear infantry and tanks advance along free friendly routes to the front.
		for _, u := range rearUnits {
			moveDist := distEnemy
			if moveDist[u.Territory] < 0 {
				moveDist = distAny
			}
			if moveDist[u.Territory] < 1 {
				g.commandBot(u, "defend", u.Territory)
				continue
			}
			bestStep, bestStepScore := -1, -1e9
			for _, n := range g.Territories[u.Territory].Neighbors {
				if g.Territories[n].Owner == p.ID && moveDist[n] >= 0 && moveDist[n] < moveDist[u.Territory] {
					sc := -float64(moveDist[n])*10 + g.sectorThreat(n, p.ID) - 0.2*g.sectorOwnPower(n, p.ID)
					if sc > bestStepScore {
						bestStepScore = sc
						bestStep = n
					}
				}
			}
			if bestStep >= 0 && g.commandBot(u, "move", bestStep) {
				continue
			}
			g.commandBot(u, "defend", u.Territory)
		}
	}
}

func (g *Game) commandBot(u *Unit, kind string, target int) bool {
	p := g.player(u.Owner)
	if p == nil {
		return false
	}
	if kind == "defend" {
		target = u.Territory
	} else if !g.adjacent(u.Territory, target) {
		return false
	}
	if kind == "bombard" && u.Type != "artillery" {
		return false
	}
	if kind == "move" && g.reserved(p.ID, u.ID)+g.moveCost(u, target) > p.Resources.Currency {
		return false
	}
	g.Orders[u.ID] = Order{u.ID, kind, target}
	return true
}
