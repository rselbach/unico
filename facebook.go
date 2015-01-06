// gplus2others - Send Google+ activities to other networks
//
// Copyright 2011 The gplus2others Authors.  All rights reserved.
// Use of this source code is governed by the Simplified BSD
// license that can be found in the LICENSE file.

package gplus2others

import (
	"appengine"
	"appengine/memcache"
	"appengine/urlfetch"
	plus "google.golang.org/api/plus/v1"
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"github.com/robteix/fblib"
)

func fbHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	id := r.FormValue("id")

	if id == "" {
		serveError(c, w, errors.New("Missing ID Parameter"))
		return
	}

	fc := fblib.NewFacebookClient(appConfig.FacebookAppId, appConfig.FacebookAppSecret)
	fc.Transport = &urlfetch.Transport{Context: c}

	code := r.FormValue("code")
	if code == "" {

		http.Redirect(w, r, fc.AuthURL("http://"+appConfig.AppHost+"/fb?id="+id, "offline_access,publish_actions"), http.StatusFound)
		return
	}

	fc.RequestAccessToken(code, "http://"+appConfig.AppHost+"/fb?id="+id)
	user := loadUser(r, id)
	if user.Id == "" {
		serveError(c, w, errors.New("Invalid user ID"))
		return
	}

	user.FBAccessToken = fc.AccessToken
	fbuser, fberr := fc.CurrentUser()
	if fberr != nil {
		c.Errorf("fc.CurrentUser() return error: %s\n", fberr)
	}
	user.FBId = fbuser.Id
	user.FBName = fbuser.Name
	saveUser(r, &user)

	http.Redirect(w, r, "/", http.StatusFound)

}

func publishActivityToFacebook(w http.ResponseWriter, r *http.Request, act *plus.Activity, user *User) {
	c := appengine.NewContext(r)
	fc := fblib.NewFacebookClient(appConfig.FacebookAppId, appConfig.FacebookAppSecret)
	fc.Transport = &urlfetch.Transport{Context: c}
	fc.AccessToken = user.FBAccessToken

	_ = w

	var attachment *plus.ActivityObjectAttachments
	obj := act.Object
	kind := ""
	content := ""

	if act.Verb == "share" {
		content = act.Annotation
		//if content == "" {
		//	content = "Resharing " + obj.Actor.DisplayName
		//}
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

	var err error

	switch kind {
	case "status":
		// post a status update
		err = fc.PostStatus(content)
		return
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
			media, err := ioutil.ReadAll(resp.Body)
			c.Debugf("Reading contents of %s (%v)\n", mediaUrl, err)
			if err != nil {
				break
			}
			memcache.Add(c, &memcache.Item{Key: "picture" + mediaUrl, Value: media})
		} else {
			media = item.Value
		}
		// now we post it
		photo := fblib.Photo{
			Message:  content,
			Source:   media,
			FileName: fileName,
		}
		err = fc.PostPhoto(photo)
		c.Debugf("Posting %s to FB (%v)\n", mediaUrl, err)
	case "article", "video":
		// post a link
		link := fblib.Link{}
		link.Text = content
		link.Url = attachment.Url
		if attachment.FullImage != nil {
			link.Image = attachment.FullImage.Url
		}
		err = fc.PostLink(link)
	default:
		if obj != nil {
			link := fblib.Link{
				Text: content,
				Url:  obj.Url,
			}
			err = fc.PostLink(link)
		}
	}

	if err == fblib.ErrOAuth {
		user.DisableFacebook()
		saveUser(r, user)
	}
	c.Debugf("publishActivityToFacebook(%s): err=%v\n", kind, err)

}
