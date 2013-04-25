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
	"robteix.com/v1/fblib"
	"net/http"
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

		http.Redirect(w, r, fc.AuthURL("http://"+appConfig.AppHost+"/fb?id="+id, "offline_access,publish_stream"), http.StatusFound)
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
	case "article", "video":
		// post a link
		err = fc.PostLink(content, attachment.Url)
	default:
		if obj != nil {
			err = fc.PostLink(content, obj.Url)
		}
	}

	if err == fblib.ErrOAuth {
		user.DisableFacebook()
		saveUser(r, user)
	}
	c.Debugf("publishActivityToFacebook(%s): err=%v\n", kind, err)

}
