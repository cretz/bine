package control

import (
	"strings"
)

type ProtocolInfo struct {
	AuthMethods []string
	CookieFile  string
	TorVersion  string
	RawResponse *Response
}

func (p *ProtocolInfo) HasAuthMethod(authMethod string) bool {
	for _, m := range p.AuthMethods {
		if m == authMethod {
			return true
		}
	}
	return false
}

func (c *Conn) RequestProtocolInfo() (*ProtocolInfo, error) {
	resp, err := c.SendRequest("PROTOCOLINFO")
	if err != nil {
		return nil, err
	}
	// Check PIVERSION
	if len(resp.Data) == 0 || resp.Data[0] != "1" {
		return nil, newProtocolError("Invalid PIVERSION: %s", resp.Reply)
	}
	// Get other response vals
	ret := &ProtocolInfo{RawResponse: resp}
	for _, piece := range resp.Data {
		key, val, ok := partitionString(piece, ' ')
		if !ok {
			continue
		}
		switch key {
		case "AUTH":
			methods, cookieFile, _ := partitionString(val, ' ')
			if !strings.HasPrefix(methods, "METHODS=") {
				continue
			}
			if cookieFile != "" {
				if !strings.HasPrefix(cookieFile, "COOKIEFILE=") {
					continue
				}
				if ret.CookieFile, err = parseQuotedString(cookieFile[11:]); err != nil {
					continue
				}
			}
			ret.AuthMethods = strings.Split(methods[8:], ",")
		case "VERSION":
			torVersion, _, _ := partitionString(val, ' ')
			if strings.HasPrefix(torVersion, "Tor=") {
				ret.TorVersion, _ = parseQuotedString(torVersion[4:])
			}
		}
	}
	return ret, nil
}
