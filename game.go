package main

import (
	"time"
)

const (
	GAME_WAITING = 0
	GAME_ROUND_1 = 1
	GAME_ROUND_2 = 2
	GAME_END = 3
)

type Game struct {
	Round      int32
	Players    []*Player
	// start time of this round
	CurStartTime    time.Time
	PlayedTime []time.Duration
	Ready      chan struct{}
	EndSignal chan struct{}
	ExitSignal chan struct{}
	ExitReason [2]string
}

func NewGame(a, b *Player) {
	game := &Game{
		Players: []*Player{a, b},
		PlayedTime: []time.Duration{time.Duration(0), time.Duration(0)},
		Ready: make(chan struct{}),
		EndSignal: make(chan struct{}),
		ExitSignal: make(chan struct{}),
	}
	a.SuddenChangeState(PLAYER_STATE_WAITING_CLIENT, game)
	b.SuddenChangeState(PLAYER_STATE_WAITING_CLIENT, game)
	go game.Start()
}

func (g *Game) Start() {
	select {
	case <-g.Ready:
	case <-g.ExitSignal:
		g.Clean()
		return
	}
	select {
	case <-g.Ready:
	case <-g.ExitSignal:
		g.Clean()
		return
	}

	// Start game
	for _, p := range g.Players {
		p.SuddenChangeState(PLAYER_STATE_GAMING, nil)
	}
	g.Round = GAME_ROUND_1
	g.CurStartTime = time.Now()

	// Playing...
	// Waiting for end...

	select {
	case <-g.ExitSignal:
		g.Clean()
		return
	case <-g.EndSignal:
		g.PlayedTime[0] = time.Now().Sub(g.CurStartTime)
		g.Round = GAME_ROUND_2
	}

	// Waiting for round2 client ready
	for _, p := range g.Players {
		p.SuddenChangeState(PLAYER_STATE_WAITING_CLIENT2, nil)
	}

	select {
	case <-g.Ready:
	case <-g.ExitSignal:
		g.Clean()
		return
	}
	select {
	case <-g.Ready:
	case <-g.ExitSignal:
		g.Clean()
		return
	}

	// start round2
	for _, p := range g.Players {
		p.SuddenChangeState(PLAYER_STATE_GAMING, nil)
	}
	g.Round = GAME_ROUND_2
	g.CurStartTime = time.Now()

	// Playing...
	// Waiting for end...
	select {
	case <-g.ExitSignal:
		g.Clean()
		return
	case <-g.EndSignal:
		g.PlayedTime[1] = time.Now().Sub(g.CurStartTime)
	}

	if g.PlayedTime[0] > g.PlayedTime[1] {
		g.ExitReason[0] = "You win!"
		g.ExitReason[1] = "You lose!"
	} else {
		g.ExitReason[1] = "You win!"
		g.ExitReason[0] = "You lose!"
	}
	g.Clean()
}

func (g *Game) Clean() {
	for i, p := range g.Players {
		if p != nil {
			p.SuddenChangeState(PLAYER_STATE_INLINE, g.ExitReason[i])
		}
	}
	g.Round = GAME_END
}

func (g *Game) Exit(bad *Player) {
	for i, p := range g.Players {
		if p == bad {
			g.ExitReason[i] = "You has exited the game!"
		} else {
			g.ExitReason[i] = "Your enemy escaped!"
		}
	}
	g.ExitSignal <- struct{}{}
}

func (g *Game) End() {
	g.EndSignal <- struct{}{}
}

func (p *Player) Forward(msg *Message) {
	Assert(p.State == PLAYER_STATE_GAMING, "Not gaming client")
	for _, p2 := range p.CurGame.Players {
		if p2 != p {
			msg.Type = MESSAGE_TYPE_FORWARDED_DATA
			_, err := p2.Conn.Write(msg.Dump())
			if err != nil {
				p2.Stop()
			}
		}
	}
}

func (p *Player) Ready() {
	Assert(p.State == PLAYER_STATE_WAITING_CLIENT ||
		p.State == PLAYER_STATE_WAITING_CLIENT2, "Not waiting client")
	p.State = PLAYER_STATE_READY
	p.CurGame.Ready <- struct {}{}
}

