package main

import (
	"context"
	"fmt"
	"log"
	"os"

	tor035 "github.com/cretz/bine/process/embedded/tor-0.3.5"
)

// Simply calls Tor will the same parameters
func main() {
	if err := runTor(os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}

func runTor(args ...string) error {
	creator := tor035.NewProcessCreator()
	creator.SetupControlSocket = true
	process, err := creator.New(context.Background(), args...)
	if err == nil {
		fmt.Printf("Socket pointer: %v\n", tor035.ProcessControlSocket(process))
		process.Start()
		err = process.Wait()
	}
	return err
}
