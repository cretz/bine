package control

import (
	"strings"

	"github.com/cretz/bine/util"
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

func (c *Conn) ProtocolInfo() (*ProtocolInfo, error) {
	var err error
	if c.protocolInfo == nil {
		c.protocolInfo, err = c.sendProtocolInfo()
	}
	return c.protocolInfo, err
}

func (c *Conn) sendProtocolInfo() (*ProtocolInfo, error) {
	resp, err := c.SendRequest("PROTOCOLINFO")
	if err != nil {
		return nil, err
	}
	// Check data vals
	ret := &ProtocolInfo{RawResponse: resp}
	for _, piece := range resp.Data {
		key, val, ok := util.PartitionString(piece, ' ')
		if !ok {
			continue
		}
		switch key {
		case "PROTOCOLINFO":
			if val != "1" {
				return nil, c.protoErr("Invalid PIVERSION: %v", val)
			}
		case "AUTH":
			methods, cookieFile, _ := util.PartitionString(val, ' ')
			if !strings.HasPrefix(methods, "METHODS=") {
				continue
			}
			if cookieFile != "" {
				if !strings.HasPrefix(cookieFile, "COOKIEFILE=") {
					continue
				}
				if ret.CookieFile, err = util.UnescapeSimpleQuotedString(cookieFile[11:]); err != nil {
					continue
				}
			}
			ret.AuthMethods = strings.Split(methods[8:], ",")
		case "VERSION":
			torVersion, _, _ := util.PartitionString(val, ' ')
			if strings.HasPrefix(torVersion, "Tor=") {
				ret.TorVersion, err = util.UnescapeSimpleQuotedString(torVersion[4:])
			}
		}
	}
	return ret, nil
}
