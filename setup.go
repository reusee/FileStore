package main

import (
	//"./baidu"
	"./kanbox"
	"log"
)

func (self *App) runSetup() {
	//TODO
	// baidu
	//err := baidu.Setup(self.register)
	//if err != nil {
	//	log.Fatalf("baidu setup: %v", err)
	//}

	err := kanbox.Setup("hashstorage", self.register)
	if err != nil {
		log.Fatalf("kanbox setup: %v", err)
	}
}
