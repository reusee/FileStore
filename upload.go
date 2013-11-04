package main

import (
	"./baidu"
	"./hashbin"
	"code.google.com/p/goauth2/oauth"
	"log"
)

func runUpload() {
	baiduBackend, err := getBaiduBackend()
	if err != nil {
		log.Fatalf("get baidu client: %v", err)
	}
	_ = baiduBackend
}

func getBaiduBackend() (*hashbin.Bin, error) {
	var dir string
	var token oauth.Token
	err := REGISTER.Get("baidu_dir", &dir)
	if err != nil {
		return nil, err
	}
	err = REGISTER.Get("baidu_token", &token)
	if err != nil {
		return nil, err
	}
	b, err := baidu.New(dir, &token)
	if err != nil {
		return nil, err
	}
	return hashbin.New(b), nil
}
