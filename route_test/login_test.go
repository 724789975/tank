package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"route_test/kitex_gen/user_center"

	"github.com/gogo/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestLogin(t *testing.T) {
	// ========== 1. 配置阶段 ==========
	routeAddr := os.Getenv("ROUTE_ADDR")
	if routeAddr == "" {
		routeAddr = "http://localhost:8006"
	}
	t.Logf("[CONFIG] Route address: %s", routeAddr)

	// ========== 2. 构建请求阶段 ==========
	t.Logf("[STEP-1] Building LoginReq...")
	req := &user_center.TestLoginReq{
		TapInfo: &user_center.TapInfo{
			Avatar:  "test_avatar",
			Gender:  "test_nick_name",
			Name:    "test_user_id",
			Openid:  "test_open_id",
			Unionid: "test_union_id",
		},
	}
	t.Logf("[STEP-1] LoginReq created: TapInfo=%v", req)

	bodyAny := &anypb.Any{}
	bodyAny.MarshalFrom(req)

	// 使用 gogo/protobuf 的 proto.Marshal 序列化 Any
	anyBytes, err := proto.Marshal(bodyAny)
	if err != nil {
		t.Errorf("[ERROR] Failed to marshal Any: %v", err)
		return
	}
	t.Logf("[STEP-2] Request marshaled to protobuf, size: %d bytes", len(anyBytes))

	// 构建 user-channel header
	now := time.Now().Unix()
	userChannel := fmt.Sprintf(`{"userId":"test_user_%d","exp":%d,"ver":"1.0"}`, now, now+3600)
	t.Logf("[STEP-3] UserChannel header created: %s", userChannel)

	// 构建完整 URL
	url := fmt.Sprintf("%s/api/1.0/public/route/user-center/TestLogin", routeAddr)
	t.Logf("[STEP-4] Target URL: %s", url)

	// ========== 3. 发送请求阶段 ==========
	t.Logf("[STEP-5] Creating HTTP request...")
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(anyBytes))
	if err != nil {
		t.Errorf("[ERROR] Failed to create HTTP request: %v", err)
		return
	}

	// 设置请求头
	httpReq.Header.Set("user-channel", userChannel)
	httpReq.Header.Set("Content-Type", "application/octet-stream")
	t.Logf("[STEP-5] HTTP headers set: user-channel=%s, Content-Type=application/octet-stream", userChannel)

	// 发送请求
	t.Logf("[STEP-6] Sending HTTP POST request...")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Errorf("[ERROR] Failed to send request: %v", err)
		t.Errorf("[ERROR] Possible causes: route service not running, network issue, or incorrect address")
		return
	}
	t.Logf("[STEP-6] Request sent successfully, response received")

	defer resp.Body.Close()

	// ========== 4. 读取响应阶段 ==========
	t.Logf("[STEP-7] Reading response body...")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("[ERROR] Failed to read response body: %v", err)
		return
	}
	t.Logf("[STEP-7] Response body read, size: %d bytes", len(body))

	// ========== 5. 输出结果 ==========
	t.Logf("========================================")
	t.Logf("[RESULT] Response Status: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	t.Logf("[RESULT] Response Headers:")
	for k, v := range resp.Header {
		t.Logf("  - %s: %v", k, v)
	}
	t.Logf("[RESULT] Response Body:")

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Logf("  Raw (not JSON): %s", string(body))
		t.Logf("  JSON parse error: %v", err)
	} else {
		jsonBytes, _ := json.MarshalIndent(result, "  ", "")
		t.Logf("  %s", string(jsonBytes))
	}

	// 检查响应码
	if resp.StatusCode >= 400 {
		t.Errorf("[FAIL] HTTP error status: %d", resp.StatusCode)
		if code, ok := result["code"]; ok {
			t.Errorf("[FAIL] Business error code: %v", code)
		}
		if msg, ok := result["msg"]; ok {
			t.Errorf("[FAIL] Error message: %v", msg)
		}
	} else {
		t.Logf("========================================")
		t.Logf("[SUCCESS] Request completed successfully!")
	}
}
