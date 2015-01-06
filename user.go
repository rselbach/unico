// gplus2others - Send Google+ activities to other networks
//
// Copyright 2011 The gplus2others Authors.  All rights reserved.
// Use of this source code is governed by the Simplified BSD
// license that can be found in the LICENSE file.

package gplus2others

type User struct {
	Id string

	// Google
	GoogleAccessToken  string `json:"access_token"`
	GoogleRefreshToken string `json:"refresh_token"`
	GoogleTokenExpiry  int64  `json:"expires_in"`

	GoogleLatest int64

	// Twitter Info
	TwitterOAuthToken  string
	TwitterOAuthSecret string
	TwitterScreenName  string
	TwitterId          string
	TwitterSinceId     string

	// app.net
	ADNAccessToken string
	ADNScreenName  string
	ADNId          string

	//FB Info
	FBAccessToken string
	FBName        string
	FBId          string

	Active bool

	// Services the user has access to
//	Services []Services
}

func (user *User) HasFacebook() bool {
	return (user.FBId != "")
}

func (user *User) HasTwitter() bool {
	return (user.TwitterId != "")
}

func (user *User) HasADN() bool {
	return (user.ADNId != "")
}

func (user *User) DisableTwitter() {
	user.TwitterId = ""
	user.TwitterOAuthSecret = ""
	user.TwitterOAuthToken = ""
	user.TwitterScreenName = ""
}

func (user *User) DisableFacebook() {
	user.FBAccessToken = ""
	user.FBId = ""
	user.FBName = ""
}

func (user *User) DisableADN() {
	user.ADNId = ""
	user.ADNAccessToken = ""
	user.ADNScreenName = ""
}

func (user *User) enableIfNeeded() {
	user.Active = (user.FBId != "" || user.TwitterId != "" || user.ADNId != "")
}
