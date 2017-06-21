package wechat

import (
	"io/ioutil"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

type BaseConf struct {
	Admin    []string `json:"admin"`
	LogLevel string `json:"loglevel"`
	ApiAddr  string `json:"apiaddr"`
	QrCode   string `json:"qrcode"`
}

type UserConf struct {
	Username string `json:"username"`
	Action   string `json:"action"`
}

type Config struct {
	Base    BaseConf `json:"base"`
	User    []UserConf `json:"user"`
	Default string `json:"default"`
}

func (wc *WeChat) LoadConf(file string) (err error) {
	body, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}

	var conf Config
	err = json.Unmarshal(body, &conf)
	if err != nil {
		return
	}

	wc.confPath = file
	wc.config = conf

	lv, err := log.ParseLevel(conf.Base.LogLevel)
	if err != nil {
		return
	}
	log.SetLevel(lv)
	return
}

func (wc *WeChat) SaveConf() (err error) {
	body, err := json.MarshalIndent(wc.config, "", "  ")
	if err != nil {
		return
	}

	err = ioutil.WriteFile(wc.confPath, body, 0644)

	return
}
