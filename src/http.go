package wechat

import (
	"fmt"
	"net/http"
	"encoding/json"
)

type Ret map[string]interface{}

func responseJson(w http.ResponseWriter, ret Ret) {
	body, _ := json.Marshal(ret)
	fmt.Fprintln(w, string(body))
}

func success(w http.ResponseWriter, data interface{}) {
	responseJson(w, Ret{"errno": 0, "data": data})
}

func errorjson(w http.ResponseWriter, msg string) {
	responseJson(w, Ret{"errno": 1, "msg": msg})
}

func (wc *WeChat)sendMsgHandler(w http.ResponseWriter, r *http.Request) {
	to := r.URL.Query().Get("to")
	content := r.URL.Query().Get("content")

	if to == "" || content == "" {
		errorjson(w, "需要 to 和 content 参数")
		return
	}

	if !wc.initDone {
		errorjson(w, "还没有准备好")
		return
	}
	err := wc.sendMsg(to, content)
	if err != nil {
		errorjson(w, err.Error())
		return
	}
	success(w, to + ": " + content)
}

func (wc *WeChat) contactHandler (w http.ResponseWriter, r *http.Request) {
	err := wc.Getcontact()
	if err != nil {
		errorjson(w, "获取联系人失败")
		return
	}

	success(w, wc.contact)
}

func (wc *WeChat) HttpApiServe() (err error){
	http.HandleFunc("/send", wc.sendMsgHandler)
	http.HandleFunc("/contact", wc.contactHandler)
	err = http.ListenAndServe(wc.config.Base.ApiAddr, nil)
	if err != nil {
		return
	}

	return
}
