package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	fsnotify "gopkg.in/fsnotify.v1"
)

var webhook = flag.String("webhook-url", "", "the url to send a request to when the specified config map volume directory has been updated")
var webhookMethod = flag.String("webhook-method", "POST", "the HTTP method url to use to send the webhook")
var webhookStatusCode = flag.Int("webhook-status-code", 200, "the HTTP status code indicating successful triggering of reload")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] volume-dir...\n\n"+
			" volume-dir...\n"+
			"    	one or more directories to be watched\n\n"+
			" Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		log.Println("Missing volume-dir")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	if *webhook == "" {
		log.Println("Missing webhook")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					if filepath.Base(event.Name) == "..data" {
						log.Println("config map updated")
						req, err := http.NewRequest(*webhookMethod, *webhook, nil)
						if err != nil {
							log.Println("error:", err)
							continue
						}
						resp, err := http.DefaultClient.Do(req)
						if err != nil {
							log.Println("error:", err)
							continue
						}
						resp.Body.Close()
						if resp.StatusCode != *webhookStatusCode {
							log.Println("error:", "Received response code", resp.StatusCode, ", expected", *webhookStatusCode)
							continue
						}
						log.Println("successfully triggered reload")
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	for _, volumeDir := range flag.Args() {
		err = watcher.Add(volumeDir)
		if err != nil {
			log.Fatal(err)
		}
	}
	<-done
}
