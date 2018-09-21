package main

import (
	"context"
	"fmt"
	"log"
	"net/textproto"
	"os"

	"github.com/cretz/bine/control"
	tor035 "github.com/cretz/bine/process/embedded/tor-0.3.5"
)

// Simply calls Tor will the same parameters, unless "embedconn" is the arg
func main() {
	fmt.Printf("Provider version: %v\n", tor035.ProviderVersion())
	var err error
	if len(os.Args) == 2 && os.Args[1] == "embedconn" {
		fmt.Println("Testing embedded conn")
		err = testEmbedConn()
	} else {
		fmt.Println("Running Tor with given args")
		err = runTor(os.Args[1:]...)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func runTor(args ...string) error {
	process, err := tor035.NewCreator().New(context.Background(), args...)
	if err == nil {
		process.Start()
		err = process.Wait()
	}
	return err
}

func testEmbedConn() error {
	process, err := tor035.NewCreator().New(context.Background())
	if err != nil {
		return fmt.Errorf("Failed creating process: %v", err)
	}
	// Try to create an embedded conn
	embedConn, err := process.EmbeddedControlConn()
	if err != nil {
		return fmt.Errorf("Failed creating embedded control conn: %v", err)
	}
	if err = process.Start(); err != nil {
		return fmt.Errorf("Failed starting process: %v", err)
	}
	controlConn := control.NewConn(textproto.NewConn(embedConn))
	info, err := controlConn.GetInfo("version")
	if err != nil {
		return fmt.Errorf("Failed getting version: %v", err)
	}
	fmt.Printf("Got info, %v: %v\n", info[0].Key, info[0].Val)
	if err = process.Wait(); err != nil {
		return fmt.Errorf("Failed waiting for process: %v", err)
	}
	return nil
}
