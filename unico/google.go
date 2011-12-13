// unico - Send Google+ activities to other networks
//
// Copyright 2011 The Unico Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unico

import (
	"http"
	"time"

	"goauth2.googlecode.com/hg/oauth"
	plus "google-api-go-client.googlecode.com/hg/plus/v1"
	"appengine"
	"appengine/urlfetch"
	"gorilla.googlecode.com/hg/gorilla/sessions"
)

func config(host string) *oauth.Config {
	return &oauth.Config{
		ClientId:     appConfig.GoogleClientId,
		ClientSecret: appConfig.GoogleClientSecret,
		Scope:        plus.PlusMeScope,
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		RedirectURL:  "http://" + host + "/oauth2callback",
	}
}

func emptyTransport() *oauth.Transport {
	return transport(User{})
}

func loginGoogle(w http.ResponseWriter, r *http.Request) {
	tr := emptyTransport()
	c := appengine.NewContext(r)
	tr.Transport = &urlfetch.Transport{Context: c}
	http.Redirect(w, r, tr.AuthCodeURL("foo"), http.StatusFound)
}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	code := r.FormValue("code")
	tr := emptyTransport()
	tr.Transport = &urlfetch.Transport{Context: c}
	if _, err := tr.Exchange(code); err != nil {
		serveError(c, w, err)
		return
	}

	// get info on the user
	httpClient := tr.Client()
	p, err := plus.New(httpClient)
	if err != nil {
		serveError(c, w, err)
		return
	}

	person, err := p.People.Get("me").Do()
	if err != nil {
		serveError(c, w, err)
		return
	}
	cookie := &http.Cookie{Name: "userId", Value: person.Id, Domain: appConfig.AppDomain, Path: "/", MaxAge: 30000000 /* about a year */ }
	http.SetCookie(w, cookie)

	if session, err := sessions.Session(r, "", "memcache"); err == nil {
		session["userID"] = person.Id
		sessions.Save(r, w)
	}

	user := loadUser(r, person.Id)
	if user.Id == "" {
		user := &User{Id: person.Id, GoogleAccessToken: tr.Token.AccessToken, GoogleTokenExpiry: tr.Token.TokenExpiry, GoogleRefreshToken: tr.Token.RefreshToken}
		user.GoogleLatest = time.Nanoseconds()
		saveUser(r, user)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func transport(user User) *oauth.Transport {
	return &oauth.Transport{
		Token:     &oauth.Token{AccessToken: user.GoogleAccessToken, RefreshToken: user.GoogleRefreshToken, TokenExpiry: user.GoogleTokenExpiry},
		Config:    config(appConfig.AppHost),
		Transport: &urlfetch.Transport{},
	}
}
