// unico - Send Google+ activities to other networks
//
// Copyright 2011 The Unico Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unico

import (
	"fmt"
	"http"
	"os"
	"tweetlib"
	plus "google-api-go-client.googlecode.com/hg/plus/v1"
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
)

var _ = fmt.Println

func twitterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.FormValue("action") {
	case "init":
		signInTwitterHandler(w, r)
	case "temp":
		twitterVerify(w, r)
	default:
		c := appengine.NewContext(r)
		serveError(c, w, os.NewError("Invalid Action Parameter"))
		return
	}
}

func twitterVerify(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("oauth_token")
	id := r.FormValue("id")
	c := appengine.NewContext(r)

	if id == "" {
		serveError(c, w, os.NewError("Missing ID parameter"))
		return
	}

	item, _ := memcache.Get(c, token)

	secret := string(item.Value)
	verifier := r.FormValue("oauth_verifier")

	conf := &tweetlib.Config{
		ConsumerKey:    appConfig.TwitterConsumerKey,
		ConsumerSecret: appConfig.TwitterConsumerSecret}
	tok := &tweetlib.Token{}
	tr := &tweetlib.Transport{Config: conf,
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c}}

	tt := &tweetlib.TempToken{Token: token, Secret: secret}
	tok, err := tr.AccessToken(tt, verifier)
	if err != nil {
		c := appengine.NewContext(r)
		serveError(c, w, err)
		c.Errorf("%v", err)
		return
	}
	tr.Token = tok
	tl, _ := tweetlib.New(tr.Client())
	u, err := tl.Account.VerifyCredentials().Do()
	fmt.Printf("err=%v\n", err)
	user := loadUser(r, id)
	user.TwitterOAuthToken = tok.OAuthToken
	user.TwitterOAuthSecret = tok.OAuthSecret
	user.TwitterId = u.IdStr
	user.TwitterScreenName = u.ScreenName
	if err := saveUser(r, &user); err != nil {
		serveError(c, w, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)

}

func signInTwitterHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)
	id := r.FormValue("id")
	if id == "" {
		serveError(c, w, os.NewError("Missing ID parameter"))
		return
	}

	conf := &tweetlib.Config{
		ConsumerKey:    appConfig.TwitterConsumerKey,
		ConsumerSecret: appConfig.TwitterConsumerSecret,
		Callback:       "http://" + appConfig.AppHost + "/twitter?action=temp&id=" + id}
	tok := &tweetlib.Token{}
	tr := &tweetlib.Transport{Config: conf,
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c}}

	tt, err := tr.TempToken()
	if err != nil {
		c := appengine.NewContext(r)
		serveError(c, w, err)
		c.Errorf("%v", err)
		return
	}
	item := &memcache.Item{
		Key:   tt.Token,
		Value: []byte(tt.Secret),
	}
	// Add the item to the memcache, if the key does not already exist
	memcache.Add(c, item)

	http.Redirect(w, r, tt.AuthURL(), http.StatusFound)
}

func publishActivityToTwitter(w http.ResponseWriter, r *http.Request, act *plus.Activity, user *User) {
	c := appengine.NewContext(r)

	conf := &tweetlib.Config{
		ConsumerKey:    appConfig.TwitterConsumerKey,
		ConsumerSecret: appConfig.TwitterConsumerSecret}
	tok := &tweetlib.Token{OAuthToken: user.TwitterOAuthToken, OAuthSecret: user.TwitterOAuthSecret}
	tr := &tweetlib.Transport{Config: conf,
		Token:     tok,
		Transport: &urlfetch.Transport{Context: c}}

	tl, _ := tweetlib.New(tr.Client())

	var attachment *plus.ActivityObjectAttachments
	obj := act.Object
	kind := "status"
	content := act.Title
	if obj != nil {
		if len(obj.Attachments) > 0 {
			attachment = obj.Attachments[0]
			kind = attachment.ObjectType
		}
		content = obj.Content
	}
	if act.Annotation != "" {
		content = act.Annotation
	}
	content = removeTags(content)
	var err os.Error
	switch kind {
	case "status":
		// post a status update
		_, err = tl.Tweets.Update(shorten(140, content, act.Url)).Do()
	case "article":
		// post a link
		_, err = tl.Tweets.Update(shortenLink(140, content, attachment.Url)).Do()
	default:
		if obj != nil {
			_, err = tl.Tweets.Update(shortenLink(140, content, obj.Url)).Do()
		}
	}

	if err == tweetlib.ErrOAuth {
		user.DisableTwitter()
		saveUser(r, user)
	}
	c.Debugf("publishActivityToTwitter(%s): err=%v\n", kind, err)
}

func shortenLink(max int, content, url string) string {
	if len(content) < max-25 {
		return content + " " + url
	}
	return content[:max-25] + " " + url
}

func shorten(max int, content, url string) string {
	if len(content) < max {
		return content
	}
	// leave 25 for URL (shortened by twitter)
	return content[:max-25] + " " + url
}
