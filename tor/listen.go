package tor

import "net"

type OnionConf struct {
	Port       int
	TargetPort int
}

func (t *Tor) Listen(conf *OnionConf) (net.Listener, error) {
	panic("TODO")
}
