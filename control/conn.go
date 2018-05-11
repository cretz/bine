package control

import (
	"fmt"
	"io"
	"net/textproto"
	"sync"
)

type Conn struct {
	// No debug logs if nil
	DebugWriter io.Writer

	conn *textproto.Conn

	asyncChansLock sync.RWMutex
	// Can be traversed outside of lock, entire field is replaced on change
	asyncChans []chan<- *Response

	// Set lazily
	protocolInfo *ProtocolInfo

	Authenticated bool

	eventListenersLock sync.RWMutex
	// The value slices can be traversed outside of lock, they are completely replaced on change, never mutated
	eventListeners map[EventCode][]chan<- Event
}

func NewConn(conn *textproto.Conn) *Conn {
	return &Conn{
		conn:           conn,
		eventListeners: map[EventCode][]chan<- Event{},
	}
}

func (c *Conn) SendRequest(format string, args ...interface{}) (*Response, error) {
	if c.debugEnabled() {
		c.debugf("Write line: %v", fmt.Sprintf(format, args...))
	}
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

func (c *Conn) Quit() error {
	_, err := c.SendRequest("QUIT")
	return err
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
	c.Quit()
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
func (c *Conn) RemoveAsyncChan(ch chan<- *Response) bool {
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
	// First, relay events
	c.relayAsyncEvents(resp)
	c.asyncChansLock.RLock()
	chans := c.asyncChans
	c.asyncChansLock.RUnlock()
	// We will allow channels to block
	for _, ch := range chans {
		ch <- resp
	}
}

func (c *Conn) debugEnabled() bool {
	return c.DebugWriter != nil
}

func (c *Conn) debugf(format string, args ...interface{}) {
	if w := c.DebugWriter; w != nil {
		fmt.Fprintf(w, format+"\n", args...)
	}
}

func (*Conn) protoErr(format string, args ...interface{}) textproto.ProtocolError {
	return textproto.ProtocolError(fmt.Sprintf(format, args...))
}
