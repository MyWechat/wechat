## 微信机器人

* 只支持了文字消息收发
* 目前不能以 daemon 方式运行


* 只实现了微信收发的基本内核, 额外功能需扩展



### 使用方式

```shell
go get -u github.com/jialeicui/wechat
cd $GOPATH/src/github.com/jialeicui/wechat
go build
```

### 配置说明

将 `config.json.example` 拷贝为 `config.json`, 修改内容即可, 配置内容如下

```json
{
  "base": {
    "admin": [
      "张三",
      "李四"
    ],
    "loglevel": "debug",
    "apiaddr": "127.0.0.1:8765"
  },
  "user": [
    {
      "username": "崔一万",
      "action": "/path/to/foo.py"
    }
  ],
  "default": "/path/to/bar.py"
}

```

* admin, 管理员, 可以通过聊天方式下发命令, 目前只支持 help/start/stop, 可以单聊, 也可以在群聊里@
* loglevel, 日志等级, 可配置为 debug/info/warn/error/fatal/panic
* apiaddr, API 地址, 用于外部调用, 目前只支持 `发送消息` 和 `获取联系人`, 调用方式都为 GET
  * 发送消息  apiaddr/send?to=xxx&content=xxx
  * 获取联系人 apiaddr/contact
* user, 基于用户的脚本调用
* default, 如果没有匹配到用户脚本, 默认调用的脚本

### 脚本使用说明 (被动逻辑)

可参考 `demo/callback.py`

* 脚本接收一个序列化的 json 字符串参数, 解开之后可以获取到收到消息的详情
* 脚本使用标准输出返回序列化的 json 字符串, 用于给用户反馈, 如果不需要反馈, 不输出即可

### 扩展示例 (主动逻辑)

扩展基本上是围绕发送消息的 API 开发, 一般做成 cron, 见 demo 目录 (需要自行安装依赖, 为了不污染开发环境, 建议使用 virtualenv 建一个测试环境)

* smzdm 根据自定义规则推送商品信息

  使用时将 send_msg 调用地方填写为正确的昵称或者备注名称

* event-notification 根据用户日历发送提醒

  * 接收人的地方需要改
  * get_calendar_content 中将读取日历内容的方式修改为直接读 webcal (只测试了 macOS 下的日历, 测试方式为建立一个公开日历, 将日历地址中的 webcal 替换为 https 即可)