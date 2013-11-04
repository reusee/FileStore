package main

import (
	"./baidu"
	"log"
)

func (self *App) runSetup() {
	// baidu
	err := baidu.Setup(self.register)
	if err != nil {
		log.Fatalf("baidu setup: %v", err)
	}
}
