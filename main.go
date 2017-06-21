package main

import (
	"github.com/jialeicui/wechat/src"
	"fmt"
)

func main() {
	wc, err := wechat.New("config.json")
	if err != nil {
		fmt.Printf("初始化失败 %s", err)
		return
	}

	wc.Run()
}
