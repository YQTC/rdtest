package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/keroserene/go-webrtc"
	"io/ioutil"
	"net/http"
	"strings"
	"ubox.golib/p2p/protocol"
)

var (
	SdpManager = make(map[string]string)
)

type webRtc struct {
	chOnGenerateOffer chan int
	chSignalRegister  chan string
	chStartGetAppSdp  chan string
	pc                *webrtc.PeerConnection
	chStartSetBoxSdp  chan string
	chAllOk           chan int
	id                string
	dcManager         *dcManager

	proxyHost string
	sendSdp   func(sdp string)
	recvSdp   func(session string) (app_sdp string)
}

func NewWebRtc(host string, sendSdp func(sdp string), recvSdp func(session string) (app_sdp string)) *webRtc {
	ins := &webRtc{
		chOnGenerateOffer: make(chan int, 1),
		chSignalRegister:  make(chan string, 1),
		chStartGetAppSdp:  make(chan string, 1),
		chStartSetBoxSdp:  make(chan string, 1),
		chAllOk:           make(chan int, 1),
		proxyHost:         host,
		sendSdp:           sendSdp,
		recvSdp:           recvSdp,
	}
	return ins
}

func (wr *webRtc) StartUp() {

	// Step 1. create pc
	wr.createConn()

	// Step 2. register callback
	wr.registerCallback()

	// Step 3. createoffer
	go func() {
		<-wr.chOnGenerateOffer //wait

		wr.generateOffer()

		localSdp := wr.pc.LocalDescription().Serialize()
		session := getSdpSession(localSdp)
		SdpManager[session] = localSdp

		fmt.Printf("session :%s sdp :%s\n", session, localSdp)
	}()

	// Step 4. registerBoxSdp
	go func() {
		sdp := <-wr.chSignalRegister //wait
		wr.sendSdp(sdp)

		wr.chStartGetAppSdp <- sdp
	}()

	// Step 5. getRemoteAppSdp
	go func() {
		box_sdp := <-wr.chStartGetAppSdp //wait
		session := getSdpSession(box_sdp)

		wr.id = session

		app_sdp := wr.recvSdp(session)

		fmt.Printf("box get remote app sdp :%s\n", app_sdp)
		wr.chStartSetBoxSdp <- app_sdp
	}()

	// Step 6. setBoxLocalRemoteSdp
	go func() {
		app_sdp := <-wr.chStartSetBoxSdp //wait
		wr.setBoxLocalRemoteSdp(app_sdp)
		wr.chAllOk <- 1
	}()

	// Step 7. blocked & loop & print status info

	wr.prepareDataChannel()

	fmt.Printf("====Waiting all ok===\n")
	<-wr.chAllOk

	fmt.Printf("====main loop===\n")
	wr.mainLoop()

}

func (wr *webRtc) mainLoop() {
	for {

		req := <-wr.dcManager.ChReq
		fmt.Printf("mainLoop get req %+v\n", req)
		wr.dcManager.SendWebRtcReq(req)

		reader := bytes.NewReader([]byte(req.Body))
		url := wr.proxyHost + req.Url
		method := strings.ToUpper(req.Method)
		fmt.Printf("http url :%s method :%s\n", req.Url, method)
		request, err := http.NewRequest(method, url, reader)
		if err != nil {
			fmt.Printf("new http request err :%s\n", err.Error())
			continue
		}

		reqHeader := make(map[string][]string)
		json.Unmarshal([]byte(req.Header), &reqHeader)
		for k, v := range reqHeader {
			request.Header[k] = v
		}

		fmt.Printf("do http req , url :%s header :%+v body :%s\n", url, reqHeader, req.Body)
		client := http.Client{}
		response, err := client.Do(request)
		if err != nil {
			fmt.Printf("http req err :%s\n", err.Error())
			return
		}

		rsp := protocol.WebRtcRsp{}

		rspHeader := make(map[string][]string)

		rsp.Code = response.StatusCode
		for k, v := range response.Header {
			rspHeader[k] = v
		}
		rspHeaderStr, _ := json.Marshal(rspHeader)
		rsp.Header = string(rspHeaderStr)

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("read http rsp err :%s\n", err.Error())
			return
		}
		rsp.Body = string(body)

		wr.dcManager.SendWebRtcReq(rsp)

		fmt.Printf("session :%s mainLoop send rsp :%+v\n", wr.id, rsp)
	}
}

func (wr *webRtc) createConn() {
	fmt.Println("Starting up PeerConnection config...")
	urls := []string{"turn:iamtest.yqtc.co:3478?transport=udp"}
	s := webrtc.IceServer{Urls: urls, Username: "1531542280:guest", Credential: "xAhVJq3B18x2tdaFQUeYc3DcK9k="} //Credential:"turn.yqtc.top"
	webrtc.NewIceServer()
	config := webrtc.NewConfiguration()
	config.IceServers = append(config.IceServers, s)
	//config.IceTransportPolicy = webrtc.IceTransportPolicyRelay

	pc, err := webrtc.NewPeerConnection(config)

	wr.pc = pc
	if nil != err {
		fmt.Println("Failed to create PeerConnection.")
		return
	}
	return
}

func (wr *webRtc) registerCallback() {
	// OnNegotiationNeeded is triggered when something important has occurred in
	// the state of PeerConnection (such as creating a new data channel), in which
	// case a new SDP offer must be prepared and sent to the remote peer.
	wr.pc.OnNegotiationNeeded = func() {
		wr.chOnGenerateOffer <- 1
	}

	// Once all ICE candidates are prepared, they need to be sent to the remote
	// peer which will attempt reaching the local peer through NATs.
	wr.pc.OnIceComplete = func() {
		fmt.Println("Finished gathering ICE candidates.")
		sdp := wr.pc.LocalDescription().Serialize()
		wr.chSignalRegister <- sdp
	}

	wr.pc.OnDataChannel = func(channel *webrtc.DataChannel) {
		fmt.Println("Datachannel established by remote... ", channel.Label())
	}
}

func (wr *webRtc) prepareDataChannel() {
	// Attempting to create the first datachannel triggers ICE.
	fmt.Println("prepareDataChannel datachannel....")
	datachannl, err := wr.pc.CreateDataChannel("test")
	if nil != err {
		fmt.Println("Unexpected failure creating Channel.")
		return
	}

	wr.dcManager = NewDcManager(datachannl)
}

func (wr *webRtc) generateOffer() {
	fmt.Println("Generating offer...")
	offer, err := wr.pc.CreateOffer() // blocking
	if err != nil {
		fmt.Println(err)
		return
	}

	wr.pc.SetLocalDescription(offer)
}

func (wr *webRtc) setBoxLocalRemoteSdp(msg string) {
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(msg), &parsed)
	if nil != err {
		fmt.Println(err, ", try again.")
		fmt.Println("input msg=" + msg)
		return
	}

	if nil != parsed["sdp"] {
		sdp := webrtc.DeserializeSessionDescription(msg)
		if nil == sdp {
			fmt.Println("Invalid SDP.")
			return
		}

		err = wr.pc.SetRemoteDescription(sdp)
		if nil != err {
			fmt.Println("ERROR", err)
			return
		}
		fmt.Println("SDP " + sdp.Type + " successfully received.")
	}

	// Allow individual ICE candidate messages, but this won't be necessary if
	// the remote peer also doesn't use trickle ICE.
	if nil != parsed["candidate"] {
		ice := webrtc.DeserializeIceCandidate(msg)
		if nil == ice {
			fmt.Println("Invalid ICE candidate.")
			return
		}
		wr.pc.AddIceCandidate(*ice)
		fmt.Println("ICE candidate successfully received.")
	}
	fmt.Println("\nNormal exit setBoxLocalRemoteSdp")
}

func getSdpSession(sdp string) string {
	data := make(map[string]string)

	json.Unmarshal([]byte(sdp), &data)

	s := data["sdp"]

	sdps := strings.Split(s, " ")

	return sdps[1]
}
