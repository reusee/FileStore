package baidu

import (
	"../hashbin"
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/jsonq"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	neturl "net/url"
	"os"
	"time"
)

type Baidu struct {
	dir              string
	client           *http.Client
	token            *oauth.Token
	keys             map[string]bool
	keyCacheFilePath string
	newKey           chan string
}

func New(dir string, token *oauth.Token, keyCacheFilePath string) (*Baidu, error) {
	baidu := &Baidu{
		dir:              dir,
		token:            token,
		client:           new(http.Client),
		keys:             make(map[string]bool),
		keyCacheFilePath: keyCacheFilePath,
		newKey:           make(chan string),
	}
	quota, used, err := baidu.GetQuota()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Baidu: quota %s, used %s\n", formatSize(quota), formatSize(used))

	if keyCacheFilePath != "" {
		f, err := os.Open(keyCacheFilePath)
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.New(fmt.Sprintf("cannot open key cache file: %v", err))
		}
		if err == nil {
			err = gob.NewDecoder(f).Decode(&baidu.keys)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("cannot decode key cache file: %v", err))
			}
		}
	}

	go baidu.start()

	return baidu, nil
}

func (self *Baidu) start() {
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case key := <-self.newKey: // new key
			self.keys[key] = true
		case <-ticker.C: // save to file
			if self.keyCacheFilePath == "" {
				continue
			}
			f, err := os.Create(self.keyCacheFilePath + ".new")
			if err != nil {
				log.Fatalf("cannot open key cache file: %v", err)
			}
			err = gob.NewEncoder(f).Encode(self.keys)
			if err != nil {
				log.Fatalf("cannot write key cache file: %v", err)
			}
			err = os.Rename(self.keyCacheFilePath+".new", self.keyCacheFilePath)
			if err != nil {
				log.Fatalf("cannot write key cache file: %v", err)
			}
			fmt.Printf("%d keys saved to cache file\n", len(self.keys))
		}
	}
}

func NewBaiduWithStringToken(dir, tokenStr string, keyCacheFilePath string) (*Baidu, error) {
	var token oauth.Token
	tokenBytes, err := hex.DecodeString(tokenStr)
	if err != nil {
		return nil, err
	}
	err = gob.NewDecoder(bytes.NewReader(tokenBytes)).Decode(&token)
	if err != nil {
		return nil, err
	}
	return New(dir, &token, keyCacheFilePath)
}

func (self *Baidu) GetQuota() (quota int, used int, err error) {
	q, err := self.get("quota", "info", nil)
	if err != nil {
		return 0, 0, err
	}
	quota, err = q.Int("quota")
	if err != nil {
		return 0, 0, err
	}
	used, err = q.Int("used")
	if err != nil {
		return 0, 0, err
	}
	return
}

func (self *Baidu) get(api, method string, params map[string]string) (*jsonq.JsonQuery, error) {
	url := fmt.Sprintf("https://pcs.baidu.com/rest/2.0/pcs/%s?method=%s&access_token=%s", api, method, self.token.AccessToken)
	for key, value := range params {
		url += fmt.Sprintf("&%s=%s", key, value)
	}
	resp, err := self.client.Get(url)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("%s %s %v", method, api, err))
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, errors.New("response body read error")
	}
	data := make(map[string]interface{})
	json.NewDecoder(buf).Decode(&data)
	return jsonq.NewQuery(data), nil
}

func (self *Baidu) upload(path string, data []byte, length int) error {
	url := fmt.Sprintf("https://c.pcs.baidu.com/rest/2.0/pcs/file?method=upload&access_token=%s&ondup=overwrite", self.token.AccessToken)
	url += "&path=" + neturl.QueryEscape(fmt.Sprintf("/apps/%s/%s", self.dir, path))

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
	respBody := make(map[string]interface{})
	err = json.NewDecoder(buf).Decode(&respBody)
	if err != nil {
		return errors.New("return json decode error")
	}
	q := jsonq.NewQuery(respBody)
	if resp.StatusCode != http.StatusOK { // error
		errCode, _ := q.Int("error_code")
		errMsg, _ := q.String("error_msg")
		return errors.New(fmt.Sprintf("server error %d %s", errCode, errMsg))
	} else { // ok
		retSize, _ := q.Int("size")
		if retSize != length {
			return errors.New("upload size wrong")
		}
		hasher := md5.New()
		hasher.Write(data)
		md5, _ := q.String("md5")
		if fmt.Sprintf("%x", hasher.Sum(nil)) != md5 {
			return errors.New("md5 not match")
		}
	}
	return nil
}

func (self *Baidu) NewWriter(length int, hash string) (io.Writer, hashbin.Callback, error) {
	buf := new(bytes.Buffer)
	return buf, func(err error) error {
		if err != nil {
			return err
		}
		err = self.upload(fmt.Sprintf("%s/%d-%s", hash[:2], length, hash), buf.Bytes(), length)
		if err != nil {
			return err
		}
		self.newKey <- fmt.Sprintf("%d-%s", length, hash)
		return nil
	}, nil
}

func (self *Baidu) NewReader(length int, hash string) (io.Reader, hashbin.Callback, error) {
	path := neturl.QueryEscape(fmt.Sprintf("/apps/%s/%s/%d-%s", self.dir, hash[:2], length, hash))
	url := fmt.Sprintf("https://d.pcs.baidu.com/rest/2.0/pcs/file?method=download&access_token=%s&path=%s", self.token.AccessToken, path)
	resp, err := self.client.Get(url)
	if err != nil {
		return nil, nil, errors.New(fmt.Sprintf("get error: %s", url))
	}
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, resp.Body)
		if err != nil {
			return nil, nil, errors.New("response body read error")
		}
		respBody := make(map[string]interface{})
		err = json.NewDecoder(buf).Decode(&respBody)
		if err != nil {
			return nil, nil, errors.New("return json decode error")
		}
		q := jsonq.NewQuery(respBody)
		errCode, _ := q.Int("error_code")
		errMsg, _ := q.String("error_msg")
		return nil, nil, errors.New(fmt.Sprintf("fetch error %d %s", errCode, errMsg))
	}
	return resp.Body, func(err error) error {
		defer resp.Body.Close()
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func (self *Baidu) Exists(length int, hash string) (bool, error) {
	key := fmt.Sprintf("%d-%s", length, hash)
	if _, ok := self.keys[key]; ok {
		return true, nil
	}
	//TODO query server
	return false, nil
}

func (self *Baidu) Mkdir(path string) error {
	url := fmt.Sprintf("https://pcs.baidu.com/rest/2.0/pcs/file?method=mkdir&access_token=%s", self.token.AccessToken)
	url += "&path=" + neturl.QueryEscape(fmt.Sprintf("/apps/%s/%s", self.dir, path))

	buf := new(bytes.Buffer)
	form := multipart.NewWriter(buf)
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
	respBody := make(map[string]interface{})
	err = json.NewDecoder(buf).Decode(&respBody)
	if err != nil {
		return errors.New("return json decode error")
	}
	q := jsonq.NewQuery(respBody)
	if resp.StatusCode != http.StatusOK {
		errCode, _ := q.Int("error_code")
		errMsg, _ := q.String("error_msg")
		return errors.New(fmt.Sprintf("server error %d %s", errCode, errMsg))
	}
	return nil
}
