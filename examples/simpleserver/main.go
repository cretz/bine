package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cretz/bine/tor"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Start tor with default config (can set start conf's DebugWriter to os.Stdout for debug logs)
	fmt.Println("Starting and registering onion service, please wait a couple of minutes...")
	t, err := tor.Start(nil, nil)
	if err != nil {
		return err
	}
	defer t.Close()
	// Add a handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, Dark World!"))
	})
	// Wait at most a few minutes to publish the service
	listenCtx, listenCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer listenCancel()
	// Create an onion service to listen on 8080 but show as 80
	onion, err := t.Listen(listenCtx, &tor.ListenConf{LocalPort: 8080, RemotePorts: []int{80}})
	if err != nil {
		return err
	}
	defer onion.Close()
	// Serve on HTTP
	fmt.Printf("Open Tor browser and navigate to http://%v.onion\n", onion.ID)
	return http.Serve(onion, nil)
}
