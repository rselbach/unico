package adnlib

import (
	"fmt"
	"testing"
	)
/*
func TestLogin(t *testing.T) {
	config := &Config{    
                ConsumerKey: "7Sk45jzPRMAswzRRtkVjnTt3zgXMDe2g",
                ConsumerSecret: "89nbdYuGRwHtBswWe9UWUwxGSj6NZgza",
                Callback: "http://unico.robteix.com/adnauth"}
        tr := &Transport{Config: config,
                              Token: &Token{}}
	//tl, _ := New(tr.Client());
	u := tr.AuthURL()
	fmt.Printf("%s\n", u)
	tok, err := tr.RequestAccessToken("AQAAAAAAAevd8X1ntaI1X9lbintY1ZZhqub5JPntIAyV2xL1yJeOCVdKzyQZkpuva58KtXMJZLTaHjXtnKPT-riWqRB9o6a4tbYRxojA69lBDC3gj39a_viozcoZCz0D0T4ReEF_5Kt6")
	if (err != nil) {
		t.Error("teste")
	} else {
		fmt.Printf("%v\n", tok)
	}
}
*/
func TestPost(t *testing.T) {
       config := &Config{
                ConsumerKey: "7Sk45jzPRMAswzRRtkVjnTt3zgXMDe2g",
                ConsumerSecret: "89nbdYuGRwHtBswWe9UWUwxGSj6NZgza",
                Callback: "http://unico.robteix.com/adnauth"}
        tr := &Transport{Config: config,
                         Token: &Token{"AQAAAAAAAevd5vmJ2KIfVTZ8kDUZdoR-FSaTGacFduvIGuH9K424rpXQ2gwvrJ0E9AxHQy424oObIMDPE2Yl8LrBu30_56qbEQ"}}
	tl, err := New(tr.Client())
	if err != nil {
		t.Error(err)
	}
	//tl.Stream.Post("Testing something...").Do()
	tok, err := tl.Stream.Token().Do()

	fmt.Printf("%v %v\n", tok.Data.User, err)
}
