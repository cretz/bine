package main

import (
	"context"
	"log"
	"os"

	"github.com/cretz/bine/process/embedded"
)

// Simply calls Tor will the same parameters
func main() {
	if err := runTor(os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}

func runTor(args ...string) error {
	process, err := embedded.NewCreator().New(context.Background(), args...)
	if err == nil {
		process.Start()
		err = process.Wait()
	}
	return err
}
