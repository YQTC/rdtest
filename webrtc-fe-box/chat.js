var $chatlog, $input, $send, $name;

// WebRTC objects
var config = {
  iceServers: [
    { 
      urls:           ["stun:139.199.180.239:7002", "turn:139.199.180.239:7002"], 
      username:       'user',
      credential:     'myPassword',
      credentialType: 'password'
    }
    // { urls: ["stun:stun.l.google.com:19302"] }
  ]
}
var cast = [
  "Alice", "Bob", "Carol", "Dave", "Eve",
  "Faythe", "Mallory", "Oscar", "Peggy",
  "Sybil", "Trent", "Wendy"
]

window.PeerConnection = window.RTCPeerConnection ||
                        window.mozRTCPeerConnection || window.webkitRTCPeerConnection;
window.RTCIceCandidate = window.RTCIceCandidate || window.mozRTCIceCandidate;
window.RTCSessionDescription = window.RTCSessionDescription || window.mozRTCSessionDescription;

var pc;  // PeerConnection
var offer, answer;
// Let's randomize initial username from the cast of characters, why not.
var username = cast[Math.floor(cast.length * Math.random())];
var channel;

var myrandsessionid = Math.floor(1000000000 * Math.random())

// Janky state machine
var MODE = {
  INIT:       0,
  CONNECTING: 1,
  CHAT:       2
}
var currentMode = MODE.INIT;

// Signalling channel - just tells user to copy paste to the peer.
var Signalling = {
  send: function(msg) {
    log("---- Please copy the below to peer ----\n");
    log(JSON.stringify(msg));
    log("\n");
    // sendLocalSDP(JSON.stringify(sdp));
    updateBoxInfo(msg)
      .then(()=>{
        log("updateBoxInfo success, should start to getRemoteAppSDP \n")
        return getRemoteAppSDP()
      })
      .then((res)=>{
         log("getRemoteAppSDP success, should start to set(Romete)Description \n")
         log("app_sdp:   " + res.app_sdp);
         Signalling.receive(res.app_sdp)
      }); 
  },
  receive: function(msg) {
    var recv;
    try {
      recv = JSON.parse(msg);
    } catch(e) {
      log("Invalid JSON.");
      return;
    }
    if (!pc) {
      throw new Error("pc not null!!!");
      start(false);
    }
    var desc = recv['sdp']
    var ice = recv['candidate']
    if (!desc && ! ice) {
      log("Invalid SDP.");
      return false;
    }
    if (desc) { receiveDescription(recv); }
    if (ice) { receiveICE(recv); }
  }
}

function welcome() {
  log("== webrtc chat demo ==");
  log("To initiate PeerConnection, type start. Otherwise, input SDP messages.");
}

function startChat() {
  currentMode = MODE.CHAT;
  $chatlog.className = "active";
  log("------- chat enabled! -------");
}

function prepareDataChannel(channel) {
  channel.onopen = function() {
    log("Data channel opened!");
    startChat();
  }
  channel.onclose = function() {
    log("Data channel closed.");
    currentMode = MODE.INIT;
    $chatlog.className = "";
    log("------- chat disabled -------");
  }
  channel.onerror = function() {
    log("Data channel error!!");
  }
  channel.onmessage = function(msg) {
    var recv = msg.data;
    console.log(msg);
    var line = recv.trim();
    log(line);
  }
}

function start(initiator) {
  log("Starting up RTCPeerConnection...");
  pc = new PeerConnection(config, {
    optional: [
      { DtlsSrtpKeyAgreement: true },
      { RtpDataChannels: false },
    ],
  });
  pc.onicecandidate = function(evt) {
  	console.log('onicecandidate.............');
    var candidate = evt.candidate;
    // Chrome sends a null candidate once the ICE gathering phase completes.
    // In this case, it makes sense to send one copy-paste blob.
    if (null == candidate) {
      log("Finished gathering ICE candidates.");
      Signalling.send(pc.localDescription);
      return;
    }
  }
  pc.onnegotiationneeded = function(ee) {
  	console.log('onnegotiationneeded.............' + JSON.stringify(ee))
    sendOffer();
  }
  pc.ondatachannel = function(dc) {
    console.log('ondatachannel............');
    channel = dc.channel;
    log("Data Channel established... ");
    prepareDataChannel(channel);
  }

  // Creating the first data channel triggers ICE negotiation.
  if (initiator) {
  	console.log("创建datachannel");
    channel = pc.createDataChannel("test");
    prepareDataChannel(channel);
  }
}


function sendOffer() {
  var next = function(sdp) {
    log("webrtc: Created Offer");
    offer = sdp;
    pc.setLocalDescription(sdp);
  }
  var promise = pc.createOffer(next);
  if (promise) {
    promise.then(next);
  }
}

function sendAnswer() {
  var next = function (sdp) {
    log("webrtc: Created Answer");
    console.log("webrtc: Created Answer");
    answer = sdp;
    console.log(sdp);
    sendLocalSDP(sdp);
    pc.setLocalDescription(sdp)
  }
  console.log("即将应答！！");
  try {
	  var promise = pc.createAnswer(next);
	  if (promise) {
	  	console.log(12345)
	    promise.then(next);
	  }
  } catch(e) {
  	console.log("创建应答失败");
  	console.log(e)
  }
}

function receiveDescription(desc) {
	console.log('--------------')
  var sdp = new RTCSessionDescription(desc);
  // console.log('*********')
  // try {
  // 	console.log("==========================");
  //   err = pc.setRemoteDescription(sdp);
  //   console.log(err, sdp)
  // } catch (e) {
  //   log("Invalid SDP message.");
  //   return false;
  // }

  pc.setRemoteDescription(sdp)
  .then((e)=>{
  	console.log("set remote 成功");
    if ("offer" == sdp.type) {
	    sendAnswer();
	  }	
	}, (e) => {
		console.log("e=", e);
		console.log("!!!!!!!")
	})

  // console.log("Set remote ret:", err);
  // log("SDP " + sdp.type + " successfully received.");
  // if ("offer" == sdp.type) {
  //   sendAnswer();
  // }
  return true;
}

function receiveICE(ice) {
  var candidate = new RTCIceCandidate(ice);
  try {
    pc.addIceCandidate(candidate);
  } catch (e) {
    log("Could not add ICE candidate.");
    return;
  }
  log("ICE candidate successfully received: " + ice.candidate);
}

function waitForSignals() {
  currentMode = MODE.CONNECTING;
}

function ajax(obj){
    // 默认参数
    var defaults = {
        type : 'get',
        data : {},
        url : '#',
        dataType : 'text',
        async : true,
        success : function(data){console.log(data)}
    }
    // 处理形参，传递参数的时候就覆盖默认参数，不传递就使用默认参数
    for(var key in obj){//把输入的参数与设置的默认数据进行覆盖更新
        defaults[key] = obj[key];
    }
    // 1、创建XMLHttpRequest对象
    var xhr = null;
    if(window.XMLHttpRequest){
        xhr = new XMLHttpRequest();
    }else{
        xhr = new ActiveXObject('Microsoft.XMLHTTP');// 兼容ie的早期版本
    }
    // 把对象形式的参数转化为字符串形式的参数
    /* {username:'zhangsan','password':123} 转换为 username=zhangsan&password=123 */
    var param = '';
    for(var attr in obj.data){
        param += attr + '=' + obj.data[attr] + '&';
    }
    if(param){//substring(start, end)截取字符串去掉最后的&符号
        param = param.substring(0,param.length - 1);
    }
    // 处理get请求参数并且处理中文乱码问题
    if(defaults.type == 'get'){
        defaults.url += '?' + encodeURI(param);
    }
    // 2、准备发送（设置发送的参数）
    xhr.open(defaults.type,defaults.url,defaults.async); // 处理post请求参数并且设置请求头信息（必须设置）
    var data = null;
    if(defaults.type == 'post'){
        data = param;
        xhr.setRequestHeader("Content-Type","application/x-www-form-urlencoded");
    //post模式下必须加的请求头，这个请求头是告诉服务器怎么去解析请求的正文部分。
    }
    // 3、执行发送动作
    xhr.send(data);
    // 处理同步请求，不会调用回调函数
    if(!defaults.async){
        if(defaults.dataType == 'json'){
            return JSON.parse(xhr.responseText);
        }else{
            return xhr.responseText;
        }
    }
    // 4、指定回调函数（处理服务器响应数据）
    xhr.onreadystatechange = function(){
        if(xhr.readyState == 4){
            //4 获取数据成功
        if(xhr.status == 200){
            //200 获取的数据格式正确
            var data = xhr.responseText;
            if(defaults.dataType == 'json'){
                // data = eval("("+ data +")");
                data = JSON.parse(data);
                //JSON.parse把获取带的json格式的数据转化为js的对象形式可以使用
                }
                defaults.success(data);//回调函数
            }
        }
    }
}

function setRemoteSDP(sdp) {
	Signalling.receive(sdp);
}

function updateBoxInfo(sdp) {
  return new Promise(function(resolve,reject){
  	setTimeout(()=>{
      send4sdp = {}
      send4sdp.type = sdp.type
      send4sdp.sdp = sdp.sdp
      send4sdp.myrandsessionid = myrandsessionid
  	  ajax({
  	  	type: "post",
  	  	url: "https://www.yqtc.co/iamtest/ubbey/turn/box_sdp",
  	  	data: {
  	  		box_id: "fatzhou1234",
  	  		action: 0,
  	  		box_sdp: JSON.stringify(send4sdp)
  	  	},
  	  	dataType: "json",
  	  	success: function(res) {
          log("updateBoxInfo input:" + JSON.stringify(send4sdp));
          log("updateBoxInfo return:" + JSON.stringify(res));
  	  		if(res.err_no === 0 ) {
            resolve(res)
  	  		} else {
  	  			alert(res.err_msg || "获取盒子sdp错误")
            reject(res)
  	  		}
  	  	}
  	  })
  	}, 0)
  });
}

function getRemoteAppSDP() {
  return new Promise(function(resolve,reject){
    var internalGetAppSdp = function(){
      ajax({
      	type: "post",
      	url: "https://www.yqtc.co/iamtest/ubbey/turn/box_sdp",
      	data: {
      		box_id: "fatzhou1234",
      		action: 1
      	},
      	dataType: "json",
      	success: function(res) {
          log("internalGetAppSdp return:" + JSON.stringify(res));
          if(res.err_no === 0 && res.app_sdp && (tmp_app_sdp = JSON.parse(res.app_sdp)) && tmp_app_sdp.myrandsessionid == myrandsessionid) {
            resolve(res)
          }else if(res.err_no === 0 && res.app_sdp) {
            log("获取盒子sdp成功, 但是myrandsessionid不匹配，稍后自动重试")
            setTimeout(internalGetAppSdp, 3000)
      		} else {
            log("获取盒子sdp错误, 稍后自动重试")
      			setTimeout(internalGetAppSdp, 5000)
      		}
      	}
      })
    }
    setTimeout(internalGetAppSdp, 0)
  });
}

function sendLocalSDP(sdp) {
  ajax({
  	type: "post",
  	url: "https://www.yqtc.co/iamtest/ubbey/turn/app_connect",
  	data: {
  		box_id: "fatzhou",
  		action: 0,
  		app_sdp: JSON.stringify(sdp)
  	},
  	dataType: "json",
  	success: function(res) {
  		console.log(res);
  		if(res.err_no === 0) {
  			log("上传本地sdp成功");
  			log(JSON.stringify(sdp));
  		} else {
  			alert(res.err_msg || "app上传sdp错误")
  		}
  	}
  })  	
}

// Get DOM elements and setup interactions.
function init() {
  console.log("loaded");
  // Setup chatwindow.
  $chatlog = document.getElementById('chatlog');
  $chatlog.value = "";

  welcome();

  start(true);
}

function speak(msg) {
	log(msg);
	channel.send(msg);
}

var log = function(msg) {
  $chatlog.value += msg + "\n";
  console.log(msg);
  // Scroll to latest.
  $chatlog.scrollTop = $chatlog.scrollHeight;
}

window.onload = init;