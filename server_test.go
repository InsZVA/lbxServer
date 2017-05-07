package main

import (
	"testing"
	"fmt"
)

var dump []byte

func TestMessage_Dump(t *testing.T) {
	msg := NewMessage(MESSAGE_TYPE_STATE_NOTIFY)
	msg.AddAttr(MESSAGE_ATTR_TYPE_STATE, []byte{1})
	msg.AddAttr(MESSAGE_ATTR_TYPE_ISA, []byte{1})
	dump = msg.Dump()
	fmt.Println(dump)
}

func TestParseMessage(t *testing.T) {
	msg := ParseMessage(dump)
	fmt.Println(msg)
}
