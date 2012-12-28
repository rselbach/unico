// adnlib - A fully oauth-authenticated Go Twitter library
//
// Copyright 2011 The Tweetlib Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package adnlib

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	authURL         = "https://account.app.net/oauth/authenticate"     // user authorization endpoint
	accessTokenURL  = "https://account.app.net/oauth/access_token"     // access token endpoint
)

type Config struct {
	ConsumerKey    string
	ConsumerSecret string
	Callback       string
}

type Token struct {
	AccessToken  string `json:"access_token"` 
}

func (t *Transport) AuthURL() string {
	return fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=write_post", authURL, t.ConsumerKey, t.callback())
}
func (t *Transport) nonce() string {
	s := time.Now()
	return strconv.FormatInt(s.Unix(), 10)
}

func (c *Config) callback() string {
	if c.Callback != "" {
		return c.Callback
	}
	return "oob"
}

type Transport struct {
	*Config
	*Token

	// Transport is the HTTP transport to use when making requests.
	// It will default to http.DefaultTransport if nil.
	// (It should never be an oauth.Transport.)
	Transport http.RoundTripper
}

// Client returns an *http.Client that makes OAuth-authenticated requests.
func (t *Transport) Client() *http.Client {
	return &http.Client{Transport: t}
}

func (t *Transport) transport() http.RoundTripper {
	if t.Transport != nil {
		return t.Transport
	}
	return http.DefaultTransport
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Config == nil {
		return nil, errors.New("no Config supplied")
	}
	if t.Token == nil {
		return nil, errors.New("no Token supplied")
	}

	// Refresh the Token if it has expired.
	//if t.Expired() {
	//	if err := t.Refresh(); err != nil {
	//		return nil, err
	//	}
	//}
	// Make the HTTP request.
	t.sign(req)
	return t.transport().RoundTrip(req)
}

// Twitter requires that all authenticated requests be
// signed with HMAC-SHA1
//
// https://dev.twitter.com/docs/auth/oauth
//
// The base string is a special combination
// of parameters:
//
//      httpMethod + "&" +
//      url_encode(  base_uri ) + "&" +
//      sorted_query_params.each  { | k, v |
//          url_encode ( k ) + "%3D" +
//          url_encode ( v )
//      }.join("%26")
//
// And then you sign this with HMAC-SHA1 with the key:
//
//    consumer_secret&oauth_token_secret
//
func (t *Transport) sign(req *http.Request) error {
	urlForBase := strings.Split(req.URL.String(), "?")[0]
	if req.Method == "POST" {
		req.URL, _ = url.Parse(urlForBase)
	}
	// Create the Authentication header
	authHeader := "Bearer " + t.AccessToken
	req.Header.Set("Authorization", authHeader)
	return nil
}

func (t *Transport) RequestAccessToken(code string) (*Token, error) {

	u := &url.Values{"client_id": {t.ConsumerKey},
		"client_secret": {t.ConsumerSecret},
		"grant_type": {"authorization_code"},
		"redirect_uri": {t.callback()},
		"code": {code}}
	var body io.Reader
	body = bytes.NewBuffer([]byte(u.Encode()))
	fmt.Printf("YYYYY %s\n", body)
	urls := fmt.Sprintf("%s", accessTokenURL)
	req, err := http.NewRequest("POST", urls, body)
	if err != nil {
		return nil, err
	}
	resp, err := t.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	//if resp.StatusCode != 200 {
	//	return os.NewError("Authentication Error")
	//}
	defer resp.Body.Close()

	ret := new(Token)
        err = json.NewDecoder(resp.Body).Decode(ret)
	if err != nil {
		return nil, err
	}
	if ret.AccessToken == "" {
		return nil, errors.New("Empty response")
	}
	return ret, nil
}

func (t *Transport) shouldEscape(c byte) bool {
	switch {
	case c >= 0x41 && c <= 0x5A:
		return false
	case c >= 0x61 && c <= 0x7A:
		return false
	case c >= 0x30 && c <= 0x39:
		return false
	case c == '-', c == '.', c == '_', c == '~':
		return false
	}
	return true
}

func (tr *Transport) percentEncode(s string) string {
	spaceCount, hexCount := 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if tr.shouldEscape(c) {
			hexCount++
		}
	}

	if spaceCount == 0 && hexCount == 0 {
		return s
	}

	t := make([]byte, len(s)+2*hexCount)
	j := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case tr.shouldEscape(c):
			t[j] = '%'
			t[j+1] = "0123456789ABCDEF"[c>>4]
			t[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		default:
			t[j] = s[i]
			j++
		}
	}
	return string(t)
}
