package tor

import (
	"context"
	"crypto"
	"fmt"
	"strconv"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/torutil/ed25519"
	othered25519 "golang.org/x/crypto/ed25519"
)

// OnionForward describes a port forward to an onion service.
type OnionForward struct {
	// ID is the service ID for this onion service.
	ID string

	// Key is the private key for this service. It is either the set key, the
	// generated key, or nil if asked to discard the key. If present, it is an
	// instance of github.com/cretz/bine/torutil/ed25519.KeyPair.
	Key crypto.PrivateKey

	// PortForwards defines the ports that will be forwarded to the onion
	// service.
	PortForwards map[string][]int

	// The Tor object that created this. Needed for Close.
	Tor *Tor
}

// ForwardConf is the configuration for Forward calls.
type ForwardConf struct {
	// PortForwards defines the ports that will be forwarded to the onion
	// service.
	PortForwards map[string][]int

	// Key is the private key to use. If not present, a key is generated. If
	// present, it must be an instance of
	// github.com/cretz/bine/torutil/ed25519.KeyPair, a
	// golang.org/x/crypto/ed25519.PrivateKey, or a
	// github.com/cretz/bine/control.Key.
	Key crypto.PrivateKey

	// ClientAuths is the credential set for clients. The values are
	// base32-encoded ed25519 public keys.
	ClientAuths []string

	// MaxStreams is the maximum number of streams the service will accept. 0
	// means unlimited.
	MaxStreams int

	// DiscardKey, if true and Key is nil (meaning a private key is generated),
	// tells Tor not to return the generated private key. This value is ignored
	// if Key is not nil.
	DiscardKey bool

	// Detach, if true, prevents the default behavior of the onion service being
	// deleted when this controller connection is closed.
	Detach bool

	// NonAnonymous must be true if Tor options HiddenServiceSingleHopMode and
	// HiddenServiceNonAnonymousMode are set. Otherwise, it must be false.
	NonAnonymous bool

	// MaxStreamsCloseCircuit determines whether to close the circuit when the
	// maximum number of streams is exceeded. If true, the circuit is closed. If
	// false, the stream is simply not connected but the circuit stays open.
	MaxStreamsCloseCircuit bool

	// NoWait if true will not wait until the onion service is published. If
	// false, the network will be enabled if it's not and then we will wait
	// until the onion service is published.
	NoWait bool
}

// Forward creates an onion service which forwards to local ports. The context
// can be nil.  conf is required and cannot be nil.
func (t *Tor) Forward(ctx context.Context, conf *ForwardConf) (*OnionForward, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Create the forward up here and make sure we close it no matter the error within
	fwd := &OnionForward{Tor: t}
	var err error

	// Henceforth, any error requires we close the svc

	// Build the onion request
	req := &control.AddOnionRequest{MaxStreams: conf.MaxStreams, ClientAuths: conf.ClientAuths}
	// Set flags
	if conf.DiscardKey {
		req.Flags = append(req.Flags, "DiscardPK")
	}
	if conf.Detach {
		req.Flags = append(req.Flags, "Detach")
	}
	if len(conf.ClientAuths) > 0 {
		req.Flags = append(req.Flags, "V3Auth")
	}
	if conf.NonAnonymous {
		req.Flags = append(req.Flags, "NonAnonymous")
	}
	if conf.MaxStreamsCloseCircuit {
		req.Flags = append(req.Flags, "MaxStreamsCloseCircuit")
	}
	// Set the key
	switch key := conf.Key.(type) {
	case nil:
		req.Key = control.GenKey(control.KeyAlgoED25519V3)
	case control.GenKey:
		req.Key = key
	case ed25519.KeyPair:
		fwd.Key = key
		req.Key = &control.ED25519Key{key}
	case othered25519.PrivateKey:
		properKey := ed25519.FromCryptoPrivateKey(key)
		fwd.Key = properKey
		req.Key = &control.ED25519Key{properKey}
	case *control.ED25519Key:
		fwd.Key = key.KeyPair
		req.Key = key
	default:
		err = fmt.Errorf("Unrecognized key type: %T", key)
	}

	// Apply the remote ports
	fwd.PortForwards = conf.PortForwards
	for localPort, remotePorts := range fwd.PortForwards {
		if len(remotePorts) == 0 {
			continue
		}
		for _, remotePort := range remotePorts {
			req.Ports = append(req.Ports, &control.KeyVal{
				Key: strconv.Itoa(remotePort),
				Val: localPort,
			})
		}
	}

	// Create the onion service
	var resp *control.AddOnionResponse
	if err == nil {
		resp, err = t.Control.AddOnion(req)
	}

	// Apply the response to the service
	if err == nil {
		fwd.ID = resp.ServiceID
		switch key := resp.Key.(type) {
		case nil:
			// Do nothing
		case *control.ED25519Key:
			fwd.Key = key.KeyPair
		default:
			err = fmt.Errorf("Unrecognized result key type: %T", key)
		}
	}

	// Wait if necessary
	if err == nil && !conf.NoWait {
		t.Debugf("Enabling network before waiting for publication")
		// First make sure network is enabled
		if err = t.EnableNetwork(ctx, true); err == nil {
			t.Debugf("Waiting for publication")
			// Now we'll take a similar approach to Stem. Several UPLOADs are sent out, so we count em. If we see
			// UPLOADED, we succeeded. If we see failed, we count those. If there are as many failures as uploads, they
			// all failed and it's a failure. NOTE: unlike Stem's comments that say they don't, we are actually seeing
			// the service IDs for UPLOADED so we don't keep a map.
			uploadsAttempted := 0
			failures := []string{}
			_, err = t.Control.EventWait(ctx, []control.EventCode{control.EventCodeHSDesc},
				func(evt control.Event) (bool, error) {
					hs, _ := evt.(*control.HSDescEvent)
					if hs != nil && hs.Address == fwd.ID {
						switch hs.Action {
						case "UPLOAD":
							uploadsAttempted++
						case "FAILED":
							failures = append(failures,
								fmt.Sprintf("Failed uploading to dir %v - reason: %v", hs.HSDir, hs.Reason))
							if len(failures) == uploadsAttempted {
								return false, fmt.Errorf("Failed all uploads, reasons: %v", failures)
							}
						case "UPLOADED":
							return true, nil
						}
					}
					return false, nil
				})
		}
	}

	// Give back err and close if there is an err
	if err != nil {
		if closeErr := fwd.Close(); closeErr != nil {
			err = fmt.Errorf("Error on listen: %v (also got error trying to close: %v)", err, closeErr)
		}
		return nil, err
	}
	return fwd, nil
}

// String implements fmt.Stringer
func (o *OnionForward) String() string {
	return fmt.Sprintf("%v.onion", o.ID)
}

// Close deletes the onion service.
func (o *OnionForward) Close() (err error) {
	o.Tor.Debugf("Closing onion %v", o)
	// Delete the onion first
	if o.ID != "" {
		err = o.Tor.Control.DelOnion(o.ID)
		o.ID = ""
	}
	if err != nil {
		o.Tor.Debugf("Failed closing onion: %v", err)
	}
	return
}
