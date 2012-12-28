// twitterlib - A simple, fully oauth-authenticated Twitter library

// Copyright (c) 2011, Roberto Teixeira <r@robteix.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facebooklib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"net/url"
	"time"
)

var (
	ErrOAuth = errors.New("OAuth authorization failure")
)

const (
	tokenRequestURL = "https://www.facebook.com/dialog/oauth"         // request token endpoint
	accessTokenURL  = "https://graph.facebook.com/oauth/access_token" // access token endpoint

	apiURL = "https://graph.facebook.com"
)

type FacebookClient struct {
	APIKey      string
	AppSecret   string
	AccessToken string
	Transport   http.RoundTripper
}

type TempToken struct {
	Token  string
	Secret string
}

func nonce() string {
	s := time.Now()
	return strconv.FormatInt(s.Unix(), 10)
}

func NewFacebookClient(key, secret string) *FacebookClient {
	return &FacebookClient{APIKey: key,
		AppSecret: secret,
		Transport: http.DefaultTransport}
}

func (fc *FacebookClient) AuthURL(redirectURI, scope string) string {
	params := make(url.Values)
	if scope != "" {
		params.Set("scope", scope)
	}
	params.Set("client_id", fc.APIKey)
	params.Set("redirect_uri", redirectURI)
	return fmt.Sprintf("%s?%s", tokenRequestURL, params.Encode())
}

func (fc *FacebookClient) RequestAccessToken(code, redirectURI string) error {
	var body io.Reader
	body = bytes.NewBuffer([]byte(""))
	params := make(url.Values)
	params.Set("client_id", fc.APIKey)
	params.Set("redirect_uri", redirectURI)
	params.Set("client_secret", fc.AppSecret)
	cmdStr := fmt.Sprintf("%s?%s&code=%s", accessTokenURL, params.Encode(), code)
	fmt.Printf("cmdStr := %s\n", cmdStr)
	req, err := http.NewRequest("GET", cmdStr, body)
	if err != nil {
		return err
	}
	resp, err := fc.Transport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var respBody []byte
	respBody, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 400 {
		return fc.parseError(respBody)
	}

	data, err := url.ParseQuery(string(respBody))
	if err != nil {
		return err
	}
	fc.AccessToken = data.Get("access_token")
	return nil
}

func (fc *FacebookClient) parseError(respBody []byte) error {
	var buf map[string]interface{}
	json.Unmarshal(respBody, &buf)
	errorMap := buf["error"]
	if errorMap != nil {
		error := errorMap.(map[string]interface{})
		msg := error["message"].(string)
		kind := error["type"].(string)
		if msg != "" {
			if kind == "OAuthException" {
				return ErrOAuth
			} else {
				return errors.New(error["message"].(string))
			}
		}
	}
	return errors.New("Unknown error")
}

func (fc *FacebookClient) GetUser(id string) (*User, error) {
	u := make(url.Values)
	u.Set("access_token", fc.AccessToken)
	body := bytes.NewBuffer([]byte(u.Encode()))
	cmdStr := fmt.Sprintf("%s/%s?%s", apiURL, id, u.Encode())
	req, err := http.NewRequest("GET", cmdStr, body)
	if err != nil {
		return nil, err
	}
	resp, err := fc.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var respBody []byte
	respBody, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 400 {
		return nil, fc.parseError(respBody)
	}

	fmt.Printf("%s\n", respBody)
	return nil, nil
}

// Performs API call based on httpMethod
// returns the response body as string and error/nil
func (fc *FacebookClient) Call(httpMethod, endpoint string, params url.Values) ([]byte, error) {
	body := bytes.NewBuffer([]byte(params.Encode()))
	cmdStr := fmt.Sprintf("%s/%s?access_token=%s", apiURL, endpoint, fc.AccessToken)
	if httpMethod == "GET" {
		cmdStr = cmdStr + "&" + params.Encode()
	}
	req, err := http.NewRequest(httpMethod, cmdStr, body)
	if err != nil {
		return []byte(""), err
	}
	resp, err := fc.Transport.RoundTrip(req)
	if err != nil {
		return []byte(""), err
	}
	defer resp.Body.Close()
	var respBody []byte
	respBody, _ = ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return []byte(""), fc.parseError(respBody)
	}

	return respBody, nil
}

func (fc *FacebookClient) User(id string) (*User, error) {
	u := new(url.Values)
	resp, err := fc.Call("GET", id, *u)
	if err != nil {
		return nil, err
	}
	user := new(User)

	//if err = json.Unmarshal(resp, user); err != nil {
	//	return nil, os.NewError(fmt.Sprintf("fc.User error -> %s (resp body: '%s')", err, resp))
	//}
	json.Unmarshal(resp, user)
	user.Client = fc
	return user, nil
}

func (fc *FacebookClient) CurrentUser() (*User, error) {
	return fc.User("me")
}

func (fc *FacebookClient) PostLink(message, link string) error {
	u := make(url.Values)
	u.Add("message", message)
	u.Add("link", link)
	_, err := fc.Call("POST", "me/links", u)

	return err
}

func (fc *FacebookClient) PostStatus(message string) error {
	u := make(url.Values)
	u.Add("message", message)
	_, err := fc.Call("POST", "me/feed", u)
	return err
}
