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

type MappedAddress struct {
	Old string
	New string
}

func NewMappedAddress(old string, new string) *MappedAddress {
	return &MappedAddress{Old: old, New: new}
}

func (c *Conn) MapAddresses(addresses []*MappedAddress) ([]*MappedAddress, error) {
	cmd := "MAPADDRESS"
	for _, address := range addresses {
		cmd += " " + address.New + "=" + address.Old
	}
	resp, err := c.SendRequest(cmd)
	if err != nil {
		return nil, err
	}
	data := resp.DataWithReply()
	ret := make([]*MappedAddress, 0, len(data))
	for _, address := range data {
		mappedAddress := &MappedAddress{}
		mappedAddress.Old, mappedAddress.New, _ = util.PartitionString(address, '=')
		ret = append(ret, mappedAddress)
	}
	return ret, nil
}

type InfoValue struct {
	Key   string
	Value string
}

func (c *Conn) GetInfo(keys ...string) ([]*InfoValue, error) {
	resp, err := c.SendRequest("GETCONF %v", strings.Join(keys, " "))
	if err != nil {
		return nil, err
	}
	ret := make([]*InfoValue, 0, len(resp.Data))
	for _, val := range resp.Data {
		infoVal := &InfoValue{}
		infoVal.Key, infoVal.Value, _ = util.PartitionString(val, '=')
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
