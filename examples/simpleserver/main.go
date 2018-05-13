package main

import (
	"log"
	"net/http"
)

func main() {
	// Start tor with default config
	t, err := tor.Start(nil)
	if err != nil {
		log.Fatal(err)
	}
	// Add a handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})
	// Create an onion service to listen on 8080 but show as 80
	l, err := t.Listen(&tor.OnionConf{Port: 80, TargetPort: 8080})
	if err != nil {
		log.Fatal(err)
	}
	// Serve on HTTP
	log.Fatal(http.Serve(l, nil))
}
