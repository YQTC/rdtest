package main

import (
	"time"
	"ubox.golib/p2p/protocol"
	"../proxy"
)


func main(){

	go protocol.TcpConnect("iamtest.yqtc.co:7005")

	go proxy.NewWebRtc().StartUp()

	for {
		time.Sleep(time.Second)
	}
}


