// unico - Send Google+ activities to other networks
//
// Copyright 2011 The Unico Authors.  All rights reserved.
// Use of this source code is governed by the Simplified BSD
// license that can be found in the LICENSE file.

package unico

import (
	"html"
	"net/http"
	"regexp"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
)

var (
	reParagraphs = regexp.MustCompile("</p>")
	reBreaks     = regexp.MustCompile("<br */?>")
	reTags       = regexp.MustCompile("<[^>]+>")
)

// 1. convert </p> to new line
// 2. convert <br/> to new line
// 3. strip all tags

func removeTags(str string) string {
	str = reParagraphs.ReplaceAllString(str, "\n")
	str = reBreaks.ReplaceAllString(str, "\n")
	return html.UnescapeString(reTags.ReplaceAllString(str, ""))
}

func serve404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "404", nil)
}

func serveError(c appengine.Context, w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "error", err)
	c.Errorf("serveError: %v\n", err)
}

func loadUser(r *http.Request, id string) User {
	c := appengine.NewContext(r)
	var user User
	_, err := memcache.JSON.Get(c, "user"+id, &user)
	if err == nil {
		return user
	}

	key := datastore.NewKey(c, "User", id, 0, nil)
	if err := datastore.Get(c, key, &user); err != nil {
		user.Id = ""
		user.Active = false
	}
	return user
}

func saveUser(r *http.Request, user *User) error {
	c := appengine.NewContext(r)

	a := user.Active
	user.enableIfNeeded()
	if user.Active != a && user.Active { // user just enabled
		user.GoogleLatest = time.Now().UnixNano()
	}
	memcache.JSON.Set(c, &memcache.Item{Key: "user" + user.Id, Object: *user})
	key := datastore.NewKey(c, "User", user.Id, 0, nil)
	_, err := datastore.Put(c, key, user)
	return err
}

type MemoryUser struct {
	Name  string
	Image string
}

func memUser(c appengine.Context, id string) (mu MemoryUser) {
	memcache.JSON.Get(c, "memuser"+id, &mu)
	return
}

func memUserSave(c appengine.Context, id string, mu MemoryUser) {
	memcache.JSON.Set(c, &memcache.Item{Key: "memuser" + id, Object: mu})
}

func memUserDelete(c appengine.Context, id string) {
	memcache.Delete(c, "memuser"+id)
}
