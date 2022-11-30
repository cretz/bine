package tor

import (
	"context"
	"crypto"
	"fmt"
	"net"
	"strconv"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/torutil/ed25519"
	othered25519 "golang.org/x/crypto/ed25519"
)

// OnionService implements net.Listener and net.Addr for an onion service.
type OnionService struct {
	// ID is the service ID for this onion service.
	ID string

	// Key is the private key for this service. It is either the set key, the
	// generated key, or nil if asked to discard the key. If present, it is an
	// instance of github.com/cretz/bine/torutil/ed25519.KeyPair.
	Key crypto.PrivateKey

	// LocalListener is the local TCP listener. This is always present.
	LocalListener net.Listener

	// RemotePorts is the set of remote ports that are forwarded to the local
	// listener. This will always have at least one value.
	RemotePorts []int

	// CloseLocalListenerOnClose is true if the local listener should be closed
	// on Close. This is set to true if a listener was created by Listen and set
	// to false of an existing LocalListener was provided to Listen.
	CloseLocalListenerOnClose bool

	// The Tor object that created this. Needed for Close.
	Tor *Tor
}

// ListenConf is the configuration for Listen calls.
type ListenConf struct {
	// LocalPort is the local port to create a TCP listener on. If the port is
	// 0, it is automatically chosen. This is ignored if LocalListener is set.
	LocalPort int

	// LocalListener is the specific local listener to back the onion service.
	// If this is nil (the default), then a listener is created with LocalPort.
	LocalListener net.Listener

	// RemotePorts are the remote ports to serve the onion service on. If empty,
	// it is the same as the local port or local listener. This must have at
	// least one value if the local listener is not a *net.TCPListener.
	RemotePorts []int

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

	// Version3

	Version3 bool
}

// Listen creates an onion service and local listener. The context can be nil.
// If conf is nil, the default struct value is used. Note, if this errors, any
// listeners created here are closed but if a LocalListener is provided it may remain open.
func (t *Tor) Listen(ctx context.Context, conf *ListenConf) (*OnionService, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Create the service up here and make sure we close it no matter the error within
	svc := &OnionService{Tor: t, CloseLocalListenerOnClose: conf.LocalListener == nil}
	var err error

	// Create the local listener if necessary
	svc.LocalListener = conf.LocalListener
	if svc.LocalListener == nil {
		if svc.LocalListener, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(conf.LocalPort)); err != nil {
			return nil, err
		}
	}

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
		svc.Key = key
		req.Key = &control.ED25519Key{key}
	case othered25519.PrivateKey:
		properKey := ed25519.FromCryptoPrivateKey(key)
		svc.Key = properKey
		req.Key = &control.ED25519Key{properKey}
	case *control.ED25519Key:
		svc.Key = key.KeyPair
		req.Key = key
	default:
		err = fmt.Errorf("Unrecognized key type: %T", key)
	}

	// Apply the remote ports
	if err == nil {
		if len(conf.RemotePorts) == 0 {
			tcpAddr, ok := svc.LocalListener.Addr().(*net.TCPAddr)
			if !ok {
				err = fmt.Errorf("Unable to derive local TCP port")
			} else {
				svc.RemotePorts = []int{tcpAddr.Port}
			}
		} else {
			svc.RemotePorts = make([]int, len(conf.RemotePorts))
			copy(svc.RemotePorts, conf.RemotePorts)
		}
	}
	// Apply the local ports with the remote ports
	if err == nil {
		localAddr := svc.LocalListener.Addr().String()
		if _, ok := svc.LocalListener.(*net.UnixListener); ok {
			localAddr = "unix:" + localAddr
		}
		for _, remotePort := range svc.RemotePorts {
			req.Ports = append(req.Ports, &control.KeyVal{Key: strconv.Itoa(remotePort), Val: localAddr})
		}
	}

	// Create the onion service
	var resp *control.AddOnionResponse
	if err == nil {
		resp, err = t.Control.AddOnion(req)
	}

	// Apply the response to the service
	if err == nil {
		svc.ID = resp.ServiceID
		switch key := resp.Key.(type) {
		case nil:
			// Do nothing
		case *control.ED25519Key:
			svc.Key = key.KeyPair
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
					if hs != nil && hs.Address == svc.ID {
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
		if closeErr := svc.Close(); closeErr != nil {
			err = fmt.Errorf("Error on listen: %v (also got error trying to close: %v)", err, closeErr)
		}
		return nil, err
	}
	return svc, nil
}

// Accept implements net.Listener.Accept.
func (o *OnionService) Accept() (net.Conn, error) {
	return o.LocalListener.Accept()
}

// Addr implements net.Listener.Addr just returning this object.
func (o *OnionService) Addr() net.Addr {
	return o
}

// Network implements net.Addr.Network always returning "tcp".
func (o *OnionService) Network() string {
	return "tcp"
}

// String implements net.Addr.String and returns "<serviceID>.onion:<virtport>".
func (o *OnionService) String() string {
	return fmt.Sprintf("%v.onion:%v", o.ID, o.RemotePorts[0])
}

// Close implements net.Listener.Close and deletes the onion service and closes
// the LocalListener if CloseLocalListenerOnClose is true.
func (o *OnionService) Close() (err error) {
	o.Tor.Debugf("Closing onion %v", o)
	// Delete the onion first
	if o.ID != "" {
		err = o.Tor.Control.DelOnion(o.ID)
		o.ID = ""
	}
	// Now if the local one needs to be closed, do it
	if o.CloseLocalListenerOnClose && o.LocalListener != nil {
		if closeErr := o.LocalListener.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("Unable to close onion: %v (also unable to close local listener: %v)", err, closeErr)
			} else {
				err = closeErr
			}
		}
		o.LocalListener = nil
	}
	if err != nil {
		o.Tor.Debugf("Failed closing onion: %v", err)
	}
	return
}
