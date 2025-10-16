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

	hmacSha1 := func(valStr, keyStr string) string {
		key := []byte(keyStr)
		mac := hmac.New(sha1.New, key)
		mac.Write([]byte(valStr))

		return base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	clientId := "oy5mqqrghbgdlrfmf4"
	kid := "1/hsYICnlCUZ19wPhUCXapwMVZo0nqj9NQkVGRYKiEyF3gr_QU15tlezQObRtUeJemepEfxqDMt60ztDd32sBxwlJfNU_Y8SA_6MFI5fGwROLaT7WMAe6zV__nhTMIVJM4zJ2eCaAY-Q40WvPAU89Mvn1-lAO2zcbE7ru0khO7fvFg4FYgzOPcQGTb9QVgchNTwejEqbRO3Zytp6Zc_XY7vg_Wp-AzZ34BHXhlqQxNY1Bnd6cwGBod9awWGW1NLKjESPRxBeFLC93JRoAR4bWqQP66_bS65e-lKdStH1qYOvbYw1X6g730l-XEOmZXQZh5j9SKfr3oY0uw_SJNgxmu5w"
	macKey := "npNS8eUyw9omtiMtjUVy2t0aTpbAQu6P0GsuZWfA"

	nonce := "8IBTHwOdqNKAWeKl7plt66=="
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	timestamp = "1760061241"
	reqHost := "open.tapapis.cn"
	reqURI := "/account/profile/v1?client_id=" + clientId
	reqURL := "https://" + reqHost + reqURI

	macStr := timestamp + "\n" + nonce + "\n" + "GET" + "\n" + reqURI + "\n" + reqHost + "\n" + "443" + "\n\n"
	fmt.Println(macStr)

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

	fmt.Println(req)
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
