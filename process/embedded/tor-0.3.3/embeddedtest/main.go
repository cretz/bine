package main

import (
	"context"
	"log"
	"os"

	tor033 "github.com/cretz/bine/process/embedded/tor-0.3.3"
)

// Simply calls Tor will the same parameters
func main() {
	if err := runTor(os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}

func runTor(args ...string) error {
	process, err := tor033.NewCreator().New(context.Background(), args...)
	if err == nil {
		process.Start()
		err = process.Wait()
	}
	return err
}
