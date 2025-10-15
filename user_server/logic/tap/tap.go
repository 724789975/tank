package tap

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"time"
	common_config "user_server/config"

	"github.com/cloudwego/kitex/pkg/klog"
)

func GetHandle(ctx context.Context, kid, macKey, reqURI string) (string, error) {
	clientId := common_config.Get("tap.client_id").(string)
	reqHost := common_config.Get("tap.host").(string)
	// baseInfoURI := common_config.Get("tap.base_info_uri").(string)
	reqURI = reqURI + "?" + "client_id=" + clientId

	nonce := "8IBTHwOdqNKAWeKl7plt66=="
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	reqURL := "https://" + reqHost + reqURI

	macStr := timestamp + "\n" + nonce + "\n" + "GET" + "\n" + reqURI + "\n" + reqHost + "\n" + "443" + "\n\n"
	mac := hmacSha1(macStr, macKey)
	authorization := "MAC id=" + "\"" + kid + "\"" + "," + "ts=" + "\"" + timestamp + "\"" + "," + "nonce=" + "\"" + nonce + "\"" + "," + "mac=" + "\"" + mac + "\""

	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		klog.CtxErrorf(ctx, "[TAP-REQ-BUILD] tap GetHandle err: %v", err)
		return "", err
	}

	req.Header.Add("Authorization", authorization)

	resp, err := client.Do(req)
	if err != nil {
		klog.CtxErrorf(ctx, "[TAP-HTTP-DO] tap GetHandle err: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.CtxErrorf(ctx, "[TAP-READ-BODY] tap GetHandle err: %v", err)
		return "", err
	}
	return string(respBody), nil
}

func hmacSha1(valStr, keyStr string) string {
	key := []byte(keyStr)
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(valStr))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
