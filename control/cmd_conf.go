package control

import (
	"strings"

	"github.com/cretz/bine/util"
)

type ConfEntry struct {
	Key   string
	Value *string
}

func NewConfEntry(key string, value *string) *ConfEntry {
	return &ConfEntry{Key: key, Value: value}
}

func (c *Conn) SetConf(entries []*ConfEntry) error {
	return c.sendSetConf("SETCONF", entries)
}

func (c *Conn) ResetConf(entries []*ConfEntry) error {
	return c.sendSetConf("RESETCONF", entries)
}

func (c *Conn) sendSetConf(cmd string, entries []*ConfEntry) error {
	for _, entry := range entries {
		cmd += " " + entry.Key
		if entry.Value != nil {
			cmd += "=" + util.EscapeSimpleQuotedStringIfNeeded(*entry.Value)
		}
	}
	_, err := c.SendRequest(cmd)
	return err
}

func (c *Conn) GetConf(keys ...string) ([]*ConfEntry, error) {
	resp, err := c.SendRequest("GETCONF %v", strings.Join(keys, " "))
	if err != nil {
		return nil, err
	}
	data := resp.DataWithReply()
	ret := make([]*ConfEntry, 0, len(data))
	for _, data := range data {
		key, val, ok := util.PartitionString(data, '=')
		entry := &ConfEntry{Key: key}
		if ok {
			if val, err = util.UnescapeSimpleQuotedStringIfNeeded(val); err != nil {
				return nil, err
			}
			entry.Value = &val
		}
		ret = append(ret, entry)
	}
	return ret, nil
}

func (c *Conn) SaveConf(force bool) error {
	cmd := "SAVECONF"
	if force {
		cmd += " FORCE"
	}
	_, err := c.SendRequest(cmd)
	return err
}
