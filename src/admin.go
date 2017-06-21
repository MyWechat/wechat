package wechat

import (
	"strings"
	log "github.com/sirupsen/logrus"
	"fmt"
)

type cmdExcutor func(wc *WeChat, msg Msg) error

var (
	cmdMap = map[string]cmdExcutor{
		"help": help,
		"start": start,
		"stop": stop,
	}
)

func replyCmdStatus(wc *WeChat, inMsg Msg, status string) {
	if inMsg.FromGroup {
		wc.sendMsg(inMsg.FromUserName, fmt.Sprintf("@%s %s", inMsg.FromGroupMember.NickName, status))
	} else {
		wc.sendMsg(inMsg.FromUserName, status)
	}
}


func help(wc *WeChat, msg Msg) error {
	replyCmdStatus(wc, msg, "start/stop/help")
	return nil
}


func start(wc *WeChat, msg Msg) error {
	wc.processMsg = true
	replyCmdStatus(wc, msg, "开始处理消息")
	log.Debug("开始处理消息")
	return nil
}

func stop(wc *WeChat, msg Msg) error {
	wc.processMsg = false
	replyCmdStatus(wc, msg, "停止处理消息")
	log.Debug("停止处理消息")
	return nil
}

func (wc *WeChat) excuteAdminCmd(msg Msg) error{
	cmd := msg.Content
	if msg.FromGroup {
		cmd = msg.MsgBodyWitoutAt
	}

	if fn, ok := cmdMap[strings.TrimSpace(cmd)]; ok {
		log.Debug("执行管理员命令")
		return fn(wc, msg)
	}

	return nil
}
