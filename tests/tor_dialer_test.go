package tests

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/context/ctxhttp"
)

func TestDialerSimpleHTTP(t *testing.T) {
	ctx := GlobalEnabledNetworkContext(t)
	httpClient := httpClient(ctx)
	// IsTor check
	byts := httpGet(ctx, httpClient, "https://check.torproject.org/api/ip")
	jsn := map[string]interface{}{}
	ctx.Require.NoError(json.Unmarshal(byts, &jsn))
	ctx.Require.True(jsn["IsTor"].(bool))
}

func httpClient(ctx *TestContext) *http.Client {
	// 15 seconds max to dial
	dialCtx, dialCancel := context.WithTimeout(ctx, 15*time.Second)
	defer dialCancel()
	// Make connection
	dialer, err := ctx.Dialer(dialCtx, nil)
	ctx.Require.NoError(err)
	return &http.Client{Transport: &http.Transport{DialContext: dialer.DialContext}}
}

func httpGet(ctx *TestContext, client *http.Client, url string) []byte {
	// We'll give it 30 seconds to respond
	callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
	defer callCancel()
	resp, err := ctxhttp.Get(callCtx, client, url)
	ctx.Require.NoError(err)
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	ctx.Require.NoError(err)
	return respBytes
}
