Unico
=====

Sends public activities from Google+ to Twitter and Facebook.

See it in action: http://unico.robteix.com

Libraries
---------

Unico uses a few external libraries and due to the way Google App Engine deals
with this, you'll need to deploy those along with the app. This is done
by including the source code for the libraries in the app root directory.

The libraries expected are:

* `fblib/` - from https://github.com/robteix/fblib
* `goauth2.googlecode.com/hg/oauth/` - from http://code.google.com/p/goauth2/
* `google-api-go-client.googlecode.com/hg/google-api/`
   from http://code.google.com/p/google-api-go-client/
* `google-api-go-client.googlecode.com/hg/plus/`
   from http://code.google.com/p/google-api-go-client/
* `tweetlib/` - from https://github.com/robteix/tweetlib


Setting it up
-------------

You will need to create a configuration file with information from apps you
create on Twitter and Facebook and your Google API access credentials.

1. Go to https://code.google.com/apis/console and request access to the 
Google+ API. Set up your `redirect_uri` to http://your-domain.com/oauth2callback
and appropriate alternatives (say http://localhost:8080/oauth2callback)

2. Now go to https://developers.facebook.com/apps and create a new app. This
is the app that will write on your Facebook wall on behalf of Unico. Again,
take note of the app ID and Secret.

3. Visit https://dev.twitter.com/apps/new and create a new app. Not
surprisingly, that's the app that will post to your twitter feed on
behalf of Unico.

4. Create a file named `config.json` on the app root directory and
set up **all** of the fields, like this:

        {
         "FacebookAppId" : "123456789012345",
         "FacebookAppSecret" : "abedcaefddca26db75a9199bc480789a32f42c",
         "GoogleClientId" : "1234567890.apps.googleusercontent.com",
         "GoogleClientSecret" : "1fKTqZBAadWyrFM-W3c_J1Pa",
         "TwitterConsumerKey" : "4legr0KASbkS2cyb0RgcsYH",
         "TwitterConsumerSecret" : "bfg69KxZHrs28ZyCeQr3tVoL4qUlggoS0nQyflwa3",
         "AppHost" : "unico.robteix.com",
         "AppDomain" : "unico.robteix.com",
         "SessionStoreKey" : "some-key-to-encrypt-cookies"
        }

5. That should be it. Upload it to appengine and have fun.

License
-------

Copyright 2011 The Unico Authors.  All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
