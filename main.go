package main

import (
	"./register"
	"./snapshot"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
)

type App struct {
	dataDir          string
	register         *register.Register
	flags            []string
	args             []string
	path             string
	escapedPath      string
	snapshotSet      *snapshot.SnapshotSet
	snapshotFilePath string
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

	var path string
	for i := 2; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg[0] == '-' {
			app.flags = append(app.flags, arg)
		} else {
			if path == "" {
				path = arg
			} else {
				app.args = append(app.args, arg)
			}
		}
	}
	if path == "" {
		fmt.Printf("path is empty\n")
		os.Exit(0)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		log.Fatalf("invalid path: %v", err)
	}
	app.path = path

	snapshotSet, err := snapshot.New(path)
	if err != nil {
		log.Fatalf("cannot create snapshot set: %v", err)
	}
	escapedPath := url.QueryEscape(path)
	app.escapedPath = escapedPath
	snapshotFilePath := filepath.Join(app.dataDir, escapedPath+".snapshots")
	app.snapshotFilePath = snapshotFilePath
	err = snapshotSet.Load(snapshotFilePath)
	if err != nil {
		log.Fatalf("cannot read snapshots from file: %v", err)
	}
	fmt.Printf("loaded %d snapshots from file\n", len(snapshotSet.Snapshots))
	app.snapshotSet = snapshotSet

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
