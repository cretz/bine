package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cretz/bine/process/embedded"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	p, err := embedded.NewCreator().New(context.Background(), "--version")
	if err != nil {
		return err
	}
	fmt.Printf("Starting...\n")
	if err = p.Start(); err != nil {
		return err
	}
	fmt.Printf("Waiting...\n")
	return p.Wait()
}
