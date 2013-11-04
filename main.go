package main

import (
	"./register"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

var DATADIR string
var REGISTER *register.Register

func init() {
	user, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get current user: %v", err)
	}
	DATADIR = filepath.Join(user.HomeDir, ".FileStore")
	_, err = os.Stat(DATADIR)
	if os.IsNotExist(err) {
		err = os.Mkdir(DATADIR, 0755)
		if err != nil {
			log.Fatalf("cannot create data dir: %v", err)
		}
	}

	REGISTER, err = register.NewRegister(filepath.Join(DATADIR, "register"))
	if err != nil {
		log.Fatalf("open register: %v", err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s [command]\n", os.Args[0])
		os.Exit(0)
	}
	switch os.Args[1] {
	case "snapshot":
		runSnapshot()
	case "upload":
		runUpload()
	case "download":
	case "setup":
		runSetup()
	case "update":
		runUpdate()
	default:
		log.Fatalf("unknown command %s", os.Args[1])
	}
}
