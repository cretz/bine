package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil"
)

func TestListenSimpleHTTPV2(t *testing.T) {
	ctx := GlobalEnabledNetworkContext(t)
	// Create an onion service to listen on random port but show as 80
	conf := &tor.ListenConf{RemotePorts: []int{80}}
	client, server, onion := startHTTPServer(ctx, conf, "/test", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test Content"))
		ctx.Require.NoError(err)
	})
	// Check the service ID
	ctx.Require.Equal(torutil.OnionServiceIDFromPrivateKey(onion.Key), onion.ID)
	defer server.Shutdown(ctx)
	// Call /test
	byts := httpGet(ctx, client, "http://"+onion.ID+".onion/test")
	ctx.Require.Equal("Test Content", string(byts))
}

func TestListenSimpleHTTPV3(t *testing.T) {
	ctx := GlobalEnabledNetworkContext(t)
	// Create an onion service to listen on random port but show as 80
	conf := &tor.ListenConf{RemotePorts: []int{80}, Version3: true}
	// _, conf.Key, _ = ed25519.GenerateKey(nil)
	client, server, onion := startHTTPServer(ctx, conf, "/test", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test Content"))
		ctx.Require.NoError(err)
	})
	defer server.Shutdown(ctx)
	// Check the service ID
	ctx.Require.Equal(torutil.OnionServiceIDFromPrivateKey(onion.Key), onion.ID)
	// Call /test
	byts := httpGet(ctx, client, "http://"+onion.ID+".onion/test")
	ctx.Require.Equal("Test Content", string(byts))
}

// Only have to shutdown the HTTP server
func startHTTPServer(
	ctx *TestContext,
	listenConf *tor.ListenConf,
	handlePattern string,
	handler func(http.ResponseWriter, *http.Request),
) (*http.Client, *http.Server, *tor.OnionService) {
	httpClient := httpClient(ctx)
	// Wait at most a few minutes for the entire test
	listenCtx, listenCancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer listenCancel()
	// Create an onion service to listen on random port but show as 80
	onion, err := ctx.Listen(listenCtx, listenConf)
	ctx.Require.NoError(err)
	// Make HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc(handlePattern, handler)
	httpServer := &http.Server{Handler: mux}
	go func() { ctx.Require.Equal(http.ErrServerClosed, httpServer.Serve(onion)) }()
	return httpClient, httpServer, onion
}
