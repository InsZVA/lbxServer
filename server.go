package main

import (
	"net"
	"log"
	"runtime/debug"
)

const (
	// Client -> Server
	MESSAGE_TYPE_CONNECT        = 0x00
	MESSAGE_TYPE_DISCONNECT     = 0x01
	MESSAGE_TYPE_START_MATCHING = 0x02
	MESSAGE_TYPE_END_MATCHING   = 0x03
	MESSAGE_TYPE_ENTER_GAME     = 0x04
	MESSAGE_TYPE_EXIT_GAME      = 0x05
	MESSAGE_TYPE_FORWARD_DATA   = 0x06
	MESSAGE_TYPE_GAME_END       = 0x07
	// Server -> Client
	MESSAGE_TYPE_STATE_NOTIFY = 0xf0
	MESSAGE_TYPE_FORWARDED_DATA = 0xf1

	// Attr
	MESSAGE_ATTR_TYPE_STATE = 0x01
	MESSAGE_ATTR_TYPE_ISA = 0x02
	MESSAGE_ATTR_TYPE_REASON = 0x03
)

//
// Message {
// 	Type uint8
// 	Len  uint8
//  Res uint16 // reserved
// 	Attr { // 4 bytes padding
//   Type uint8
//	 Len uint8
//	 Data ...
//  }...
//
// all len is contain header

func Handler(conn *net.TCPConn) {
	defer func () {
		err := recover()
		log.Println(err)
		log.Println(string(debug.Stack()))
		conn.Close()
	} ()

	buff := make([]byte, 256)
	var p *Player
	n, err := conn.Read(buff)
	if err != nil {
		return
	}
	m := ParseMessage(buff[:n])
	if m == nil {
		return
	}
	if m.Type != MESSAGE_TYPE_CONNECT {
		return
	}
	p = NewPlayer(conn)
	for p.State != PLAYER_STATE_OFFLINE {
		n, err := conn.Read(buff)
		if err != nil {
			if p != nil {
				p.Stop()
				return
			}
			return
		}
		p.TranslateMessage(buff[:n])
	}
}

func main() {
	laddr, _ := net.ResolveTCPAddr("tcp4", "0.0.0.0:6789")
	listenConn, err := net.ListenTCP("tcp4", laddr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listenConn.AcceptTCP()
		if err != nil { continue }
		go Handler(conn)
	}
}

type MessageAttr struct {
	Type uint8
	Len uint8
	Data []byte
}

type Message struct {
	Type uint8
	Len uint8
	Res uint16
	Attrs []MessageAttr
}

func ParseAttribute(data []byte) *MessageAttr {
	if len(data) < 4 {
		return nil
	}
	attr := &MessageAttr{
		Type: data[0],
		Len: data[1],
	}
	if uint8(len(data)) < attr.Len {
		return nil
	}
	attr.Data = data[2:attr.Len]
	return attr
}

func ParseMessage(data []byte) *Message {
	if len(data) < 4 {
		return nil
	}
	m := &Message{
		Type: data[0],
		Len: data[1],
		Attrs: []MessageAttr{},
	}
	off := uint8(4)
	for attr := ParseAttribute(data[off:]); attr != nil; attr = ParseAttribute(data[off:]) {
		m.Attrs = append(m.Attrs, *attr)
		off += attr.Len
	}
	return m
}

func NewMessage(tp uint8) *Message {
	return &Message{Type: tp, Attrs:[]MessageAttr{}}
}

func (m *Message) AddAttr(tp uint8, data []byte) {
	m.Attrs = append(m.Attrs, MessageAttr{
		tp,
		uint8((len(data) + 2) + 3) & 0xfc,
		data,
	})
}

func (m *Message) Dump() []byte {
	buff := make([]byte, 4)
	buff[0] = m.Type
	for _, attr := range m.Attrs {
		buff2 := make([]byte, attr.Len)
		buff2[0] = attr.Type
		buff2[1] = attr.Len
		copy(buff2[2:], attr.Data)
		buff = append(buff, buff2...)
	}
	buff[1] = uint8(len(buff))
	return buff
}

func (p *Player) StateNotify(state int, data interface{}) {
	msg := NewMessage(MESSAGE_TYPE_STATE_NOTIFY)
	stateB := []byte{byte(state)}
	msg.AddAttr(MESSAGE_ATTR_TYPE_STATE, stateB)
	switch state {
	case PLAYER_STATE_WAITING_CLIENT:
		isA, ok := data.(bool)
		if !ok {
			panic("state notify data type error")
		}
		isa := make([]byte, 1)
		if isA {
			isa[0] = 1
		} else {
			isa[0] = 0
		}
		msg.AddAttr(MESSAGE_ATTR_TYPE_ISA, isa)
		_, err := p.Conn.Write(msg.Dump())
		if err != nil {
			p.Stop()
		}
	case PLAYER_STATE_INLINE:
		reason, ok := data.(string)
		if !ok {
			//panic("state notify data type error")
		} else {
			msg.AddAttr(MESSAGE_ATTR_TYPE_REASON, []byte(reason))
		}
	default:
	}
	p.Conn.Write(msg.Dump())
}

// parse the tcp message from client
func (p *Player) TranslateMessage(buff []byte) {
	m := ParseMessage(buff)
	if m == nil {
		p.Stop()
		return
	}
	switch m.Type {
	case MESSAGE_TYPE_CONNECT:
		return
	case MESSAGE_TYPE_START_MATCHING:
		if p.State == PLAYER_STATE_INLINE {
			p.StartMatch()
		}
	case MESSAGE_TYPE_ENTER_GAME:
		if p.State == PLAYER_STATE_WAITING_CLIENT ||
			p.State == PLAYER_STATE_WAITING_CLIENT2 {
			p.Ready()
		}
	case MESSAGE_TYPE_FORWARD_DATA:
		if p.State == PLAYER_STATE_GAMING {
			p.Forward(m)
		}
	case MESSAGE_TYPE_EXIT_GAME:
		if p.State == PLAYER_STATE_GAMING {
			p.CurGame.Exit(p)
		}
	case MESSAGE_TYPE_DISCONNECT:
		p.Stop()
		return
	case MESSAGE_TYPE_END_MATCHING:
		p.StopMatch()
	case MESSAGE_TYPE_GAME_END:
		p.CurGame.End()
	}
}