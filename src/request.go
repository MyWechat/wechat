package wechat

import (
	"net/http"
	"errors"
	"io/ioutil"
	"net/http/cookiejar"
	"net/url"
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

type RawBody map[string]interface{}

func buildReq(url string, param map[string]string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.New("build request")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux i686; U;) Gecko/20070322 Kazehakase/0.4.5")

	if param != nil {
		q := req.URL.Query()
		for k, v := range param {
			q.Add(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	return req, nil
}

func (wc *WeChat) MakeClient() (c *http.Client, err error) {
	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return
	}

	c = &http.Client{
		Jar: cookieJar,
	}

	wc.client = c

	return
}

func (wc *WeChat) Get(url string, param map[string]string) (ret string, err error) {
	req, err := buildReq(url, param)
	if err != nil {
		return
	}

	c := wc.client
	resp, err := c.Do(req)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	ret = string(body)

	return
}

func (wc *WeChat) Post(url string, param url.Values) (ret string, err error) {
	resp, err := http.PostForm(url, param)

	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	ret = string(body)
	return
}

func (wc *WeChat) PostJson(url string, param RawBody) (ret string, err error) {
	c := wc.client

	body := []byte("{}")
	if param != nil {
		jsonData, err := json.Marshal(param)
		if err != nil {
			return "", err
		}

		body = bytes.NewBuffer(jsonData).Bytes()
	}

	// 重试
	// TODO 将所有网络请求添加重试
	var resp *http.Response
	for i := 0; i < 3; i++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		log.Debug("尝试第 ", i+1, " 次...")
		resp, err = c.Do(req)
		// 成功, 停止尝试
		if err == nil {
			break
		}
	}

	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(resp.Body)

	ret = string(body)
	return
}
