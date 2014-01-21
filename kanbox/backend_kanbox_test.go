package kanbox

import (
	"log"
	"os/user"
	"path/filepath"
	"testing"

	"../hashbin"
	"../register"
	"code.google.com/p/goauth2/oauth"
)

func TestKanboxBackend(t *testing.T) {
	user, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get current user: %v", err)
	}
	dataDir := filepath.Join(user.HomeDir, ".FileStore")
	reg, err := register.NewRegister(filepath.Join(dataDir, "register"))
	if err != nil {
		log.Fatalf("open register: %v", err)
	}
	var token *oauth.Token
	reg.Get("kanbox_token", &token)
	kanbox, err := New("hashstorage", token)
	if err != nil {
		t.Fatal(err)
	}
	bin := hashbin.New(kanbox)
	hashbin.RunTest(bin, t)
}
