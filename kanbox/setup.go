package kanbox

import (
	"errors"
	"fmt"
	"log"

	"../register"
	"code.google.com/p/goauth2/oauth"
)

func Setup(dir string, register *register.Register) error {
	var key, secret string
	fmt.Printf("enter key:\n")
	n, err := fmt.Scanf("%s\n", &key)
	if n != 1 || err != nil {
		return errors.New(fmt.Sprintf("key error: %v", err))
	}
	fmt.Printf("enter secret:\n")
	n, err = fmt.Scanf("%s\n", &secret)
	if n != 1 || err != nil {
		return errors.New(fmt.Sprintf("secret error: %v", err))
	}

	config := &oauth.Config{
		ClientId:     key,
		ClientSecret: secret,
		//RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		RedirectURL: "oob",
		AuthURL:     "https://auth.kanbox.com/0/auth?response_type=code&user_language=ZH&user_platform=linux",
		TokenURL:    "https://auth.kanbox.com/0/token",
	}
	transport := &oauth.Transport{Config: config}

	url := transport.Config.AuthCodeURL("")
	fmt.Printf("auth: %s\n", url)

	var code string
	fmt.Scanf("%s", &code)
	token, err := transport.Exchange(code)
	if err != nil {
		log.Fatal(err)
	}
	err = register.Set("kanbox_token", token)
	if err != nil {
		return err
	}

	c, err := New(dir, token)
	if err != nil {
		return err
	}
	chars := "0123456789abcdef"
	for _, a := range chars {
		for _, b := range chars {
			path := fmt.Sprintf("%c%c", a, b)
			c.Mkdir(path)
			fmt.Printf("created %s\n", path)
		}
	}

	return nil
}
