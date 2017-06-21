package wechat

import (
	log "github.com/sirupsen/logrus"
	"errors"
	"encoding/json"
	"os/exec"
	"strings"
)

func (wc *WeChat) Proc(msg Msg) (err error) {
	log.Debug("接收到消息", msg)

	wc.parseMsg(&msg)

	user, err := wc.getContactFromName(msg.FromUserName)
	if err != nil {
		return
	}

	wc.checkAdminCmd(msg, user)

	if !wc.processMsg {
		log.Debug("不处理消息")
		return
	}

	log.Debug("准备遍历配置规则, 用户", user)
	for _, conf := range wc.config.User {
		log.Debug(conf)
		if wc.sameUser(user, conf.Username) {
			log.Debug("匹配到用户, 执行", conf.Action)
			go wc.excuteAction(msg, conf.Action)
			return
		}
	}

	go wc.excuteAction(msg, wc.config.Default)

	return
}

func (wc *WeChat) excuteAction(msg Msg, action string) (err error) {
	log.Debug("执行", action)

	type postMsg struct {
		From    string `json:"from"`
		Content string `json:"content"`
	}

	arg, err := json.Marshal(msg)
	if err != nil {
		log.Error(err)
		return
	}

	cmd := exec.Command(action, string(arg))
	out, err := cmd.Output()
	if err != nil {
		log.Error(err)
		return
	}

	log.Debug("脚本返回 ", string(out))

	if string(out) == "" {
		return
	}

	type retMsg struct {
		To      string `json:"to"`
		Content string `json:"content"`
	}

	var ret retMsg
	err = json.Unmarshal(out, &ret)
	if err != nil {
		log.Error(err)
		return
	}
	log.Debug("解析脚本返回 ", ret)
	wc.sendMsg(ret.To, ret.Content)

	return
}

func (wc *WeChat) sendMsg(to string, content string) (err error) {
	contact, err := wc.getContactFromName(to)
	if err != nil {
		return
	}

	log.Debug("发送消息: to ", contact, ", 内容: ", content)

	reqUrl := wc.baseUrl + "webwxsendmsg?pass_ticket=" + wc.passTicket

	msgId := wc.getTimeStamp()
	paramBody := map[string]interface{}{
		"BaseRequest": wc.authInfo,
		"Msg": map[string]interface{}{
			"Type":         1,
			"Content":      content,
			"FromUserName": wc.userName,
			"ToUserName":   contact.UserName,
			"ClientMsgId":  msgId,
			"LocalID":      msgId,
		},
	}

	_, err = wc.PostJson(reqUrl, paramBody)
	if err != nil {
		return
	}

	return
}

func (wc *WeChat) sameUser(user Contact, name string) bool {
	return user.NickName == name || user.UserName == name || (user.RemarkName != "" && user.RemarkName == name)
}

func (wc *WeChat) getContactFromName(name string) (c Contact, err error) {
	for _, iter := range wc.contact {
		if wc.sameUser(iter, name) {
			c = iter
			return
		}
	}

	err = errors.New("找不到联系人")
	return
}

func (wc *WeChat) parseMsg(msg *Msg) {
	if strings.HasPrefix(msg.FromUserName, "@@") {
		msg.FromGroup = true
		log.Debug("来自组的消息 ", msg)
	}

	if msg.FromGroup {
		const sep = ":<br/>"
		i := strings.Index(msg.Content, sep)
		if i == -1 {
			log.Error("找不到发消息的人 ")
			return
		}
		name := msg.Content[:i]
		contact, err := wc.getContactFromName(name)
		if err != nil {
			log.Error("拆出发消息的userid ", name, ", 但是在本地联系人名单中找不到")
			return
		}

		msg.FromGroupMember = contact
		msg.Content = msg.Content[i+len(sep):]
		msg.MsgBodyWitoutAt = msg.Content

		// 看是否有@
		parts := strings.Split(msg.Content, " ")
		var partsForJoin []string
		for _, part := range parts {
			if strings.HasPrefix(part, "@") {
				if wc.nickName == part[1:] {
					log.Debug("@me")
					msg.AtMe = true
				}
			} else {
				partsForJoin = append(partsForJoin, part)
			}
		}

		msg.MsgBodyWitoutAt = strings.Join(partsForJoin, "")
	}
}

func (wc *WeChat) checkAdminCmd(msg Msg, user Contact) {
	for _, adm := range wc.config.Base.Admin {
		// 组里的at
		if msg.FromGroup && msg.AtMe {
			if wc.sameUser(msg.FromGroupMember, adm) {
				wc.excuteAdminCmd(msg)
				return
			}
		} else if !msg.FromGroup {
			// 单聊
			if wc.sameUser(user, adm) {
				wc.excuteAdminCmd(msg)
				return
			}
		}
	}
}
