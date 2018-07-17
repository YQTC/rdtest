package main

import (
	"../proxy"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"
	"ubox.golib/p2p/protocol"
)

var BOXID = "123"

var WebRtcSdpChannelMap = make(map[string]chan string)

func main() {
	flag.StringVar(&BOXID, "boxid", BOXID, "boxid")
	flag.Parse()

	protocol.GetProtManagerIns().SetFuncHandler(protocol.ReqRegisterSdp{}, handleRegisterSdpReq)
	protocol.GetProtManagerIns().SetFuncHandler(protocol.PushAppSdp{}, handleRemoteAppSdp)

	go protocol.TcpConnect("iamtest.yqtc.co:7005")

	startWebRtcServer()

	for {
		time.Sleep(time.Second)
	}
}

func handleRegisterSdpReq(context protocol.Context) {
	fmt.Printf("handleRegisterSdpReq get data %+v\n", context.Data)
	req := protocol.ReqRegisterSdp{}
	json.Unmarshal(context.Data, &req)

	startWebRtcServer()
}

func startWebRtcServer() {
	go proxy.NewWebRtc("http://192.168.0.36:37867", sendSdp, recvSdp).StartUp()
}

func sendSdp(sdp string) {
	fmt.Println(" ---- register sdp to host ---- ")

	req := protocol.RegisterSdp{}
	req.BoxId = BOXID
	req.Sdp = sdp

	err := protocol.GetTcpConn().HandleWrite(req)
	if err != nil {
		fmt.Printf("registerBoxSdp register failed , err :%s\n", err.Error())
	} else {
		fmt.Printf("registerBoxSdp register success ...\n")
	}
}

func recvSdp(session string) (app_sdp string) {

	_, ok := WebRtcSdpChannelMap[session]

	if !ok {
		WebRtcSdpChannelMap[session] = make(chan string, 1)
	}

	return <-WebRtcSdpChannelMap[session]
}

func handleRemoteAppSdp(context protocol.Context) {
	fmt.Println(" ---- get sdp connect from host ---- ")
	fmt.Printf("handleRemoteAppSdp get data %s\n", context.Data)

	//handle app sdp push req
	req := protocol.PushAppSdp{}
	json.Unmarshal(context.Data, &req)

	//get session from sdp
	sdpPack := map[string]string{}

	json.Unmarshal([]byte(req.AppSdp), &sdpPack)

	session, ok := sdpPack["myrandsessionid"]
	boxSdp := proxy.SdpManager[session]

	//compare session whether macth
	rsp := protocol.PushRes{
		ErrNo:  0,
		ErrMsg: "success",
	}
	rsp.RequestId = req.RequestId
	fmt.Printf("app box sdp match , app :%s box%s\n", req.AppSdp, boxSdp)
	if ok && strings.Index(boxSdp, session) >= 0 {

		_, ok := WebRtcSdpChannelMap[session]
		if !ok {
			WebRtcSdpChannelMap[session] = make(chan string, 1)
		}
		WebRtcSdpChannelMap[session] <- req.AppSdp

		fmt.Printf("app sdp match , start to set remote sdp...\n")
	} else {
		fmt.Printf("local sdp :%s remote sdp :%s\n", boxSdp, req.AppSdp)
		rsp.ErrNo = 1001
		rsp.ErrMsg = "app sdp not match box sdp..."
		fmt.Printf("app sdp not match box sdp...\n")
	}

	//send push sdp resp to server
	err := context.Conn.HandleWrite(rsp)
	if err != nil {
		fmt.Printf("handleRemoteAppSdp send rsp err :%s\n", err.Error())
	} else {
		fmt.Printf("handleRemoteAppSdp send rsp success :%+v\n", rsp)
	}
}
