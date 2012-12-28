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
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
)

const (
	apiURL  = "https://alpha-api.app.net"
)

var (
	ErrOAuth = errors.New("OAuth failure")
)

type errorReply struct {
	Error   string `json:"error"`
	Request string `json:"request"`
}

type errorsReply struct {
	Errors string
}

func checkResponse(res *http.Response) error {
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return nil
	}
	slurp, err := ioutil.ReadAll(res.Body)
	fmt.Printf("%s\n", slurp)
	if err == nil {
		jerr := new(errorReply)
		err = json.Unmarshal(slurp, jerr)
		if err == nil && jerr.Error != "" {
			return errors.New(jerr.Error)
		}
		errs := new(errorsReply)
		err = json.Unmarshal(slurp, errs)
		if err == nil && errs.Errors != "" {
			return errors.New(errs.Errors)
		}

	}
	return fmt.Errorf("googleapi: got HTTP response code %d and error reading body: %v",
		res.StatusCode, err)
}

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client}
	s.Stream = &StreamService{s: s}
	return s, nil
}

type Service struct {
	client *http.Client

	Stream    *StreamService
}

type StreamService struct {
	s *Service
}

// Automatically generated
// ./misc/gen/gen -service Stream -call Post -endpoint stream/0/posts -method POST -options text:string,in_reply_to:string

type StreamPostCall struct {
	s    *Service
	opt_ map[string]interface{}
}


func (r *StreamService) Post(text string) *StreamPostCall {
	c := &StreamPostCall{s: r.s, opt_: make(map[string]interface{})}
	c.opt_["text"] = text
	return c
}

func (c *StreamPostCall) InReplyTo(in_reply_to string) *StreamPostCall {
	c.opt_["in_reply_to"] = in_reply_to
	return c
}


func (c *StreamPostCall) Do() (*interface{}, error) {
	var body io.Reader = nil
	params := make(url.Values)

	if v, ok := c.opt_["text"]; ok {
		params.Set("text", fmt.Sprintf("%v", v))
	}

	if v, ok := c.opt_["in_reply_to"]; ok {
		params.Set("in_reply_to", fmt.Sprintf("%v", v))
	}

	urls := fmt.Sprintf("%s/%s", apiURL, "stream/0/posts")
	urls += "?" + params.Encode()
	body = bytes.NewBuffer([]byte(params.Encode()))
	ctype := "application/x-www-form-urlencoded"
	req, _ := http.NewRequest("POST", urls, body)
	req.Header.Set("Content-Type", ctype)
	res, err := c.s.client.Do(req)

	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	ret := new(interface{})
	if err := json.NewDecoder(res.Body).Decode(ret); err != nil && reflect.TypeOf(err) != reflect.TypeOf(&json.UnmarshalTypeError{}){
		return ret, err
	}
	return ret, nil
}

// Automatically generated
// ./misc/gen/gen -service Stream -call Token -endpoint stream/0/token -method GET -ret ADNToken

type StreamTokenCall struct {
	s    *Service
	opt_ map[string]interface{}
}


func (r *StreamService) Token() *StreamTokenCall {
	c := &StreamTokenCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}


func (c *StreamTokenCall) Do() (*ADNToken, error) {
	var body io.Reader = nil
	params := make(url.Values)

	urls := fmt.Sprintf("%s/%s", apiURL, "stream/0/token")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	res, err := c.s.client.Do(req)

	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	ret := new(ADNToken)
	if err := json.NewDecoder(res.Body).Decode(ret); err != nil && reflect.TypeOf(err) != reflect.TypeOf(&json.UnmarshalTypeError{}){
		return ret, err
	}
	return ret, nil
}
