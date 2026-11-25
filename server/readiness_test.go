package main

import (
	"testing"
	"time"
)

func TestGatherHasNoDeadlineAndAdvancesOnLock(t *testing.T) {
	g := fixture()
	g.Phase = "simulate"
	g.advance()
	if g.Phase != "gather" || g.Deadline != 0 {
		t.Fatal("Gather must have no deadline")
	}
	g.tick()
	if g.Phase != "gather" {
		t.Fatal("Gather advanced before locks")
	}
	for _, p := range g.Players {
		if err := g.command(p.ID, Command{Action: "ready"}); err != nil {
			t.Fatal(err)
		}
	}
	if g.Phase != "plan" || g.Round != 2 {
		t.Fatal("All locks must advance Gather")
	}
}
func TestAllLocksSkipTimedPhases(t *testing.T) {
	g := fixture()
	for _, p := range g.Players {
		g.command(p.ID, Command{Action: "ready"})
	}
	if g.Phase != "simulate" || g.Deadline <= time.Now().UnixMilli() {
		t.Fatal("Plan did not advance before timer")
	}
	for _, p := range g.Players {
		g.command(p.ID, Command{Action: "ready"})
	}
	if g.Phase != "gather" || g.Deadline != 0 {
		t.Fatal("Simulation locks did not advance to untimed Gather")
	}
}
func TestEliminatedCommanderDoesNotBlock(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	out := g.addPlayer("Eliminated", false)
	if !g.eliminated(out.ID) {
		t.Fatal("Commander without territory or armies is not eliminated")
	}
	g.command(a.ID, Command{Action: "ready"})
	if g.Phase != "plan" {
		t.Fatal("Advanced before remaining active player locked")
	}
	g.command(b.ID, Command{Action: "ready"})
	if g.Phase != "simulate" {
		t.Fatal("Eliminated commander blocked Plan")
	}
	ps := g.view(a.ID)["players"].([]Player)
	if !ps[2].Eliminated {
		t.Fatal("Client elimination status missing")
	}
	g.command(a.ID, Command{Action: "ready"})
	g.command(b.ID, Command{Action: "ready"})
	g.command(a.ID, Command{Action: "ready"})
	g.command(b.ID, Command{Action: "ready"})
	if g.Phase != "plan" || g.Round != 2 {
		t.Fatal("Eliminated commander blocked Gather")
	}
}
func TestSurvivingArmyKeepsCommanderActive(t *testing.T) {
	g := fixture()
	p := g.addPlayer("Survivor", false)
	g.Units = append(g.Units, &Unit{ID: "survivor", Owner: p.ID, Count: 1, Health: 100})
	if g.eliminated(p.ID) {
		t.Fatal("Surviving army ignored")
	}
}
