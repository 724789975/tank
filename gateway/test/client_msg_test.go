package test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"gate_way_module/kitex_gen/gate_way"
	"gate_way_module/kitex_gen/user_center"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestLogin(t *testing.T) {
	// ========== 1. 配置阶段 ==========
	gatewayAddr := os.Getenv("GATEWAY_ADDR")
	if gatewayAddr == "" {
		gatewayAddr = "ws://localhost:12001/ws"
	}
	t.Logf("[CONFIG] Gateway WebSocket address: %s", gatewayAddr)

	// ========== 2. 连接 WebSocket ==========
	t.Logf("[STEP-1] Connecting to WebSocket server...")
	conn, _, err := websocket.DefaultDialer.Dial(gatewayAddr, nil)
	if err != nil {
		t.Errorf("[ERROR] Failed to connect to WebSocket: %v", err)
		t.Errorf("[ERROR] Possible causes: gateway service not running, incorrect address")
		return
	}
	defer conn.Close()
	t.Logf("[STEP-1] WebSocket connection established")

	// ========== 3. 发送第一条消息: gateway.LoginRequest ==========
	t.Logf("[STEP-2] Building LoginRequest (gateway)...")
	loginRequest := &gate_way.LoginRequest{
		Id: "test_user_12345",
	}
	t.Logf("[STEP-2] LoginRequest created: id=%s", loginRequest.Id)

	// 封装为 Any
	loginReqAny := &anypb.Any{}
	if err := loginReqAny.MarshalFrom(loginRequest); err != nil {
		t.Errorf("[ERROR] Failed to marshal LoginRequest to Any: %v", err)
		return
	}

	// 序列化为二进制
	loginReqBytes, err := proto.Marshal(loginReqAny)
	if err != nil {
		t.Errorf("[ERROR] Failed to marshal LoginRequest: %v", err)
		return
	}
	t.Logf("[STEP-3] LoginRequest marshaled, size: %d bytes", len(loginReqBytes))

	// 发送第一条消息
	t.Logf("[STEP-4] Sending LoginRequest via WebSocket...")
	if err := conn.WriteMessage(websocket.BinaryMessage, loginReqBytes); err != nil {
		t.Errorf("[ERROR] Failed to send LoginRequest: %v", err)
		return
	}
	t.Logf("[STEP-4] LoginRequest sent successfully")

	// ========== 4. 接收第一条响应 ==========
	t.Logf("[STEP-5] Waiting for LoginResp...")
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	msgType1, response1, err := conn.ReadMessage()
	if err != nil {
		t.Errorf("[ERROR] Failed to read LoginResp: %v", err)
		return
	}
	t.Logf("[STEP-5] LoginResp received, type: %d, size: %d bytes", msgType1, len(response1))

	// 解析第一条响应
	loginRespAny := &anypb.Any{}
	if err := proto.Unmarshal(response1, loginRespAny); err != nil {
		t.Logf("[WARN] Failed to unmarshal LoginResp as protobuf: %v", err)
		t.Logf("[RESULT] Raw LoginResp: %s", string(response1))
	} else {
		t.Logf("[RESULT] LoginResp type_url: %s", loginRespAny.TypeUrl)
		loginResp := &gate_way.LoginResp{}
		if err := loginRespAny.UnmarshalTo(loginResp); err != nil {
			t.Logf("[WARN] Failed to unmarshal to LoginResp: %v", err)
		} else {
			t.Logf("[RESULT] LoginResp code: %v", loginResp.Code)
		}
	}

	// ========== 5. 发送第二条消息: ClientMsgReq (TestLogin) ==========
	t.Logf("[STEP-6] Building ClientMsgReq (user_center/TestLogin)...")
	testLoginReq := &user_center.TestLoginReq{
		TapInfo: &user_center.TapInfo{
			Avatar:  "test_avatar",
			Gender:  "test_gender",
			Name:    "test_name",
			Openid:  "test_openid",
			Unionid: "test_unionid",
		},
	}
	t.Logf("[STEP-6] TestLoginReq created: TapInfo=%v", testLoginReq.TapInfo)

	// 封装 TestLoginReq 为 Any
	testLoginAny := &anypb.Any{}
	if err := testLoginAny.MarshalFrom(testLoginReq); err != nil {
		t.Errorf("[ERROR] Failed to marshal TestLoginReq to Any: %v", err)
		return
	}

	// 构建 ClientMsgReq
	clientMsgReq := &gate_way.ClientMsgReq{
		ServiceName: "user-center",
		Method:      "TestLogin",
		Data:        testLoginAny,
	}
	t.Logf("[STEP-7] ClientMsgReq: service=%s, method=%s", clientMsgReq.ServiceName, clientMsgReq.Method)

	// 封装 ClientMsgReq 为 Any
	clientMsgAny := &anypb.Any{}
	if err := clientMsgAny.MarshalFrom(clientMsgReq); err != nil {
		t.Errorf("[ERROR] Failed to marshal ClientMsgReq to Any: %v", err)
		return
	}

	// 序列化为二进制
	clientMsgBytes, err := proto.Marshal(clientMsgAny)
	if err != nil {
		t.Errorf("[ERROR] Failed to marshal ClientMsgReq: %v", err)
		return
	}
	t.Logf("[STEP-8] ClientMsgReq marshaled, size: %d bytes", len(clientMsgBytes))

	// 发送第二条消息
	t.Logf("[STEP-9] Sending ClientMsgReq via WebSocket...")
	if err := conn.WriteMessage(websocket.BinaryMessage, clientMsgBytes); err != nil {
		t.Errorf("[ERROR] Failed to send ClientMsgReq: %v", err)
		return
	}
	t.Logf("[STEP-9] ClientMsgReq sent successfully")

	// ========== 6. 接收第二条响应 ==========
	t.Logf("[STEP-10] Waiting for TestLoginRsp...")
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	msgType2, response2, err := conn.ReadMessage()
	if err != nil {
		t.Errorf("[ERROR] Failed to read TestLoginRsp: %v", err)
		return
	}
	t.Logf("[STEP-10] TestLoginRsp received, type: %d, size: %d bytes", msgType2, len(response2))

	// ========== 7. 解析第二条响应 ==========
	t.Logf("========================================")
	t.Logf("[RESULT] Message Type: %d", msgType2)

	// 尝试解析为 protobuf Any
	responseAny := &anypb.Any{}
	if err := proto.Unmarshal(response2, responseAny); err != nil {
		// 如果不是 protobuf，尝试解析为 JSON
		t.Logf("[RESULT] Raw response (not protobuf): %s", string(response2))
		t.Logf("[WARN] Failed to unmarshal as protobuf: %v", err)

		var result map[string]interface{}
		if err := json.Unmarshal(response2, &result); err != nil {
			t.Logf("[RESULT] Raw string: %s", string(response2))
		} else {
			jsonBytes, _ := json.MarshalIndent(result, "  ", "  ")
			t.Logf("[RESULT] Response body:\n  %s", string(jsonBytes))
		}
	} else {
		t.Logf("[RESULT] Protobuf Any type_url: %s", responseAny.TypeUrl)
		t.Logf("[RESULT] Response size: %d bytes", len(response2))

		// 尝试解析为 TestLoginRsp
		testLoginRsp := &user_center.TestLoginRsp{}
		if err := responseAny.UnmarshalTo(testLoginRsp); err != nil {
			t.Logf("[WARN] Failed to unmarshal to TestLoginRsp: %v", err)
		} else {
			t.Logf("[RESULT] TestLoginRsp:")
			t.Logf("  - Code: %v", testLoginRsp.Code)
			t.Logf("  - Msg: %s", testLoginRsp.Msg)
			if testLoginRsp.Data != nil && testLoginRsp.Data.TapInfo != nil {
				t.Logf("  - Data.TapInfo:")
				t.Logf("      Avatar: %s", testLoginRsp.Data.TapInfo.Avatar)
				t.Logf("      Gender: %s", testLoginRsp.Data.TapInfo.Gender)
				t.Logf("      Name: %s", testLoginRsp.Data.TapInfo.Name)
				t.Logf("      Openid: %s", testLoginRsp.Data.TapInfo.Openid)
				t.Logf("      Unionid: %s", testLoginRsp.Data.TapInfo.Unionid)
			}
		}
	}

	t.Logf("========================================")
	t.Logf("[SUCCESS] Login flow completed!")
}
