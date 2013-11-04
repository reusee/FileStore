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
	"mime/multipart"
	"net/http"
	neturl "net/url"
)

type Baidu struct {
	dir    string
	client *http.Client
	token  *oauth.Token
}

func New(dir string, token *oauth.Token) (*Baidu, error) {
	baidu := &Baidu{
		dir:    dir,
		token:  token,
		client: new(http.Client),
	}
	quota, used, err := baidu.GetQuota()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Baidu: quota %s, used %s\n", formatSize(quota), formatSize(used))
	return baidu, nil
}

func NewBaiduWithStringToken(dir, tokenStr string) (*Baidu, error) {
	var token oauth.Token
	tokenBytes, err := hex.DecodeString(tokenStr)
	if err != nil {
		return nil, err
	}
	err = gob.NewDecoder(bytes.NewReader(tokenBytes)).Decode(&token)
	if err != nil {
		return nil, err
	}
	return New(dir, &token)
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

func (self *Baidu) NewWriter(length int, hash []byte) (io.Writer, hashbin.Callback, error) {
	buf := new(bytes.Buffer)
	return buf, func(err error) error {
		if err != nil {
			return err
		}
		return self.upload(fmt.Sprintf("%d-%x", length, hash), buf.Bytes(), length)
	}, nil
}

func (self *Baidu) NewReader(length int, hash []byte) (io.Reader, hashbin.Callback, error) {
	path := neturl.QueryEscape(fmt.Sprintf("/apps/%s/%d-%x", self.dir, length, hash))
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

func (self *Baidu) Exists(length int, hash []byte) (bool, error) {
	return false, nil
}
