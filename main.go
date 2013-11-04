package main

import (
	"./register"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/user"
	"path/filepath"
)

type App struct {
	dataDir  string
	register *register.Register
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s [command]\n", os.Args[0])
		os.Exit(0)
	}

	go http.ListenAndServe("0.0.0.0:55555", nil)

	app := new(App)

	user, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get current user: %v", err)
	}
	dataDir := filepath.Join(user.HomeDir, ".FileStore")
	_, err = os.Stat(dataDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(dataDir, 0755)
		if err != nil {
			log.Fatalf("cannot create data dir: %v", err)
		}
	}
	app.dataDir = dataDir

	reg, err := register.NewRegister(filepath.Join(dataDir, "register"))
	if err != nil {
		log.Fatalf("open register: %v", err)
	}
	app.register = reg

	switch os.Args[1] {
	case "snapshot":
		app.runSnapshot()
	case "upload":
		app.runUpload()
	case "setup":
		app.runSetup()
	case "update":
		app.runUpdate()
	case "list":
		app.runList()
	default:
		log.Fatalf("unknown command %s", os.Args[1])
	}
}
