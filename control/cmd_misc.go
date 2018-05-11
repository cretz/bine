package control

import "github.com/cretz/bine/util"

func (c *Conn) Signal(signal string) error {
	_, err := c.SendRequest("SIGNAL %v", signal)
	return err
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
