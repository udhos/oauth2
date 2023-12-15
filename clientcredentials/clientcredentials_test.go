package clientcredentials

import (
	"testing"
	"time"
)

const (
	expectSucess  = true
	expectFailure = false
)

type expectResult bool

type parseTokenTestCase struct {
	name             string
	token            string
	expect           expectResult
	expectAcessToken string
	expectExpire     time.Duration
}

var parseTokenTestTable = []parseTokenTestCase{
	{"empty", "", expectFailure, "", 0},
	{"no fields", `{}`, expectFailure, "", 0},
	{"missing access_token", `{"other":"field"}`, expectFailure, "", 0},
	{"simple", `{"access_token":"abc"}`, expectSucess, "abc", 0},
	{"expire integer", `{"access_token":"abc","expires_in":300}`, expectSucess, "abc", 300 * time.Second},
	{"expire float", `{"access_token":"abc","expires_in":300.0}`, expectSucess, "abc", 300 * time.Second},
	{"expire string", `{"access_token":"abc","expires_in":"300"}`, expectSucess, "abc", 300 * time.Second},
}

func TestParseToken(t *testing.T) {
	for _, data := range parseTokenTestTable {
		buf := []byte(data.token)
		info, errParse := parseToken(buf)
		success := errParse == nil
		if success != bool(data.expect) {
			t.Errorf("%s: expectedError=%t gotError=%t error:%v", data.name, data.expect, success, errParse)
			continue
		}

		if info.accessToken != data.expectAcessToken {
			t.Errorf("%s: expectedAccessToken=%s gotAccessToken=%s", data.name, data.expectAcessToken, info.accessToken)
		}

		if info.expiresIn != data.expectExpire {
			t.Errorf("%s: expectedExpire=%v gotExpire=%v", data.name, data.expectExpire, info.expiresIn)
		}

		if !t.Failed() {
			t.Logf("%s: ok", data.name)
		}
	}
}
