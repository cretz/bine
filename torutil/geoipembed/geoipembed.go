// Package geoipembed contains embedded db files for GeoIP.
//
// The GeoIPReader can be used as tor.StartConf.GeoIPFileReader.
package geoipembed

import (
	"bytes"
	"io"
	"time"
)

// LastUpdated is the mod time of the embedded geoip files.
func LastUpdated() time.Time { return time.Unix(1537539535, 0) }

// GeoIPBytes returns the full byte slice of the geo IP file.
func GeoIPBytes(ipv6 bool) ([]byte, error) {
	if ipv6 {
		return geoip6Bytes()
	}
	return geoipBytes()
}

// GeoIPReader returns a ReadCloser for GeoIPBytes. Close does nothing.
func GeoIPReader(ipv6 bool) (io.ReadCloser, error) {
	if byts, err := GeoIPBytes(ipv6); err != nil {
		return nil, err
	} else {
		return &readNoopClose{bytes.NewReader(byts)}, nil
	}
}

type readNoopClose struct {
	*bytes.Reader
}

func (*readNoopClose) Close() error { return nil }
