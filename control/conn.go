package control

import (
	"fmt"
	"net/textproto"
	"sync"
)

type Conn struct {
	conn *textproto.Conn

	asyncChansLock sync.RWMutex
	// Never mutated outside of lock, always created anew
	asyncChans []chan<- *Response
}

func NewConn(conn *textproto.Conn) *Conn { return &Conn{conn: conn} }

func (c *Conn) SendSignal(signal string) error {
	_, err := c.SendRequest("SIGNAL %v", signal)
	return err
}

func (c *Conn) SendRequest(format string, args ...interface{}) (*Response, error) {
	id, err := c.conn.Cmd(format, args...)
	if err != nil {
		return nil, err
	}
	c.conn.StartResponse(id)
	defer c.conn.EndResponse(id)
	// Get the first non-async response
	var resp *Response
	for {
		if resp, err = c.ReadResponse(); err != nil || !resp.IsAsync() {
			break
		}
		c.onAsyncResponse(resp)
	}
	if err == nil && !resp.IsOk() {
		err = resp.Err
	}
	return resp, err
}

func (c *Conn) Close() error {
	// We'll close all the chans first
	c.asyncChansLock.Lock()
	for _, ch := range c.asyncChans {
		close(ch)
	}
	c.asyncChans = nil
	c.asyncChansLock.Unlock()
	// Ignore the response and ignore the error
	c.SendRequest("QUIT")
	return c.conn.Close()
}

func (c *Conn) AddAsyncChan(ch chan<- *Response) {
	c.asyncChansLock.Lock()
	chans := make([]chan<- *Response, len(c.asyncChans)+1)
	copy(chans, c.asyncChans)
	chans[len(chans)-1] = ch
	c.asyncChans = chans
	c.asyncChansLock.Unlock()
}

// Does not close
func (c *Conn) RemoveChan(ch chan<- *Response) bool {
	c.asyncChansLock.Lock()
	chans := make([]chan<- *Response, len(c.asyncChans)+1)
	copy(chans, c.asyncChans)
	index := -1
	for i, existing := range chans {
		if existing == ch {
			index = i
			break
		}
	}
	if index != -1 {
		chans = append(chans[:index], chans[index+1:]...)
	}
	c.asyncChans = chans
	c.asyncChansLock.Unlock()
	return index != -1
}

func (c *Conn) onAsyncResponse(resp *Response) {
	c.asyncChansLock.RLock()
	chans := c.asyncChans
	c.asyncChansLock.RUnlock()
	// We will allow channels to block
	for _, ch := range chans {
		ch <- resp
	}
}

func newProtocolError(format string, args ...interface{}) textproto.ProtocolError {
	return textproto.ProtocolError(fmt.Sprintf(format, args...))
}
