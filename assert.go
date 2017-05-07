package main

import "log"

func Assert(b bool, msg string) {
	if !b {
		log.Fatalln(msg)
	}
}
