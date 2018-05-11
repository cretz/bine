package control

import (
	"strings"

	"github.com/cretz/bine/util"
)

func (c *Conn) Signal(signal string) error {
	return c.sendRequestIgnoreResponse("SIGNAL %v", signal)
}

func (c *Conn) Quit() error {
	return c.sendRequestIgnoreResponse("QUIT")
}

func (c *Conn) MapAddresses(addresses ...*KeyVal) ([]*KeyVal, error) {
	cmd := "MAPADDRESS"
	for _, address := range addresses {
		cmd += " " + address.Key + "=" + address.Val
	}
	resp, err := c.SendRequest(cmd)
	if err != nil {
		return nil, err
	}
	data := resp.DataWithReply()
	ret := make([]*KeyVal, 0, len(data))
	for _, address := range data {
		mappedAddress := &KeyVal{}
		mappedAddress.Key, mappedAddress.Val, _ = util.PartitionString(address, '=')
		ret = append(ret, mappedAddress)
	}
	return ret, nil
}

func (c *Conn) GetInfo(keys ...string) ([]*KeyVal, error) {
	resp, err := c.SendRequest("GETINFO %v", strings.Join(keys, " "))
	if err != nil {
		return nil, err
	}
	ret := make([]*KeyVal, 0, len(resp.Data))
	for _, val := range resp.Data {
		infoVal := &KeyVal{}
		infoVal.Key, infoVal.Val, _ = util.PartitionString(val, '=')
		ret = append(ret, infoVal)
	}
	return ret, nil
}

func (c *Conn) PostDescriptor(descriptor string, purpose string, cache string) error {
	cmd := "+POSTDESCRIPTOR"
	if purpose != "" {
		cmd += " purpose=" + purpose
	}
	if cache != "" {
		cmd += " cache=" + cache
	}
	cmd += "\r\n" + descriptor + "\r\n."
	return c.sendRequestIgnoreResponse(cmd)
}

func (c *Conn) UseFeatures(features ...string) error {
	return c.sendRequestIgnoreResponse("USEFEATURE " + strings.Join(features, " "))
}

// TODO: can this take multiple
func (c *Conn) ResolveAsync(address string, reverse bool) error {
	cmd := "RESOLVE "
	if reverse {
		cmd += "mode=reverse "
	}
	return c.sendRequestIgnoreResponse(cmd + address)
}

func (c *Conn) TakeOwnership() error {
	return c.sendRequestIgnoreResponse("TAKEOWNERSHIP")
}

func (c *Conn) DropGuards() error {
	return c.sendRequestIgnoreResponse("DROPGUARDS")
}
