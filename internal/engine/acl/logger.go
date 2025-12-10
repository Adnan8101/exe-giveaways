package acl

import (
	"fmt"
)

var logQueue = make(chan string, 5000)

func PushLog(msg string) {
	select {
	case logQueue <- msg:
	default:
		// Drop logs if too fast
	}
}

func StartLogger() {
	go func() {
		for msg := range logQueue {
			fmt.Println("[LOG]", msg)
		}
	}()
}
