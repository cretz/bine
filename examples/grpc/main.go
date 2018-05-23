package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/cretz/bine/examples/grpc/pb"
	"github.com/cretz/bine/tor"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.Printf("Starting Tor")
	// We'll give it 5 minutes to run the whole thing (way too much of course, usually about 20 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	t, err := tor.Start(ctx, nil)
	if err != nil {
		return err
	}
	defer t.Close()

	log.Printf("Starting onion service, please wait")
	server, onionID, err := startServer(ctx, t)
	if err != nil {
		return err
	}
	defer server.Stop()
	log.Printf("Onion service available at %v.onion", onionID)

	log.Printf("Connecting to onion service")
	conn, client, err := startClient(ctx, t, onionID+".onion:80")
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("Doing simple RPC")
	resp, err := client.JoinStrings(ctx, &pb.JoinStringsRequest{Strings: []string{"foo", "bar", "baz"}, Delimiter: "-"})
	if err != nil {
		return err
	} else if resp.Joined != "foo-bar-baz" {
		return fmt.Errorf("Invalid response: %v", resp.Joined)
	}

	log.Printf("Doing server-side streaming RPC")
	pStream, err := client.ProvideStrings(ctx, &pb.ProvideStringsRequest{Count: 10})
	if err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		if resp, err := pStream.Recv(); err != nil {
			return err
		} else if resp.String_ != fmt.Sprintf("string-%v", i+1) {
			return fmt.Errorf("Invalid response: %v", resp.String_)
		}
	}
	if _, err = pStream.Recv(); err != io.EOF {
		return fmt.Errorf("Expected EOF, got %v", err)
	}

	log.Printf("Doing client-side streaming RPC")
	rStream, err := client.ReceiveStrings(ctx)
	strs := []string{"foo", "bar", "baz"}
	for _, str := range strs {
		if err := rStream.Send(&pb.ReceiveStringsRequest{String_: str}); err != nil {
			return err
		}
	}
	if resp, err := rStream.CloseAndRecv(); err != nil {
		return err
	} else if !reflect.DeepEqual(resp.Received, strs) {
		return fmt.Errorf("Unexpected response: %v", resp.Received)
	}

	log.Printf("Doing bi-directional streaming RPC")
	eStream, err := client.ExchangeStrings(ctx)
	for _, str := range strs {
		if err := eStream.Send(&pb.ExchangeStringsRequest{String_: str, WantReturn: str == "baz"}); err != nil {
			return err
		}
	}
	if resp, err := eStream.Recv(); err != nil {
		return err
	} else if !reflect.DeepEqual(resp.Received, strs) {
		return fmt.Errorf("Unexpected response: %v", resp.Received)
	}
	err = eStream.Send(&pb.ExchangeStringsRequest{String_: "one"})
	if err == nil {
		err = eStream.Send(&pb.ExchangeStringsRequest{String_: "two", WantReturn: true})
	}
	if err == nil {
		err = eStream.CloseSend()
	}
	if err != nil {
		return err
	}
	if resp, err := eStream.Recv(); err != nil {
		return err
	} else if !reflect.DeepEqual(resp.Received, []string{"one", "two"}) {
		return fmt.Errorf("Unexpected response: %v", resp.Received)
	}
	if _, err = eStream.Recv(); err != io.EOF {
		return fmt.Errorf("Expected EOF, got %v", err)
	}

	log.Printf("All completed successfully, shutting down")
	return nil
}

func startServer(ctx context.Context, t *tor.Tor) (server *grpc.Server, onionID string, err error) {
	// Wait at most a few minutes to publish the service
	listenCtx, listenCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer listenCancel()
	// Create an onion service to listen on a random local port but show as 80
	// We'll do version 3 since it's quicker
	onion, err := t.Listen(listenCtx, &tor.ListenConf{Version3: true, RemotePorts: []int{80}})
	if err != nil {
		return nil, "", err
	}
	onionID = onion.ID
	// Create the grpc server and start it
	server = grpc.NewServer()
	pb.RegisterSimpleServiceServer(server, simpleService{})
	go func() {
		if err := server.Serve(onion); err != nil {
			log.Printf("Error serving: %v", err)
		}
	}()
	return
}

func startClient(
	ctx context.Context, t *tor.Tor, addr string,
) (conn *grpc.ClientConn, client pb.SimpleServiceClient, err error) {
	// Wait at most a few minutes to connect to the service
	connCtx, connCancel := context.WithTimeout(ctx, 3*time.Minute)
	defer connCancel()
	// Make the dialer
	dialer, err := t.Dialer(connCtx, nil)
	if err != nil {
		return nil, nil, err
	}
	// Make the connection
	conn, err = grpc.DialContext(connCtx, addr,
		grpc.FailOnNonTempDialError(true),
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			dialCtx, dialCancel := context.WithTimeout(ctx, timeout)
			defer dialCancel()
			return dialer.DialContext(dialCtx, "tcp", addr)
		}),
	)
	if err == nil {
		client = pb.NewSimpleServiceClient(conn)
	}
	return
}

type simpleService struct{}

func (simpleService) JoinStrings(ctx context.Context, req *pb.JoinStringsRequest) (*pb.JoinStringsResponse, error) {
	return &pb.JoinStringsResponse{Joined: strings.Join(req.Strings, req.Delimiter)}, nil
}

func (simpleService) ProvideStrings(req *pb.ProvideStringsRequest, srv pb.SimpleService_ProvideStringsServer) error {
	for i := 0; uint32(i) < req.Count; i++ {
		if err := srv.Send(&pb.ProvideStringsResponse{String_: fmt.Sprintf("string-%v", i+1)}); err != nil {
			return err
		}
	}
	return nil
}

func (simpleService) ReceiveStrings(srv pb.SimpleService_ReceiveStringsServer) error {
	resp := &pb.ReceiveStringsResponse{}
	for {
		if req, err := srv.Recv(); err == io.EOF {
			return srv.SendAndClose(resp)
		} else if err != nil {
			return err
		} else {
			resp.Received = append(resp.Received, req.String_)
		}
	}
}

func (simpleService) ExchangeStrings(srv pb.SimpleService_ExchangeStringsServer) error {
	resp := &pb.ExchangeStringsResponse{}
	for {
		if req, err := srv.Recv(); err == io.EOF {
			return srv.Send(resp)
		} else if err != nil {
			return err
		} else {
			resp.Received = append(resp.Received, req.String_)
			if req.WantReturn {
				if err := srv.Send(resp); err != nil {
					return err
				}
				resp = &pb.ExchangeStringsResponse{}
			}
		}
	}
}
