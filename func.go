package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	fdk "github.com/fnproject/fdk-go"
)

func main() {
	fdk.Handle(fdk.HandlerFunc(webhookHandler))
}

func webhookHandler(ctx context.Context, in io.Reader, out io.Writer) {
	log.Println("invoked webhook on ", time.Now())

	fnCtx := fdk.GetContext(ctx)
	eventType := fnCtx.Header().Get("X-GitHub-Event")
	log.Println("eventType ", eventType)

	if eventType == "release" {

		buf := new(bytes.Buffer)
		buf.ReadFrom(in)
		payload := buf.String()
		//log.Println(payload)

		signatureFromGithub := fnCtx.Header().Get("X-Hub-Signature")
		//log.Println("signatureFromGithub ", signatureFromGithub)

		if !matchSignature(signatureFromGithub, fnCtx.Config()["github_webhook_secret"], payload) {
			log.Println("Signature did not match. Webhook was not invoked by Github")
			return
		}

		var notification newReleaseNotification

		json.NewDecoder(strings.NewReader(payload)).Decode(&notification)

		//out.Write([]byte("eventType ? " + eventType))
		//out.Write([]byte("message " + notification.Details()))
		err := tweet(notification.Details(), fnCtx.Config()["twitter_consumerkey"], fnCtx.Config()["twitter_consumersecret"], fnCtx.Config()["twitter_accesstoken"], fnCtx.Config()["twitter_accesstokensecret"])
		if err != nil {
			fdk.WriteStatus(out, 500)
			prob := "Could not tweet new release details due to " + err.Error()
			log.Println(prob)
			return
			//out.Write([]byte(prob))
		}
		log.Println(notification.Details())
		out.Write([]byte(notification.Details()))
	}

}

func matchSignature(signature, key, payload string) bool {

	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(payload))
	expectedHMAC := mac.Sum(nil)

	//signature format is sha1=foobarred
	githubHMAC, _ := hex.DecodeString(strings.Split(signature, "=")[1])
	match := hmac.Equal(githubHMAC, expectedHMAC)
	log.Println("signature match ? ", match)
	return match
}

func tweet(tweet, consumerkey, consumersecret, accesstoken, accesstokensecret string) error {
	anaconda.SetConsumerKey(consumerkey)
	anaconda.SetConsumerSecret(consumersecret)
	api := anaconda.NewTwitterApi(accesstoken, accesstokensecret)

	_, err := api.PostTweet(tweet, url.Values{})

	if err != nil {
		//log.Println("COULD NOT POST TWEET")
		return err
	}

	return nil
	//return "tweeted new release details !!!"

}

type newReleaseNotification struct {
	Release    release `json:"release"`
	Repository repo    `json:"repository"`
}

type release struct {
	Version string `json:"tag_name"`
	Link    string `json:"html_url"`
}

type repo struct {
	Name string `json:"full_name"`
}

func (notification newReleaseNotification) Details() string {
	return "Release " + notification.Release.Version + " for " + notification.Repository.Name + " in out! Grab it while it's hot - " + notification.Release.Link
}
