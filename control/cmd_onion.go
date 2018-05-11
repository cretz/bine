package control

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/cretz/bine/util"
	"golang.org/x/crypto/ed25519"
)

type KeyType string

const (
	KeyTypeNew       KeyType = "NEW"
	KeyTypeRSA1024   KeyType = "RSA1024"
	KeyTypeED25519V3 KeyType = "ED25519-V3"
)

type KeyAlgo string

const (
	KeyAlgoBest      KeyAlgo = "BEST"
	KeyAlgoRSA1024   KeyAlgo = "RSA1024"
	KeyAlgoED25519V3 KeyAlgo = "ED25519-V3"
)

type Key interface {
	Type() KeyType
	Blob() string
}

func KeyFromString(str string) (Key, error) {
	typ, blob, _ := util.PartitionString(str, ':')
	switch KeyType(typ) {
	case KeyTypeNew:
		return GenKeyFromBlob(blob), nil
	case KeyTypeRSA1024:
		return RSA1024KeyFromBlob(blob)
	case KeyTypeED25519V3:
		return ED25519KeyFromBlob(blob)
	default:
		return nil, fmt.Errorf("Unrecognized key type: %v", typ)
	}
}

type GenKey KeyAlgo

func GenKeyFromBlob(blob string) GenKey { return GenKey(KeyAlgo(blob)) }
func (GenKey) Type() KeyType            { return KeyTypeNew }
func (g GenKey) Blob() string           { return string(g) }

type RSAKey struct{ *rsa.PrivateKey }

func RSA1024KeyFromBlob(blob string) (*RSAKey, error) {
	byts, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return nil, err
	}
	rsaKey, err := x509.ParsePKCS1PrivateKey(byts)
	if err != nil {
		return nil, err
	}
	return &RSAKey{rsaKey}, nil
}
func (*RSAKey) Type() KeyType { return KeyTypeRSA1024 }
func (r *RSAKey) Blob() string {
	return base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(r.PrivateKey))
}

type ED25519Key ed25519.PrivateKey

func ED25519KeyFromBlob(blob string) (ED25519Key, error) {
	byts, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return nil, err
	}
	return ED25519Key(ed25519.PrivateKey(byts)), nil
}
func (ED25519Key) Type() KeyType  { return KeyTypeED25519V3 }
func (e ED25519Key) Blob() string { return base64.StdEncoding.EncodeToString(e) }

type AddOnionRequest struct {
	Key         Key
	Flags       []string
	MaxStreams  int
	Ports       map[string]string
	ClientAuths map[string]string
}

type AddOnionResponse struct {
	ServiceID   string
	Key         Key
	ClientAuths map[string]string
	RawResponse *Response
}

func (c *Conn) AddOnion(req *AddOnionRequest) (*AddOnionResponse, error) {
	// Build command
	if req.Key == nil {
		return nil, c.protoErr("Key required")
	}
	cmd := "ADDONION " + string(req.Key.Type()) + ":" + req.Key.Blob()
	if len(req.Flags) > 0 {
		cmd += " Flags=" + strings.Join(req.Flags, ",")
	}
	if req.MaxStreams > 0 {
		cmd += " MaxStreams=" + strconv.Itoa(req.MaxStreams)
	}
	for virt, target := range req.Ports {
		cmd += " Port=" + virt
		if target != "" {
			cmd += "," + target
		}
	}
	for name, blob := range req.ClientAuths {
		cmd += " ClientAuth=" + name
		if blob != "" {
			cmd += ":" + blob
		}
	}
	// Invoke and read response
	resp, err := c.SendRequest(cmd)
	if err != nil {
		return nil, err
	}
	ret := &AddOnionResponse{RawResponse: resp}
	for _, data := range resp.Data {
		key, val, _ := util.PartitionString(data, '=')
		switch key {
		case "ServiceID":
			ret.ServiceID = val
		case "PrivateKey":
			if ret.Key, err = KeyFromString(val); err != nil {
				return nil, err
			}
		case "ClientAuth":
			name, pass, _ := util.PartitionString(val, ':')
			if ret.ClientAuths == nil {
				ret.ClientAuths = map[string]string{}
			}
			ret.ClientAuths[name] = pass
		}
	}
	return ret, nil
}

func (c *Conn) DelOnion(serviceID string) error {
	return c.sendRequestIgnoreResponse("DELONION %v", serviceID)
}
