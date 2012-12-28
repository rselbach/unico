// adnlib - A fully oauth-authenticated Go Twitter library
//
// Copyright 2011 The Tweetlib Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package adnlib

type ADNToken struct {
	Data struct {
		User *User `json:"user"`
	} `json:"data"`
	Meta struct {
		Code int64 `json:"code"`
	} `json:"meta"`
}

type ADNTokenList []ADNToken

type User struct {
	Id string `json:"id"`
	UserName string `json:"username"`
}

type UserList []User

