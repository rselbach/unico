// gplus2others - Send Google+ activities to other networks
//
// Copyright 2011 The gplus2others Authors.  All rights reserved.
// Use of this source code is governed by the Simplified BSD
// license that can be found in the LICENSE file.

package gplus2others

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"google.golang.org/api/plus/v1"
	"gopkg.in/tweetlib.v2"

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
		serveError(c, w, errors.New("Invalid Action Parameter"))
		return
	}
}

func twitterVerify(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("oauth_token")
	id := r.FormValue("id")
	c := appengine.NewContext(r)

	if id == "" {
		serveError(c, w, errors.New("Missing ID parameter"))
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
	u, err := tl.Account.VerifyCredentials(nil)
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
		serveError(c, w, errors.New("Missing ID parameter"))
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

	c.Debugf("Post (%s):\n\tkind: %s\n\tcontent: %s\n", user.TwitterId, kind, content)
	var err error
	switch kind {
	case "status":
		// post a status update
		_, err = tl.Tweets.Update(shorten(c, "status", content, act.Url, tl), nil)
	case "status_share":
		_, err = tl.Tweets.Update(shorten(c, "link", content, act.Url, tl), nil)
	case "article":
		// post a link
		c.Debugf("Article (%s):\n\tcontent: %s\n\turl: %s\n", user.TwitterId, content, attachment.Url)

		if content == attachment.Url || content == "" {
			if attachment.DisplayName != "" {
				content = attachment.DisplayName
			} else {
				content = "Shared a link."
			}
		}
		_, err = tl.Tweets.Update(shorten(c, "link", content, attachment.Url, tl), nil)
	case "photo":
		// download photo
		mediaUrl := attachment.FullImage.Url
		fileName := path.Base(mediaUrl)

		var media []byte
		item, err := memcache.Get(c, "picture"+mediaUrl)
		if err != nil {
			client := urlfetch.Client(c)
			resp, err := client.Get(attachment.FullImage.Url)
			c.Debugf("Downloading %s (%v)\n", mediaUrl, err)
			if err != nil {
				break
			}
			media, err = ioutil.ReadAll(resp.Body)
			c.Debugf("Reading contents of %s (%v)\n", mediaUrl, err)
			if err != nil {
				break
			}
			memcache.Add(c, &memcache.Item{Key: "picture" + mediaUrl, Value: media})
		} else {
			media = item.Value
		}
		// now we post it
		tweetMedia := &tweetlib.TweetMedia{
			Filename: fileName,
			Data:     media}
		_, err = tl.Tweets.UpdateWithMedia(shorten(c, "media", content, act.Url, tl), tweetMedia, nil)
		c.Debugf("Tweeting %s (%v)\n", mediaUrl, err)
	default:
		if obj != nil {
			_, err = tl.Tweets.Update(shorten(c, "link", content, obj.Url, tl), nil)
		}
	}

	c.Debugf("publishActivityToTwitter(%s): err=%v\n", kind, err)
}

// queries twitter.com for the current configuration
func twitterConf(c appengine.Context, client *tweetlib.Client) *tweetlib.Configuration {
	var conf *tweetlib.Configuration
	_, err := memcache.JSON.Get(c, "twitterConfig", conf)
	if err != nil {
		conf, err := client.Help.Configuration()
		if err != nil {
			return nil
		}
		// we have our length, let's cache it
		// (+1 to account for the space char)
		memcache.JSON.Set(c, &memcache.Item{Key: "twitterConfig", Object: conf})
	}
	return conf
}

func shorten(c appengine.Context, kind, content, url string, tl *tweetlib.Client) string {
	max := 140
	conf := twitterConf(c, tl)
	if conf == nil {
		// use some reasonable defaults if we could not
		// query twitter.com
		conf = &tweetlib.Configuration{
			CharactersReservedPerMedia: 25,
			ShortUrlLengthHttps:        25,
			ShortUrlLength:             24,
		}
	}
	if kind == "media" {
		// -1 for the space character
		max = max - conf.CharactersReservedPerMedia - 1
		kind = "status"
	}

	if kind == "status" && len(content) <= max {
		return content
	}

	var tcl int
	if strings.HasPrefix(url, "https:") {
		tcl = conf.ShortUrlLengthHttps
	} else {
		tcl = conf.ShortUrlLength
	}
	// add room for a space
	tcl++
	// leave room for URL (shortened by twitter)
	l := max - tcl
	if l < len(content) {
		return fmt.Sprintf("%s... %s", content[:l-3], url)
	}
	return fmt.Sprintf("%s %s", content, url)
}
