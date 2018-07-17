package main

import (
	"../proxy"
	"encoding/json"
	"flag"
	"log"
	"strings"
	"time"
	"ubox.golib/p2p/protocol"
)

const timeout = 60 * 5

var BOXID = "123"
var lastKeepalive int64

var WebRtcSdpChannelMap = make(map[string]chan string)

func main() {
	flag.StringVar(&BOXID, "boxid", BOXID, "boxid")
	flag.Parse()

	protocol.GetProtManagerIns().SetFuncHandler(protocol.ReqRegisterSdp{}, handleRegisterSdpReq)
	protocol.GetProtManagerIns().SetFuncHandler(protocol.PushAppSdp{}, handleRemoteAppSdp)
	protocol.GetProtManagerIns().SetFuncHandler(protocol.KeepAlive{}, handleKeepalive)

	checkTcpKeepalive()

	for {
		time.Sleep(time.Second)
	}
}

func checkTcpKeepalive() {
	go func() {
		for {
			log.Printf("checkTcpKeepalive last keepalive time :%d\n", lastKeepalive)
			p := protocol.KeepAlive{}
			protocol.GetTcpConn().HandleWrite(p)
			time.Sleep(time.Second * 60)
		}
	}()
	go func() {
		for {
			if !protocol.GetTcpConn().IsConnect() {
				lastKeepalive = time.Now().Unix()
				log.Printf("checkTcpKeepalive conn nil , create conn\n")
				go protocol.TcpConnect("iamtest.yqtc.co:7005")
			}

			if time.Now().Unix()-lastKeepalive > timeout && protocol.GetTcpConn().IsConnect() {
				protocol.GetTcpConn().Close()
				log.Printf("checkTcpKeepalive conn time out , close conn\n")
				go protocol.TcpConnect("iamtest.yqtc.co:7005")
			}

			time.Sleep(time.Second * 3)
		}
	}()

}

func startWebRtcServer() {

	go proxy.NewWebRtc("http://192.168.0.36:37867", sendSdp, recvSdp).StartUp()
}

func handleKeepalive(context protocol.Context) {
	log.Printf("handleKeepalive get data %+v\n", context.Data)
	req := protocol.KeepAlive{}
	json.Unmarshal(context.Data, &req)
	lastKeepalive = time.Now().Unix()

}

func handleRegisterSdpReq(context protocol.Context) {
	log.Printf("handleRegisterSdpReq get data %+v\n", context.Data)
	req := protocol.ReqRegisterSdp{}
	json.Unmarshal(context.Data, &req)

	startWebRtcServer()
}

func handleRemoteAppSdp(context protocol.Context) {
	log.Println(" ---- get sdp connect from host ---- ")
	log.Printf("handleRemoteAppSdp get data %s\n", context.Data)

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
	log.Printf("app box sdp match , app :%s box%s\n", req.AppSdp, boxSdp)
	if ok && strings.Index(boxSdp, session) >= 0 {

		_, ok := WebRtcSdpChannelMap[session]
		if !ok {
			WebRtcSdpChannelMap[session] = make(chan string, 1)
		}
		WebRtcSdpChannelMap[session] <- req.AppSdp

		log.Printf("app sdp match , start to set remote sdp...\n")
	} else {
		log.Printf("local sdp :%s remote sdp :%s\n", boxSdp, req.AppSdp)
		rsp.ErrNo = 1001
		rsp.ErrMsg = "app sdp not match box sdp..."
		log.Printf("app sdp not match box sdp...\n")
	}

	//send push sdp resp to server
	err := context.Conn.HandleWrite(rsp)
	if err != nil {
		log.Printf("handleRemoteAppSdp send rsp err :%s\n", err.Error())
	} else {
		log.Printf("handleRemoteAppSdp send rsp success :%+v\n", rsp)
	}
}

func sendSdp(sdp string) {
	log.Println(" ---- register sdp to host ---- ")

	req := protocol.RegisterSdp{}
	req.BoxId = BOXID
	req.Sdp = sdp

	err := protocol.GetTcpConn().HandleWrite(req)
	if err != nil {
		log.Printf("registerBoxSdp register failed , err :%s\n", err.Error())
	} else {
		log.Printf("registerBoxSdp register success ...\n")
	}
}

func recvSdp(session string) (app_sdp string) {

	_, ok := WebRtcSdpChannelMap[session]

	if !ok {
		WebRtcSdpChannelMap[session] = make(chan string, 1)
	}

	return <-WebRtcSdpChannelMap[session]
}
