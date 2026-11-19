package main

import (
	"testing"
	"time"
)

func testHub(g *Game) *hub {
	return &hub{games: map[string]*Game{g.Code: g}, clients: map[*client]bool{}}
}
func TestFiveMinuteGraceAndReconnect(t *testing.T) {
	g := fixture()
	p := g.Players[0]
	h := testHub(g)
	now := time.Now()
	socket := &client{game: g, player: p.ID, out: make(chan any, 10)}
	h.disconnected(socket, now)
	if p.DisconnectedUntil != now.Add(5*time.Minute).UnixMilli() {
		t.Fatal("Incorrect grace deadline")
	}
	h.expireDisconnected(now.Add(5*time.Minute - time.Millisecond))
	if g.player(p.ID) == nil {
		t.Fatal("Removed before deadline")
	}
	h.clients[socket] = true
	h.expireDisconnected(now.Add(6 * time.Minute))
	if g.player(p.ID) == nil || p.DisconnectedUntil != 0 {
		t.Fatal("Reconnect did not cancel timer")
	}
	delete(h.clients, socket)
	h.disconnected(socket, now.Add(7*time.Minute))
	h.expireDisconnected(now.Add(12 * time.Minute))
	if g.player(p.ID) != nil {
		t.Fatal("Expired player not removed")
	}
	if h.games[g.Code] == nil {
		t.Fatal("Lobby containing another player disbanded")
	}
}
func TestDisconnectExpiryNeutralizesAndRetainsConnectedPlayer(t *testing.T) {
	g := fixture()
	a, b := g.Players[0], g.Players[1]
	g.Units = []*Unit{{ID: "army", Owner: a.ID, Type: "infantry", Territory: 0, Count: 4, Health: 100, Ration: 1}}
	g.Orders["army"] = Order{"army", "move", 1}
	h := testHub(g)
	live := &client{game: g, player: b.ID, out: make(chan any, 10)}
	h.clients[live] = true
	now := time.Now()
	h.disconnected(&client{game: g, player: a.ID}, now)
	h.expireDisconnected(now.Add(5 * time.Minute))
	if g.unit("army") != nil {
		t.Fatal("Army retained")
	}
	for _, tr := range g.Territories {
		if tr.Owner == a.ID {
			t.Fatal("Territory not neutralized")
		}
	}
	if h.games[g.Code] == nil || g.player(b.ID) == nil {
		t.Fatal("Connected player lost lobby")
	}
	h.expireDisconnected(now.Add(48 * time.Hour))
	if h.games[g.Code] == nil {
		t.Fatal("Connected idle lobby evicted")
	}
}
func TestAllDisconnectedRoomExpiresAndDuplicateSocketDoesNotStartTimer(t *testing.T) {
	g := newGame("ROOM")
	p := g.addPlayer("Human", false)
	g.addPlayer("Bot", true)
	h := testHub(g)
	one := &client{game: g, player: p.ID, out: make(chan any, 4)}
	h.clients[one] = true
	now := time.Now()
	h.disconnected(&client{game: g, player: p.ID}, now)
	if p.DisconnectedUntil != 0 {
		t.Fatal("Timer started while another socket was connected")
	}
	delete(h.clients, one)
	h.disconnected(one, now)
	h.expireDisconnected(now.Add(4 * time.Minute))
	if h.games[g.Code] == nil {
		t.Fatal("Room evicted before grace expired")
	}
	h.expireDisconnected(now.Add(5 * time.Minute))
	if h.games[g.Code] != nil {
		t.Fatal("Abandoned bot-only room retained")
	}
}
