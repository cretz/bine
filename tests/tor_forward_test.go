package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("forward response"))
	if err != nil {
		panic(err)
	}
}

func TestForwardSimpleHTTP(t *testing.T) {
	remotePorts := []int{80, 8080}

	// Create a test server listening on a random port
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	t.Cleanup(server.Close)

	ctx := GlobalEnabledNetworkContext(t)

	// Forward as an onion service on test ports
	conf := &tor.ForwardConf{
		Version3: true,
		PortForwards: map[string][]int{
			server.Listener.Addr().String(): remotePorts,
		},
	}
	httpClient := httpClient(ctx, nil)

	forwardCtx, forwardCancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer forwardCancel()
	// Create an onion service to listen on random port but show as 80
	onion, err := ctx.Forward(forwardCtx, conf)
	ctx.Require.NoError(err)

	// Check the service ID
	ctx.Require.Equal(torutil.OnionServiceIDFromPrivateKey(onion.Key), onion.ID)
	for _, remotePort := range remotePorts {
		// Request onion endpoint
		contents := httpGet(ctx, httpClient, fmt.Sprintf("http://"+onion.ID+".onion:%d/test", remotePort))
		ctx.Require.Equal("forward response", string(contents))
	}
}
