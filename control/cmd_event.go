package control

import (
	"strings"
	"time"

	"github.com/cretz/bine/util"
)

type EventCode string

const (
	EventCodeAddrMap EventCode = "ADDRMAP"
	EventCodeCirc    EventCode = "CIRC"
)

func (c *Conn) AddEventListener(events []EventCode, ch chan<- Event) error {
	// TODO: do we want to set the local map first? Or do we want to lock on the net request too?
	c.eventListenersLock.Lock()
	for _, event := range events {
		// Must completely replace the array, never mutate it
		prevArr := c.eventListeners[event]
		newArr := make([]chan<- Event, len(prevArr)+1)
		copy(newArr, prevArr)
		newArr[len(newArr)-1] = ch
		c.eventListeners[event] = newArr
	}
	c.eventListenersLock.Unlock()
	return c.sendSetEvents()
}

func (c *Conn) RemoveEventListener(events []EventCode, ch chan<- Event) error {
	// TODO: do we want to mutate the local map first?
	c.eventListenersLock.Lock()
	for _, event := range events {
		arr := c.eventListeners[event]
		index := -1
		for i, listener := range arr {
			if listener == ch {
				index = i
				break
			}
		}
		if index != -1 {
			if len(arr) == 1 {
				delete(c.eventListeners, event)
			} else {
				// Must completely replace the array, never mutate it
				newArr := make([]chan<- Event, len(arr)-1)
				copy(newArr, arr[:index])
				copy(newArr[index:], arr[index+1:])
				c.eventListeners[event] = newArr
			}
		}
	}
	c.eventListenersLock.Unlock()
	return c.sendSetEvents()
}

func (c *Conn) sendSetEvents() error {
	c.eventListenersLock.RLock()
	cmd := "SETEVENTS"
	for event := range c.eventListeners {
		cmd += " " + string(event)
	}
	c.eventListenersLock.RUnlock()
	return c.sendRequestIgnoreResponse(cmd)
}

// zero on fail
func parseISOTime2Frac(str string) time.Time {
	// Essentially time.RFC3339Nano but without TZ info
	const layout = "2006-01-02T15:04:05.999999999"
	ret, err := time.Parse(layout, str)
	if err != nil {
		ret = time.Time{}
	}
	return ret
}

type CircuitEvent struct {
	CircuitID     string
	Status        string
	Path          []string
	BuildFlags    []string
	Purpose       string
	HSState       string
	RendQuery     string
	TimeCreated   time.Time
	Reason        string
	RemoteReason  string
	SocksUsername string
	SocksPassword string
	Raw           string
}

func ParseCircuitEvent(raw string) *CircuitEvent {
	event := &CircuitEvent{Raw: raw}
	event.CircuitID, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	event.Status, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	first := true
	for ok {
		if attr, raw, ok = util.PartitionString(raw, ' '); !ok {
			break
		}
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "BUILD_FLAGS":
			event.BuildFlags = strings.Split(val, ",")
		case "PURPOSE":
			event.Purpose = val
		case "HS_STATE":
			event.HSState = val
		case "REND_QUERY":
			event.RendQuery = val
		case "TIME_CREATED":
			event.TimeCreated = parseISOTime2Frac(val)
		case "REASON":
			event.Reason = val
		case "REMOTE_REASON":
			event.RemoteReason = val
		case "SOCKS_USERNAME":
			event.SocksUsername = val
		case "SOCKS_PASSWORD":
			event.SocksPassword = val
		default:
			if first {
				event.Path = strings.Split(val, ",")
			}
		}
		first = false
	}
	return event
}

type Event interface {
	Code() EventCode
}

func (*CircuitEvent) Code() EventCode { return EventCodeCirc }

func (c *Conn) relayAsyncEvents(resp *Response) {
	code, data, _ := util.PartitionString(resp.Reply, ' ')
	// Only relay if there are chans
	c.eventListenersLock.RLock()
	chans := c.eventListeners[EventCode(code)]
	c.eventListenersLock.RUnlock()
	if len(chans) == 0 {
		return
	}
	// Parse the event
	// TODO: more events
	var event Event
	switch EventCode(code) {
	case EventCodeCirc:
		event = ParseCircuitEvent(data)
	}
	if event != nil {
		for _, ch := range chans {
			// Just send, if closed or blocking, that's not our problem
			ch <- event
		}
	}
}
