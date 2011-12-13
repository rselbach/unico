// unico - Send Google+ activities to other networks
//
// Copyright 2011 The Unico Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unico

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

	//FB Info
	FBAccessToken string
	FBName        string
	FBId          string

	Active bool
}

func (user *User) HasFacebook() bool {
	return (user.FBId != "")
}

func (user *User) HasTwitter() bool {
	return (user.TwitterId != "")
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

func (user *User) disableIfNeeded() {
	user.Active = (user.FBId == "" && user.TwitterId == "")
}
