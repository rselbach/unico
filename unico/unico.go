// unico - Send Google+ activities to other networks
//
// Copyright 2011 The Unico Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unico

import (
	"http"
	"io/ioutil"
	"json"
	"template"
	"tweetlib"
	"time"
	plus "google-api-go-client.googlecode.com/hg/plus/v1"
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"appengine/urlfetch"

	appengineSessions "gorilla.googlecode.com/hg/gorilla/appengine/sessions"
	"gorilla.googlecode.com/hg/gorilla/sessions"
)

var appConfig struct {
	FacebookAppId         string
	FacebookAppSecret     string
	GoogleClientId        string
	GoogleClientSecret    string
	TwitterConsumerKey    string
	TwitterConsumerSecret string
	AppHost               string
	AppDomain             string
	SessionStoreKey       string
}

var (
	templates = template.SetMust(template.ParseSetFiles(
		"404.html",
		"home.html",
		"header.html",
		"footer.html",
		"error.html"))
)

func init() {
	// Read configuration file
	content, err := ioutil.ReadFile("config.json")
	if err == nil {
		err = json.Unmarshal(content, &appConfig)
	}
	if err != nil {
		panic("Can't load configuration")
	}
	
	// Make sure every conf option has been completed, except
	// for AppDomain, because it is useful to test the app with
	// localhost but some browsers require localhost cookies
	// to have Domain as ""
	if appConfig.FacebookAppId == "" || appConfig.FacebookAppSecret == "" ||
		appConfig.GoogleClientId == "" || appConfig.GoogleClientSecret == "" ||
		appConfig.TwitterConsumerKey == "" || appConfig.TwitterConsumerSecret == "" ||
		appConfig.AppHost == "" ||
		appConfig.SessionStoreKey == "" {
		panic("Invalid configuration")
	}

	// Register the datastore and memcache session stores.
	sessions.SetStore("datastore", new(appengineSessions.DatastoreSessionStore))
	sessions.SetStore("memcache", new(appengineSessions.MemcacheSessionStore))

	// Set secret keys for the session stores.
	sessions.SetStoreKeys("datastore", []byte(appConfig.SessionStoreKey))
	sessions.SetStoreKeys("memcache", []byte(appConfig.SessionStoreKey))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/twitter", twitterHandler)
	http.HandleFunc("/loginGoogle", loginGoogle)
	http.HandleFunc("/oauth2callback", googleCallbackHandler)
	http.HandleFunc("/fb", fbHandler)
	http.HandleFunc("/sync", syncHandler)
	http.HandleFunc("/deleteAccount", deleteAccountHandler)
	http.HandleFunc("/deleteFacebook", deleteFacebookHandler)
	http.HandleFunc("/deleteTwitter", deleteTwitterHandler)

}

// Displays the home page. 
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if appConfig.AppHost == "" {
		appConfig.AppHost = r.Host
	}
	c := appengine.NewContext(r)
	if r.Method != "GET" || r.URL.Path != "/" {
		serve404(w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	params := make(map[string]string)

	// Look for a browser cookie containing the user id
	// We can use this to load the user information
	userCookie, err := r.Cookie("userId")
	var user User
	if err == nil {
		user = loadUser(r, userCookie.Value)
	}
	c.Debugf("loadUser: %v\n", user)
	if user.Id != "" {
		if session, err := sessions.Session(r, "", "datastore"); err == nil {
			session["userID"] = user.Id
			f := sessions.Save(r, w)
			c.Debugf("saveSession: %v\n", f)
		}

		if user.TwitterId != "" {

			item := new(memcache.Item)
			item, err := memcache.Get(c, "pic"+user.Id)

			if err != nil {
				// get the user profile pic
				conf := &tweetlib.Config{
					ConsumerKey:    appConfig.TwitterConsumerKey,
					ConsumerSecret: appConfig.TwitterConsumerSecret}
				tok := &tweetlib.Token{
					OAuthSecret: user.TwitterOAuthSecret,
					OAuthToken:  user.TwitterOAuthToken}
				tr := &tweetlib.Transport{Config: conf,
					Token:     tok,
					Transport: &urlfetch.Transport{Context: c}}

				tl, _ := tweetlib.New(tr.Client())
				u, err := tl.Users.Show().UserId(user.TwitterId).Do()
				if err == nil {
					params["pic"] = u.ProfileImageUrl
					memcache.Add(c, &memcache.Item{Key: "pic" + user.Id, Value: []byte(u.ProfileImageUrl)})
				}

			} else {
				params["pic"] = string(item.Value)
			}

		}
		params["twitterid"] = user.TwitterId
		params["twittername"] = user.TwitterScreenName
		params["googleid"] = user.Id
		params["fbid"] = user.FBId
		params["fbname"] = user.FBName

		mu := memUser(c, user.Id)
		if mu.Name == "" {
			tr := transport(user)
			tr.Transport = &urlfetch.Transport{Context: c}
			p, _ := plus.New(tr.Client())
			person, err := p.People.Get(user.Id).Do()
			c.Debugf("Home people get: %v,(%v)\n", person, err)
			if err == nil {
				mu.Image = person.Image.Url
				mu.Name = person.DisplayName
				memUserSave(c, user.Id, mu)
			}

		}
		params["googleimg"] = mu.Image
		params["googlename"] = mu.Name

	}

	if err := templates.Execute(w, "home", params); err != nil {
		serveError(c, w, err)
		c.Errorf("%v", err)
		return
	}

}

func syncHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	q := datastore.NewQuery("User").
		Filter("Active=", true)

	for t := q.Run(c); ; {
		var u User
		_, err := t.Next(&u)
		if err == datastore.Done {
			break
		}
		if err != nil {
			serveError(c, w, err)
			return
		}

		syncStream(w, r, &u)
	}
	// schedule next run
}

func syncStream(w http.ResponseWriter, r *http.Request, user *User) {
	c := appengine.NewContext(r)
	tr := transport(*user)
	tr.Transport = &urlfetch.Transport{Context: c}

	httpClient := tr.Client()
	p, err := plus.New(httpClient)
	if err != nil {
		serveError(c, w, err)
		return
	}

	latest := user.GoogleLatest
	c.Debugf("syncStream: fetching for %s\n", user.Id)
	activityFeed, err := p.Activities.List(user.Id, "public").MaxResults(5).Do()
	if err != nil {
		c.Debugf("syncStream: activity fetch failed for %s. Err: %v\n", user.Id, err)
		return
	}

	for _, act := range activityFeed.Items {
		published, _ := time.Parse(time.RFC3339, act.Published)
		nPub := published.Nanoseconds()

		c.Debugf("syncStream: user: %s, nPub: %v, Latest: %v\n", user.Id, nPub, user.GoogleLatest)

		if nPub > user.GoogleLatest {
			if user.HasFacebook() {
				publishActivityToFacebook(w, r, act, user)
			}
			if user.HasTwitter() {
				publishActivityToTwitter(w, r, act, user)
			}
		}
		if nPub > latest {
			latest = nPub
		}
	}
	if latest > user.GoogleLatest {
		user.GoogleLatest = latest
		saveUser(r, user)
	}
}

func deleteAccountHandler(w http.ResponseWriter, r *http.Request) {

	id := ""
	session, err := sessions.Session(r, "", "datastore")
	c1:= appengine.NewContext(r)
	c1.Debugf("deleteAccount: id=%v, session=%v, err=%v\n", session["userID"], session, err)
	if err == nil {
		if session["userID"] != nil {
		id = session["userID"].(string)
		}
	}
	if id != "" {
		user := loadUser(r, id)
		if user.Id != "" {
			c := appengine.NewContext(r)
			key := datastore.NewKey(c, "User", user.Id, 0, nil)
			datastore.Delete(c, key)
			session["userID"] = ""
			sessions.Save(r, w)
			memUserDelete(c, user.Id)
			memcache.Delete(c, "user" + user.Id)
			http.SetCookie(w, &http.Cookie{Name: "userId", Value: "", Domain: appConfig.AppDomain, Path: "/", MaxAge: -1})
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func deleteTwitterHandler(w http.ResponseWriter, r *http.Request) {

	id := ""
	session, err := sessions.Session(r, "", "datastore")
	if err == nil {
		c := appengine.NewContext(r)
		c.Debugf("session: %v\n",  session)
		id = session["userID"].(string)
	}
	if id != "" {
		user := loadUser(r, id)
		if user.Id != "" {
			user.DisableTwitter()
			saveUser(r, &user)
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func deleteFacebookHandler(w http.ResponseWriter, r *http.Request) {

	id := ""
	session, err := sessions.Session(r, "", "datastore")
	if err == nil {
		id = session["userID"].(string)
	}
	if id != "" {
		user := loadUser(r, id)
		if user.Id != "" {
			user.DisableFacebook()
			saveUser(r, &user)
		}
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
