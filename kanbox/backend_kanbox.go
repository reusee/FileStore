package kanbox

import (
	"../hashbin"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	neturl "net/url"
	"time"

	"code.google.com/p/goauth2/oauth"
)

type KanBox struct {
	keys   map[string]bool
	token  *oauth.Token
	client *http.Client
	dir    string
}

func New(dir string, token *oauth.Token) (*KanBox, error) {
	kanbox := &KanBox{
		dir:   dir,
		token: token,
		client: &http.Client{
			Transport: &http.Transport{
				Dial: func(network, addr string) (net.Conn, error) {
					return net.DialTimeout(network, addr, time.Second*30)
				},
				ResponseHeaderTimeout: time.Minute * 2,
			},
		},
	}
	resp, err := kanbox.client.Get(fmt.Sprintf("https://api.kanbox.com/0/info?bearer_token=%s", kanbox.token.AccessToken))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("server %d %s", resp.StatusCode, buf.Bytes()))
	}
	fmt.Printf("%s\n", buf.Bytes())
	return kanbox, nil
}

func (self *KanBox) Exists(length int, hash string) (bool, error) {
	key := fmt.Sprintf("%d-%s", length, hash)
	if _, ok := self.keys[key]; ok {
		return true, nil
	}
	//TODO query server
	return false, nil
}

func (self *KanBox) NewReader(length int, hash string) (io.Reader, hashbin.Callback, error) {
	path := neturl.QueryEscape(fmt.Sprintf("/%s/%s/%d-%s", self.dir, hash[:2], length, hash))
	url := fmt.Sprintf("https://api.kanbox.com/0/download?bearer_token=%s&path=%s", self.token.AccessToken, path)
	resp, err := self.client.Get(url)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("get error, %s", url))
	}
	if resp.StatusCode != http.StatusOK { //TODO
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, resp.Body)
		if err != nil {
			return nil, nil, errors.New("response body read error")
		}
		return nil, nil, errors.New(fmt.Sprintf("fetch error %s", buf.Bytes()))
	}
	return resp.Body, func(err error) error {
		defer resp.Body.Close()
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func (self *KanBox) NewWriter(length int, hash string) (io.Writer, hashbin.Callback, error) {
	buf := new(bytes.Buffer)
	return buf, func(err error) error {
		if err != nil {
			return err
		}
		err = self.upload(fmt.Sprintf("%s/%d-%s", hash[:2], length, hash), buf.Bytes(), length)
		if err != nil {
			return err
		}
		//self.newKey <- fmt.Sprintf("%d-%s", length, hash) //TODO
		return nil
	}, nil
}

func (self *KanBox) upload(path string, data []byte, length int) error {
	url := fmt.Sprintf("https://api-upload.kanbox.com/0/upload?bearer_token=%s", self.token.AccessToken)
	url += "&path=" + neturl.QueryEscape(fmt.Sprintf("/%s/%s", self.dir, path))
	fmt.Printf("%s\n", url)

	buf := new(bytes.Buffer)
	form := multipart.NewWriter(buf)
	field, _ := form.CreateFormFile("file", "file")
	field.Write(data)
	form.Close()
	resp, err := self.client.Post(url, form.FormDataContentType(), buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf.Reset()
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return errors.New("response body read error")
	}
	if resp.StatusCode != http.StatusOK { // error
		return errors.New(fmt.Sprintf("server error %d %s", resp.StatusCode, buf.Bytes()))
	}
	return nil
}

func (self *KanBox) Mkdir(path string) error {
	url := fmt.Sprintf("https://api.kanbox.com/0/create_folder?bearer_token=%s", self.token.AccessToken)
	url += "&path=" + neturl.QueryEscape(fmt.Sprintf("/%s/%s", self.dir, path))

	resp, err := self.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return errors.New("response body read error")
	}
	fmt.Printf("%s\n", buf.Bytes())

	return nil
}
