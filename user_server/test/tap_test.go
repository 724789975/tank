package test

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestTap(t *testing.T) {

	hmacSha1 := func (valStr, keyStr string) string {
		key := []byte(keyStr)
		mac := hmac.New(sha1.New, key)
		mac.Write([]byte(valStr))

		return base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	clientId := "oy5mqqrghbgdlrfmf4"
	kid := "1/TBb9UU5l8KUFesdZbESlDswvs4GnYqepUEsv8NiLvspxasuv4h6m7_karII5-SuNRsqmkBSEW8AyKXUwzMvuQ1QsZuaP8mHjsJQAd33SAUNQrMFm1AqavV2YevBO-pIF8HLYQUYmv-6J1ePJ1B-DUFKRXVXMkXAIv9rl5wiGmn7glWnikgGzAx3oqDy1l0tj4NEHHSXYE_ySpqoCy4fXt-x7SEBQTOoUszaU1sag2OlqcfstAcJxa-f6Lhxa37jcKBN0sla7e0LhiYKJYdaOQyoP8POTGvbZ3A41YByvINh_BKPEwa0luGIkHMDWkOuMBquX_wzmQ52LhcHlge0Fyg"
	macKey := "IEBIQHUyWLSzgWlaqgD7at6ge1Yle6MMOurWM497"

	nonce := "8IBTHwOdqNKAWeKl7plt66=="
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	reqHost := "open.tapapis.cn"
	reqURI := "/account/profile/v1?client_id=" + clientId
	reqURL := "https://" + reqHost + reqURI

	macStr := timestamp + "\n" + nonce + "\n" + "GET" + "\n" + reqURI + "\n" + reqHost + "\n" + "443" + "\n\n"
	mac := hmacSha1(macStr, macKey)
	authorization := "MAC id=" + "\"" + kid + "\"" + "," + "ts=" + "\"" + timestamp + "\"" + "," + "nonce=" + "\"" + nonce + "\"" + "," + "mac=" + "\"" + mac + "\""

	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// q := req.URL.Query()
	// q.Add("client_id", clientId)
	// req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", authorization)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(respBody))
}

