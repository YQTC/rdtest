package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/keroserene/go-webrtc"
	"log"
	"ubox.golib/p2p/protocol"
)

type dcManager struct {
	Buffer      *bytes.Buffer
	ChReq       chan protocol.WebRtcReq
	dataChannel *webrtc.DataChannel
}

func NewDcManager(dc *webrtc.DataChannel) (pdc *dcManager) {

	instance := &dcManager{dataChannel: dc}

	instance.Buffer = new(bytes.Buffer)
	//instance.ChRsp = make(chan protocol.WebRtcRsp , 10)
	instance.ChReq = make(chan protocol.WebRtcReq, 10)

	dc.OnOpen = func() {
		log.Println("Data Channel Opened!")
		//startChat()
	}
	dc.OnClose = func() {
		log.Println("Data Channel closed.")
		close(instance.ChReq)
	}
	dc.OnMessage = func(msg []byte) {
		log.Printf("recv msg : %s\n", msg)
		instance.recvData(msg)
	}

	return instance
}

func (pdc *dcManager) SendWebRtcReq(i interface{}) {
	data, _ := json.Marshal(i)

	log.Printf("data channel status :%s\n", pdc.dataChannel.ReadyState().String())
	pdc.dataChannel.Send([]byte(string(data) + "\n"))
}

func (pdc *dcManager) recvData(buf []byte) {
	n, err := pdc.Buffer.Write(buf)

	if err != nil {
		log.Printf("write buffer error %s\n", err.Error())
	}
	log.Printf("write buffer data success n:%d data:%s\n", n, buf[:n])

	data, err := pdc.Buffer.ReadString('\n')
	if err != nil {
		pdc.Buffer.Write([]byte(data))
		log.Printf("read buffer err :%s\n", err.Error())
		return
	}

	bdata, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Printf("decode string err :%s\n", err.Error())
	}
	log.Printf("read buffer data success :%s\n", bdata)

	p := protocol.WebRtcReq{}
	err = json.Unmarshal(bdata, &p)

	if err != nil {
		log.Printf("unmarshal req err :%s\n", err.Error())
		return
	}

	pdc.ChReq <- p

}
