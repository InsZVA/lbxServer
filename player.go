package main

import (
	"net"
)

const (
	PLAYER_STATE_OFFLINE = 0
	PLAYER_STATE_INLINE = 1
	PLAYER_STATE_MATCHING = 2
	PLAYER_STATE_WAITING_CLIENT = 3
	PLAYER_STATE_READY = 4
	PLAYER_STATE_GAMING = 5
	PLAYER_STATE_WAITING_CLIENT2 = 6

	PLAYER_MSG_STATECHANGE = 0

)

type PlayerMessage struct {
	Type int
	Value int
	Data interface{}
}

type Player struct {
	State int
	CurGame *Game

	// only can be assigned by Match module
	StopMatchSignal chan struct{}

	StopSignal	chan struct{}
	MessageBox chan PlayerMessage
	Conn *net.TCPConn
}

func NewPlayer(conn *net.TCPConn) *Player {
	p := &Player{
		MessageBox: make(chan PlayerMessage, 4),
		Conn: conn,
		State: PLAYER_STATE_INLINE,
	}
	p.StateNotify(PLAYER_STATE_INLINE, nil)
	go p.Start()
	return p
}

func (p *Player) Start() {
	for {
		select {
		case m := <- p.MessageBox:
			switch m.Type {
			case PLAYER_MSG_STATECHANGE:
				p._handleSuddenStateChange(m)
			}
		case <- p.StopSignal:
			if p.StopMatchSignal != nil {
				p.StopMatchSignal <- struct {}{}
			}
			if p.State == PLAYER_STATE_GAMING {
				p.CurGame.Exit(p)
			}
		}
	}
}

func (p *Player) Stop() {
	p.StopSignal <- struct{}{}
}