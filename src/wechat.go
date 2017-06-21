package wechat

import (
	"fmt"
	"time"
	"strconv"
	"regexp"
	qrcode "github.com/skip2/go-qrcode"
	"bytes"
	"net/http"
	"encoding/xml"
	"net/url"
	"strings"
	"encoding/json"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"errors"
	"os/exec"
)

type Contact struct {
	UserName   string
	NickName   string
	RemarkName string

	Type uint32
}

type SyncInfo struct {
	msg Msg
	err error
}

type key struct {
	Key int
	Val int
}

type SyncKey struct {
	Count int
	List  []key
}

type user struct {
	UserName string
	NickName string
}

type InitInfo struct {
	SyncKey SyncKey
	User    user
}

type Msg struct {
	MsgId        string
	FromUserName string
	ToUserName   string
	MsgType      int
	Content      string

	AtMe            bool
	FromGroup       bool
	FromGroupMember Contact
	MsgBodyWitoutAt string
}

type MsgList struct {
	AddMsgList   []Msg
	SyncCheckKey SyncKey
}

type WeChat struct {
	client     *http.Client
	authInfo   map[string]string
	passTicket string
	baseUrl    string

	syncKey    SyncKey
	syncKeyStr string
	userName   string
	nickName   string

	contact  []Contact
	config   Config
	confPath string

	initDone bool

	// control
	processMsg bool
}

const (
	normal_contact   = iota
	group_contact
	official_contact
	public_contact
)

var officialContacts = []string{
	"weixin",
	"filehelper",
}

func (wc WeChat) getTimeStamp() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (wc *WeChat) GetUuid() string {
	param := map[string]string{
		"appid": "wx782c26e4c19acffb",
		"fun":   "new",
		"lang":  "zh_CN",
		"_":     wc.getTimeStamp(),
	}
	content, err := wc.Get("https://login.weixin.qq.com/jslogin", param)
	if err != nil {

	}

	re := regexp.MustCompile("window.QRLogin.code = 200; window.QRLogin.uuid = \"(.*)\";")
	match := re.FindStringSubmatch(content)

	return match[1]
}

func (wc *WeChat) GenQrCode(uuid string) (err error) {
	qr, err := qrcode.New("https://login.weixin.qq.com/l/"+uuid, qrcode.Highest)
	if err != nil {
		return
	}

	if wc.config.Base.QrCode == "img" {
		png, err := qr.PNG(256)
		tmpfile, err := ioutil.TempFile("", "weChatQrCode")
		tmpfile.Write(png)
		log.Debug(tmpfile.Name())
		err = exec.Command("open", tmpfile.Name()).Start()
		if err != nil {
			return err
		}
	} else {
		fmt.Println(wc.qr2String(qr, false))
	}

	return
}

func (wc *WeChat) qr2String(qr *qrcode.QRCode, inverseColor bool) string {
	bits := qr.Bitmap()
	var buf bytes.Buffer
	for y := range bits {
		for x := range bits[y] {
			if bits[y][x] != inverseColor {
				buf.WriteString("\033[49m  \033[0m")
			} else {
				buf.WriteString("\033[7m  \033[0m")
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func (wc *WeChat) WaitForLogin(uuid string) (redirect string, err error) {
	retry := 10
	tip := "1"
	for i := 0; i < retry; i++ {
		param := map[string]string{
			"tip":  tip,
			"uuid": uuid,
			"_":    wc.getTimeStamp(),
		}
		content, err := wc.Get("https://login.weixin.qq.com/cgi-bin/mmwebwx-bin/login", param)
		if err != nil {
			log.Error(err)
			return "", err
		}

		re := regexp.MustCompile("window.code=([0-9]+);")
		match := re.FindStringSubmatch(content)
		if len(match) < 2 {
			continue
		}

		code := match[1]

		if code == "201" {
			tip = "0"
			fmt.Println("扫描成功, 点击登录")
		} else if code == "200" {
			fmt.Println("登录成功")
			re := regexp.MustCompile("window.redirect_uri=\"(.*)\";")
			match := re.FindStringSubmatch(content)
			if len(match) < 2 {
				continue
			}
			redirect = match[1]
			return redirect, nil
		} else {
			log.Error("登录失败, 返回: ", content)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return "", errors.New("超时")
}

func (wc *WeChat) Init(redirect string) (err error) {
	param := map[string]string{
		"fun": "new",
	}

	body, err := wc.Get(redirect, param)
	if err != nil {
		return
	}

	type Result struct {
		XMLName xml.Name `xml:"error"`
		Skey    string   `xml:"skey"`
		Sid     string   `xml:"wxsid"`
		Uin     string   `xml:"wxuin"`
		Pass    string   `xml:"pass_ticket"`
	}
	res := Result{}
	err = xml.Unmarshal([]byte(body), &res)
	if err != nil {
		return
	}

	wc.authInfo = map[string]string{
		"Skey":     res.Skey,
		"Sid":      res.Sid,
		"Uin":      res.Uin,
		"DeviceID": "e1234567890",
	}

	wc.passTicket = res.Pass

	u, err := url.Parse(redirect)
	if err != nil {
		return
	}
	uri := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
	wc.baseUrl = uri[:strings.LastIndex(uri, "/")+1]

	p := url.Values{
		"lang":        {"en_US"},
		"pass_ticket": {wc.passTicket},
		"i":           {wc.getTimeStamp()},
	}

	reqBody, err := json.Marshal(map[string]map[string]string{"BaseRequest": wc.authInfo})

	if err != nil {
		return
	}

	reqUrl := wc.baseUrl + "webwxinit?" + p.Encode()
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(reqBody))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wc.client.Do(req)
	if err != nil {
		return
	}

	retBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var info InitInfo
	err = json.Unmarshal(retBody, &info)
	if err != nil {
		return
	}

	wc.userName = info.User.UserName
	wc.nickName = info.User.NickName
	wc.syncKey = info.SyncKey
	wc.syncKeyStr = wc.syncKeyToStr(wc.syncKey)

	return
}

func (wc WeChat) syncKeyToStr(syncKey SyncKey) (ret string) {
	var listString []string
	for _, key := range syncKey.List {
		listString = append(listString, fmt.Sprintf("%d_%d", key.Key, key.Val))
	}

	ret = strings.Join(listString, "|")
	return
}

func (wc *WeChat) Getcontact() (err error) {
	log.Info("开始获取联系人信息")
	reqUrl := wc.baseUrl + "webwxgetcontact?r=" + wc.getTimeStamp()
	body, err := wc.PostJson(reqUrl, nil)
	if err != nil {
		return
	}

	type ContantResp struct {
		MemberList []Contact
	}
	var resp ContantResp

	err = json.Unmarshal([]byte(body), &resp)
	if err != nil {
		log.Debug("原始返回 ", string(body))
		return
	}

	for _, c := range resp.MemberList {
		wc.addContact(c)
	}

	log.Info(wc.contact)

	if err = wc.GetGroupMemberInfo(wc.getGroupList()); err != nil {
		return
	}
	return
}

func (wc *WeChat) userNameExists(username string) (exists bool) {
	exists = false
	for _, c := range wc.contact {
		if username == c.UserName {
			exists = true
			return
		}
	}

	return
}

func (wc *WeChat) addContact(contact Contact) (err error) {
	if wc.userNameExists(contact.UserName) {
		return errors.New(fmt.Sprintf("联系人 %s 已经存在", contact.NickName))
	}
	if strings.HasPrefix(contact.UserName, "@@") {
		contact.Type = group_contact
	} else if (inArray(contact.UserName, officialContacts)) {
		contact.Type = official_contact
	} else {
		contact.Type = normal_contact
	}

	wc.contact = append(wc.contact, contact)
	return
}

func (wc *WeChat) getGroupList() (groups []string) {
	for _, c := range wc.contact {
		if c.Type == group_contact {
			groups = append(groups, c.UserName)
		}
	}
	return
}

func (wc *WeChat) Sync(syncInfo chan SyncInfo) {
	for {
		log.Info("开始同步状态")

		param := map[string]string{
			"r":        wc.getTimeStamp(),
			"sid":      wc.authInfo["Sid"],
			"uin":      wc.authInfo["Uin"],
			"skey":     wc.authInfo["Skey"],
			"deviceid": wc.authInfo["DeviceID"],
			"synckey":  wc.syncKeyStr,
			"_":        wc.getTimeStamp(),
		}

		body, err := wc.Get(wc.baseUrl+"synccheck", param)
		if err != nil {
			log.Info(err.Error())
			continue
		}

		re := regexp.MustCompile(`window.synccheck=\{retcode:"(\d+)",selector:"(\d+)"\}`)
		match := re.FindStringSubmatch(body)
		if len(match) != 3 {
			continue
		}

		retcode := match[1]
		//selector := match[2]

		if retcode != "0" {
			log.Error("同步失败: ", body)
			syncInfo <- SyncInfo{msg: Msg{}, err: errors.New("同步失败")}
			return
		}

		jsonParam := map[string]interface{}{
			"BaseRequest": wc.authInfo,
			"SyncKey":     wc.syncKey,
		}

		if err != nil {
			log.Error("Marshal fail")
			continue
		}

		p := url.Values{
			"sid": {wc.authInfo["Sid"]},
			"r":   {wc.getTimeStamp()},
		}

		ret, err := wc.PostJson(wc.baseUrl+"webwxsync?"+p.Encode(), jsonParam)
		var msgList MsgList
		err = json.Unmarshal([]byte(ret), &msgList)

		if err != nil {
			log.Error("解析json失败", ret)
			continue
		}

		log.Info("同步结果", msgList)

		wc.syncKey = msgList.SyncCheckKey
		wc.syncKeyStr = wc.syncKeyToStr(wc.syncKey)

		for _, msg := range msgList.AddMsgList {
			syncInfo <- SyncInfo{msg: msg, err: nil}
		}
	}
}

func (wc *WeChat) GetGroupMemberInfo(groups []string) (err error) {
	log.Info("获取组联系人 ", groups)
	list := make([]map[string]string, len(groups))
	for i, g := range groups {
		list[i] = map[string]string{"UserName": g}
	}

	param := RawBody{
		"BaseRequest": wc.authInfo,
		"Count":       len(groups),
		"List":        list,
	}

	q := url.Values{
		"type":        {"ex"},
		"r":           {wc.getTimeStamp()},
		"pass_ticket": {wc.passTicket},
	}

	body, err := wc.PostJson(wc.baseUrl+"webwxbatchgetcontact?"+q.Encode(), param)
	if err != nil {
		return
	}

	type contactList struct {
		MemberList []Contact
	}
	type memberContact struct {
		ContactList []contactList
	}

	var m memberContact
	err = json.Unmarshal([]byte(body), &m)
	if err != nil {
		log.Debug("原始返回 ", body)
		return
	}

	for _, g := range m.ContactList {
		for _, c := range g.MemberList {
			adderr := wc.addContact(c)
			if adderr != nil {
				// 只是为了debug, 走到这里是正常流程
				log.Debug(adderr)
			}
		}
	}

	return
}

func (wc *WeChat) Run() {
	go wc.HttpApiServe()

	for {
		uuid := wc.GetUuid()
		err := wc.GenQrCode(uuid)
		if err != nil {
			log.Error("生成二维码失败", err)
			continue
		}
		redirect, err := wc.WaitForLogin(uuid)
		if err != nil {
			fmt.Println("登录失败")
			continue
		}

		err = wc.Init(redirect)
		if err != nil {
			fmt.Println("初始化失败")
			continue
		}

		err = wc.Getcontact()
		if err != nil {
			log.Error(err)
			fmt.Println("获取联系人失败")
			continue
		}

		wc.initDone = true
		wc.processMsg = true

		syncInfo := make(chan SyncInfo)
		go wc.Sync(syncInfo)

		for info := range syncInfo {
			if info.err != nil {
				break
			}

			go wc.Proc(info.msg)
		}
	}
}

func New(file string) (wc *WeChat, err error) {
	wc = &WeChat{initDone: false}
	_, err = wc.MakeClient()
	if err != nil {
		return
	}

	err = wc.LoadConf(file)
	if err != nil {
		return
	}
	log.Info("加载配置成功\n", wc.config)

	return
}
