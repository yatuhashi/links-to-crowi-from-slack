package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/crowi/go-crowi"
	"github.com/nlopes/slack"
	"golang.org/x/net/context"
)

func WikiUpdate(text, url, token, path string) error {
	config := crowi.Config{
		URL:   url,
		Token: token,
	}
	client, err := crowi.NewClient(config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Pages.Get(ctx, path)
	if err != nil {
		return err
	}

	if !res.OK {
		return errors.New(res.Error)
	}

	body := res.Page.Revision.Body
	index := strings.Index(body, "[LinkList]")
	head := body[:index+12]
	tail := body[index+12:]

	t := time.Now()
	const layout = "01/02"
	body = head + "\n" + t.Format(layout) + " " + text + "\n" + tail

	res, err = client.Pages.Update(ctx, res.Page.ID, body)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	url := flag.String("url", "", "https://server.net")
	wikitoken := flag.String("wtoken", "", "crowi access-token")
	path := flag.String("path", "", "page path")
	slacktoken := flag.String("stoken", "", "slack xoxb-access-token")
	flag.Parse()

	if *url == "" || *wikitoken == "" || *path == "" {
		log.Fatal("Error required -url, -wtoken, -path, -stoken")
	}

	api := slack.New(*slacktoken)
	logger := log.New(os.Stdout, "slack-crowi: ", log.Lshortfile|log.LstdFlags)
	slack.SetLogger(logger)
	api.SetDebug(true)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if ev.Type == "message" {
				log.Printf("Message: %v\n", ev.Text)
				linkindex := strings.Index(ev.Text, "<@U7NGFCBNX>")
				if linkindex == -1 { // not mention
					continue
				}
				text := ev.Text[linkindex+12:]
				text = text[:strings.Index(text, "\n")]
				if text == "" {
					continue
				}
				if err := WikiUpdate(text, *url, *wikitoken, *path); err != nil {
					log.Println(err)
				}
			}
		}
	}
}
