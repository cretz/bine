package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cretz/bine/process/embedded"
	"github.com/cretz/bine/tor"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Parse flags. By default, non-verbose served in the current working dir.
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Whether to have verbose logging")
	var directory string
	flag.StringVar(&directory, "dir", ".", "The directory to serve (current dir is default)")
	flag.Parse()
	var err error
	if directory, err = filepath.Abs(directory); err != nil {
		return err
	}
	// Start tor
	startConf := &tor.StartConf{ProcessCreator: embedded.NewCreator()}
	if verbose {
		startConf.DebugWriter = os.Stdout
	} else {
		startConf.ExtraArgs = []string{"--quiet"}
	}
	fmt.Printf("Starting and registering onion service to serve files from %v\n", directory)
	fmt.Println("Please wait a couple of minutes...")
	t, err := tor.Start(nil, startConf)
	if err != nil {
		return err
	}
	defer t.Close()
	// Wait at most a few minutes to publish the service
	listenCtx, listenCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer listenCancel()
	// Create an onion service to listen on a random local port but show as
	// Do version 3, it's faster to set up
	onion, err := t.Listen(listenCtx, &tor.ListenConf{RemotePorts: []int{80}, Version3: true})
	if err != nil {
		return err
	}
	defer onion.Close()
	// Start server asynchronously
	fmt.Printf("Open Tor browser and navigate to http://%v.onion\n", onion.ID)
	fmt.Println("Press enter to exit")
	server := &http.Server{Handler: http.FileServer(http.Dir(directory))}
	defer server.Shutdown(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(onion) }()
	// Wait for key asynchronously
	go func() {
		fmt.Scanln()
		errCh <- nil
	}()
	// Stop when one happens
	defer fmt.Println("Closing")
	return <-errCh
}
