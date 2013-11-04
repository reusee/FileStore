package main

import (
	"./baidu"
	"log"
)

func runSetup() {
	// baidu
	err := baidu.Setup(REGISTER)
	if err != nil {
		log.Fatalf("baidu setup: %v", err)
	}
}
