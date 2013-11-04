package baidu

import (
	"../register"
	"code.google.com/p/goauth2/oauth"
	"errors"
	"fmt"
	"log"
)

func Setup(register *register.Register) error {
	var key, secret, dir string
	fmt.Printf("enter app key:\n")
	n, err := fmt.Scanf("%s\n", &key)
	if n != 1 || err != nil {
		return errors.New(fmt.Sprintf("key error: %v", err))
	}
	fmt.Printf("enter app secret:\n")
	n, err = fmt.Scanf("%s\n", &secret)
	if n != 1 || err != nil {
		return errors.New(fmt.Sprintf("secret error: %v", err))
	}
	fmt.Printf("enter app dir:\n")
	n, err = fmt.Scanf("%s\n", &dir)
	if n != 1 || err != nil {
		return errors.New(fmt.Sprintf("dir error: %v", err))
	}

	config := &oauth.Config{
		ClientId:     key,
		ClientSecret: secret,
		RedirectURL:  "oob",
		Scope:        "netdisk",
		AuthURL:      "https://openapi.baidu.com/social/oauth/2.0/authorize?media_type=baidu&confirm_login=1&force_login=1",
		TokenURL:     "https://openapi.baidu.com/social/oauth/2.0/token",
	}
	transport := &oauth.Transport{Config: config}
	url := transport.Config.AuthCodeURL("")
	fmt.Printf("访问此网址并授权，然后输入跳转到的网址中的code参数: \n%s\n", url)
	var code string
	fmt.Scanf("%s", &code)
	token, err := transport.Exchange(code)
	if err != nil {
		log.Fatal(err)
	}
	err = register.Set("baidu_token", token)
	err = register.Set("baidu_dir", dir)
	return err
}
