package main

import (
	"time"
	"ubox.golib/p2p/protocol"
	"../proxy"
	"flag"
)

var BOXID = "123"


func main(){
	flag.StringVar(&BOXID, "boxid", BOXID, "boxid")
	flag.Parse()

	go protocol.TcpConnect("iamtest.yqtc.co:7005")

	go proxy.NewWebRtc().StartUp()

	for {
		time.Sleep(time.Second)
	}
}


