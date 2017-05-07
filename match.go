package main

import "log"

var matchQueue = make(chan *Player, 1024)

func init() {
	log.Println("Match subsystem inited.")
	go MatchWatcher()
}

func MatchWatcher() {
	for {
		a := <- matchQueue
		a.StopMatchSignal = make(chan struct{})
		select {
		case <- a.StopMatchSignal:
			a.StopMatchSignal = nil
			continue
		case b := <- matchQueue:
			NewGame(a, b)
		}
	}
}

func (p *Player) StartMatch() {
	matchQueue <- p
	p.State = PLAYER_STATE_MATCHING
	p.StateNotify(PLAYER_STATE_MATCHING, nil)
}

func (p *Player) StopMatch() {
	p.StopMatchSignal <- struct{}{}
	p.State = PLAYER_STATE_INLINE
	p.StateNotify(PLAYER_STATE_INLINE, "Matching stop!")
}

// Post a change state message to player only called by match and game module
func (p *Player) SuddenChangeState(state int, data interface{}) {
	p.MessageBox <- PlayerMessage{PLAYER_MSG_STATECHANGE, state, data}
}


func (p *Player) _handleSuddenStateChange(m PlayerMessage) {
	p.State = m.Value
	switch m.Value {
	case PLAYER_STATE_WAITING_CLIENT:
		g, ok := m.Data.(*Game)
		if !ok {
			panic("No game")
		}
		p.CurGame = g
		isA := false
		if p == p.CurGame.Players[0] {
			isA = true
		}
		p.StateNotify(PLAYER_STATE_WAITING_CLIENT, isA)
	case PLAYER_STATE_GAMING:
		p.StateNotify(PLAYER_STATE_GAMING, nil)
	case PLAYER_STATE_INLINE:
		reason, ok := m.Data.(string)
		if !ok {
			panic("no reason")
		}
		p.CurGame = nil
		p.StateNotify(PLAYER_STATE_INLINE, reason)
	case PLAYER_STATE_WAITING_CLIENT2:
		p.StateNotify(PLAYER_STATE_WAITING_CLIENT2, nil)
	}
}