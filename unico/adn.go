// unico - Send Google+ activities to other networks
//
// Copyright 2011 The Unico Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unico

import (
	"appengine"
	"appengine/urlfetch"
	plus "code.google.com/p/google-api-go-client/plus/v1"
	"errors"
	"fmt"
	"github.com/robteix/adnlib"
	"net/http"
)

var _ = fmt.Println

func adnHandler(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("code") != "" {
		adnVerify(w, r)
	} else {
		signInADNHandler(w, r)
	}
}

func adnVerify(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	c := appengine.NewContext(r)

	if id == "" {
		serveError(c, w, errors.New("Missing ID parameter"))
		return
	}

	code := r.FormValue("code")

	conf := &adnlib.Config{
		ConsumerKey:    appConfig.ADNConsumerKey,
		ConsumerSecret: appConfig.ADNConsumerSecret,
		Callback:       "http://" + appConfig.AppDomain + "/adnauth?id=" + id}
	tok := &adnlib.Token{}
	tr := &adnlib.Transport{Config: conf,
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c}}
	c.Debugf("Requesting ADN Token with code %s\n", code)
	tok, err := tr.RequestAccessToken(code)
	if err != nil {
		c := appengine.NewContext(r)
		serveError(c, w, err)
		c.Errorf("%v", err)
		return
	}
	tr.Token = tok
	tl, _ := adnlib.New(tr.Client())
	adnTok, err := tl.Stream.Token().Do()
	if err != nil {
		fmt.Printf("err=%v\n", err)
		serveError(c, w, err)
		return
	}
	user := loadUser(r, id)
	user.ADNAccessToken = tok.AccessToken
	user.ADNId = adnTok.Data.User.Id
	user.ADNScreenName = adnTok.Data.User.UserName
	if err := saveUser(r, &user); err != nil {
		serveError(c, w, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)

}

func signInADNHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)
	id := r.FormValue("id")
	if id == "" {
		serveError(c, w, errors.New("Missing ID parameter"))
		return
	}

	conf := &adnlib.Config{
		ConsumerKey:    appConfig.ADNConsumerKey,
		ConsumerSecret: appConfig.ADNConsumerSecret,
		Callback:       "http://" + appConfig.AppHost + "/adnauth?id=" + id}
	tok := &adnlib.Token{}
	tr := &adnlib.Transport{Config: conf,
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c}}

	http.Redirect(w, r, tr.AuthURL(), http.StatusFound)
}

func publishActivityToADN(w http.ResponseWriter, r *http.Request, act *plus.Activity, user *User) {
	c := appengine.NewContext(r)

	conf := &adnlib.Config{
		ConsumerKey:    appConfig.ADNConsumerKey,
		ConsumerSecret: appConfig.ADNConsumerSecret}
	tok := &adnlib.Token{AccessToken: user.ADNAccessToken}
	tr := &adnlib.Transport{Config: conf,
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c}}

	tl, _ := adnlib.New(tr.Client())

	var attachment *plus.ActivityObjectAttachments
	obj := act.Object
	kind := ""
	content := ""

	if act.Verb == "share" {
		content = act.Annotation
		if content == "" {
			content = "Resharing " + obj.Actor.DisplayName
		}
		kind = "status_share"
	} else {
		kind = "status"
		if obj != nil {
			if len(obj.Attachments) > 0 {
				attachment = obj.Attachments[0]
				kind = attachment.ObjectType
			}
			content = obj.Content
		} else {
			content = act.Title
		}

	}
	content = removeTags(content)

	c.Debugf("Post (%s):\n\tkind: %s\n\tcontent: %s\n", user.ADNId, kind, content)
	var err error
	switch kind {
	case "status":
		// post a status update
		_, err = tl.Stream.Post(adnShorten(140, content, act.Url)).Do()
	case "status_share":
		_, err = tl.Stream.Post(adnShortenLink(140, content, act.Url)).Do()
	case "article":
		// post a link
		c.Debugf("Article (%s):\n\tcontent: %s\n\turl: %s\n", user.ADNId, content, attachment.Url)

		if content == attachment.Url || content == "" {
			if attachment.DisplayName != "" {
				content = attachment.DisplayName
			} else {
				content = "Shared a link."
			}
		}
		_, err = tl.Stream.Post(adnShortenLink(140, content, attachment.Url)).Do()
	default:
		if obj != nil {
			_, err = tl.Stream.Post(adnShortenLink(140, content, obj.Url)).Do()
		}
	}

	if err == adnlib.ErrOAuth {
		user.DisableADN()
		saveUser(r, user)
	}
	c.Debugf("publishActivityToADN(%s): err=%v\n", kind, err)
}

func adnShortenLink(max int, content, url string) string {
	// maximum size for the context itself
	contentMax := max - len(url) - 1
	if len(content) <= contentMax {
		return content + " " + url
	}
	return content[:contentMax] + " " + url
}

func adnShorten(max int, content, url string) string {
	if len(content) < max {
		return content
	}
	// maximum size for the context itself
	contentMax := max - len(url) - 1
	// leave 25 for URL (adnShortened by adn)
	return content[:contentMax] + " " + url
}
