package tests

import (
	"testing"

	"github.com/cretz/bine/tor"
	"github.com/cretz/bine/torutil/geoipembed"
)

func TestEmbeddedGeoIPFile(t *testing.T) {
	ctx := NewTestContext(t, &tor.StartConf{GeoIPFileReader: geoipembed.GeoIPReader})
	defer ctx.Close()
	// Check available and grab a couple of known IPs and check the country
	// (taken from https://my.pingdom.com/probes/feed)
	usIpv4, usIpv6 := "209.58.139.193", "2605:fe80:2100:a00f:4::4045"
	kv, err := ctx.Control.GetInfo(
		"ip-to-country/ipv4-available",
		"ip-to-country/ipv6-available",
		"ip-to-country/"+usIpv4,
		"ip-to-country/"+usIpv6,
	)
	ctx.Require.NoError(err)
	vals := map[string]string{}
	for _, kv := range kv {
		vals[kv.Key] = kv.Val
	}
	ctx.Require.Len(vals, 4)
	ctx.Require.Equal("1", vals["ip-to-country/ipv4-available"])
	ctx.Require.Equal("1", vals["ip-to-country/ipv6-available"])
	ctx.Require.Equal("us", vals["ip-to-country/"+usIpv4])
	ctx.Require.Equal("us", vals["ip-to-country/"+usIpv6])
}
