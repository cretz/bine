package control

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cretz/bine/util"
)

// EventCode represents an asynchronous event code (ref control spec 4.1).
type EventCode string

const (
	// EventCodeAddrMap is ADDRMAP
	EventCodeAddrMap           EventCode = "ADDRMAP"
	EventCodeBandwidth         EventCode = "BW"
	EventCodeBuildTimeoutSet   EventCode = "BUILDTIMEOUT_SET"
	EventCodeCellStats         EventCode = "CELL_STATS"
	EventCodeCircuit           EventCode = "CIRC"
	EventCodeCircuitBandwidth  EventCode = "CIRC_BW"
	EventCodeCircuitMinor      EventCode = "CIRC_MINOR"
	EventCodeClientsSeen       EventCode = "CLIENTS_SEEN"
	EventCodeConfChanged       EventCode = "CONF_CHANGED"
	EventCodeConnBandwidth     EventCode = "CONN_BW"
	EventCodeDescChanged       EventCode = "DESCCHANGED"
	EventCodeGuard             EventCode = "GUARD"
	EventCodeHSDesc            EventCode = "HS_DESC"
	EventCodeHSDescContent     EventCode = "HS_DESC_CONTENT"
	EventCodeLogDebug          EventCode = "DEBUG"
	EventCodeLogErr            EventCode = "ERR"
	EventCodeLogInfo           EventCode = "INFO"
	EventCodeLogNotice         EventCode = "NOTICE"
	EventCodeLogWarn           EventCode = "WARN"
	EventCodeNetworkLiveness   EventCode = "NETWORK_LIVENESS"
	EventCodeNetworkStatus     EventCode = "NS"
	EventCodeNewConsensus      EventCode = "NEWCONSENSUS"
	EventCodeNewDesc           EventCode = "NEWDESC"
	EventCodeORConn            EventCode = "ORCONN"
	EventCodeSignal            EventCode = "SIGNAL"
	EventCodeStatusClient      EventCode = "STATUS_CLIENT"
	EventCodeStatusGeneral     EventCode = "STATUS_GENERAL"
	EventCodeStatusServer      EventCode = "STATUS_SERVER"
	EventCodeStream            EventCode = "STREAM"
	EventCodeStreamBandwidth   EventCode = "STREAM_BW"
	EventCodeTokenBucketEmpty  EventCode = "TB_EMPTY"
	EventCodeTransportLaunched EventCode = "TRANSPORT_LAUNCHED"
)

func EventCodes() []EventCode {
	return []EventCode{
		EventCodeAddrMap,
		EventCodeBandwidth,
		EventCodeBuildTimeoutSet,
		EventCodeCellStats,
		EventCodeCircuit,
		EventCodeCircuitBandwidth,
		EventCodeCircuitMinor,
		EventCodeClientsSeen,
		EventCodeConfChanged,
		EventCodeConnBandwidth,
		EventCodeDescChanged,
		EventCodeGuard,
		EventCodeHSDesc,
		EventCodeHSDescContent,
		EventCodeLogDebug,
		EventCodeLogErr,
		EventCodeLogInfo,
		EventCodeLogNotice,
		EventCodeLogWarn,
		EventCodeNetworkLiveness,
		EventCodeNetworkStatus,
		EventCodeNewConsensus,
		EventCodeNewDesc,
		EventCodeORConn,
		EventCodeSignal,
		EventCodeStatusClient,
		EventCodeStatusGeneral,
		EventCodeStatusServer,
		EventCodeStream,
		EventCodeStreamBandwidth,
		EventCodeTokenBucketEmpty,
		EventCodeTransportLaunched,
	}
}

// HandleEvents loops until the context is closed dispatching async events. Can dispatch events even after context is
// done and of course during synchronous request. This will always end with an error, either from ctx.Done() or from an
// error reading/handling the event.
func (c *Conn) HandleEvents(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		for ctx.Err() == nil {
			if err := c.HandleNextEvent(); err != nil {
				errCh <- err
				break
			}
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HandleNextEvent attempts to read and handle the next event. It will return on first message seen, event or not.
// Otherwise it will wait until there is a message read.
func (c *Conn) HandleNextEvent() error {
	c.readLock.Lock()
	defer c.readLock.Unlock()
	// We'll just peek for the next 3 bytes and see if they are async
	byts, err := c.conn.R.Peek(3)
	if err != nil {
		return err
	}
	statusCode, err := strconv.Atoi(string(byts))
	if err != nil || statusCode != StatusAsyncEvent {
		return err
	}
	// Read the entire thing and handle it
	resp, err := c.ReadResponse()
	if err != nil {
		return err
	}
	c.onAsyncResponse(resp)
	return nil
}

func (c *Conn) AddEventListener(ch chan<- Event, events ...EventCode) error {
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

func (c *Conn) RemoveEventListener(ch chan<- Event, events ...EventCode) error {
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

func (c *Conn) relayAsyncEvents(resp *Response) {
	var code, data string
	var dataArray []string
	if len(resp.Data) == 1 {
		// If there is a single line of data, first line of it is the code, rest of the first line is data
		firstNewline := strings.Index(resp.Data[0], "\r\n")
		if firstNewline == -1 {
			return
		}
		code, data = resp.Data[0][:firstNewline], resp.Data[0][firstNewline+2:]
	} else if len(resp.Data) > 0 {
		// If there are multiple lines, the entire first line is the code
		code, dataArray = resp.Data[0], resp.Data[1:]
	} else {
		// Otherwise, the reply line has the data
		code, data, _ = util.PartitionString(resp.Reply, ' ')
	}
	// Only relay if there are chans
	c.eventListenersLock.RLock()
	chans := c.eventListeners[EventCode(code)]
	c.eventListenersLock.RUnlock()
	if len(chans) == 0 {
		return
	}
	// Parse the event and only send if known event
	if event := ParseEvent(EventCode(code), data, dataArray); event != nil {
		for _, ch := range chans {
			// Just send, if closed or blocking, that's not our problem
			ch <- event
		}
	}
}

// zero on fail
func parseISOTime(str string) time.Time {
	// Essentially time.RFC3339 but without 'T' or TZ info
	const layout = "2006-01-02 15:04:05"
	ret, err := time.Parse(layout, str)
	if err != nil {
		ret = time.Time{}
	}
	return ret
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

type Event interface {
	Code() EventCode
}

func ParseEvent(code EventCode, raw string, dataArray []string) Event {
	switch code {
	case EventCodeAddrMap:
		return ParseAddrMapEvent(raw)
	case EventCodeBandwidth:
		return ParseBandwidthEvent(raw)
	case EventCodeBuildTimeoutSet:
		return ParseBuildTimeoutSetEvent(raw)
	case EventCodeCellStats:
		return ParseCellStatsEvent(raw)
	case EventCodeCircuit:
		return ParseCircuitEvent(raw)
	case EventCodeCircuitBandwidth:
		return ParseCircuitBandwidthEvent(raw)
	case EventCodeCircuitMinor:
		return ParseCircuitMinorEvent(raw)
	case EventCodeClientsSeen:
		return ParseClientsSeenEvent(raw)
	case EventCodeConfChanged:
		return ParseConfChangedEvent(dataArray)
	case EventCodeConnBandwidth:
		return ParseConnBandwidthEvent(raw)
	case EventCodeDescChanged:
		return ParseDescChangedEvent(raw)
	case EventCodeGuard:
		return ParseGuardEvent(raw)
	case EventCodeHSDesc:
		return ParseHSDescEvent(raw)
	case EventCodeHSDescContent:
		return ParseHSDescContentEvent(raw)
	case EventCodeLogDebug, EventCodeLogErr, EventCodeLogInfo, EventCodeLogNotice, EventCodeLogWarn:
		return ParseLogEvent(code, raw)
	case EventCodeNetworkLiveness:
		return ParseNetworkLivenessEvent(raw)
	case EventCodeNetworkStatus:
		return ParseNetworkStatusEvent(raw)
	case EventCodeNewConsensus:
		return ParseNewConsensusEvent(raw)
	case EventCodeNewDesc:
		return ParseNewDescEvent(raw)
	case EventCodeORConn:
		return ParseORConnEvent(raw)
	case EventCodeSignal:
		return ParseSignalEvent(raw)
	case EventCodeStatusClient, EventCodeStatusGeneral, EventCodeStatusServer:
		return ParseStatusEvent(code, raw)
	case EventCodeStream:
		return ParseStreamEvent(raw)
	case EventCodeStreamBandwidth:
		return ParseStreamBandwidthEvent(raw)
	case EventCodeTokenBucketEmpty:
		return ParseTokenBucketEmptyEvent(raw)
	case EventCodeTransportLaunched:
		return ParseTransportLaunchedEvent(raw)
	default:
		return nil
	}
}

type CircuitEvent struct {
	Raw           string
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
}

func ParseCircuitEvent(raw string) *CircuitEvent {
	event := &CircuitEvent{Raw: raw}
	event.CircuitID, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	event.Status, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	first := true
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
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

func (*CircuitEvent) Code() EventCode { return EventCodeCircuit }

type StreamEvent struct {
	Raw           string
	StreamID      string
	Status        string
	CircuitID     string
	TargetAddress string
	TargetPort    int
	Reason        string
	RemoteReason  string
	Source        string
	SourceAddress string
	SourcePort    int
	Purpose       string
}

func ParseStreamEvent(raw string) *StreamEvent {
	event := &StreamEvent{Raw: raw}
	event.StreamID, raw, _ = util.PartitionString(raw, ' ')
	event.Status, raw, _ = util.PartitionString(raw, ' ')
	event.CircuitID, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	event.TargetAddress, raw, ok = util.PartitionString(raw, ' ')
	if target, port, hasPort := util.PartitionStringFromEnd(event.TargetAddress, ':'); hasPort {
		event.TargetAddress = target
		event.TargetPort, _ = strconv.Atoi(port)
	}
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "REASON":
			event.Reason = val
		case "REMOTE_REASON":
			event.RemoteReason = val
		case "SOURCE":
			event.Source = val
		case "SOURCE_ADDR":
			event.SourceAddress = val
			if source, port, hasPort := util.PartitionStringFromEnd(event.SourceAddress, ':'); hasPort {
				event.SourceAddress = source
				event.SourcePort, _ = strconv.Atoi(port)
			}
		case "PURPOSE":
			event.Purpose = val
		}
	}
	return event
}

func (*StreamEvent) Code() EventCode { return EventCodeStream }

type ORConnEvent struct {
	Raw         string
	Target      string
	Status      string
	Reason      string
	NumCircuits int
	ConnID      string
}

func ParseORConnEvent(raw string) *ORConnEvent {
	event := &ORConnEvent{Raw: raw}
	event.Target, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	event.Status, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "REASON":
			event.Reason = val
		case "NCIRCS":
			event.NumCircuits, _ = strconv.Atoi(val)
		case "ID":
			event.ConnID = val
		}
	}
	return event
}

func (*ORConnEvent) Code() EventCode { return EventCodeORConn }

type BandwidthEvent struct {
	Raw          string
	BytesRead    int64
	BytesWritten int64
}

func ParseBandwidthEvent(raw string) *BandwidthEvent {
	event := &BandwidthEvent{Raw: raw}
	var temp string
	temp, raw, _ = util.PartitionString(raw, ' ')
	event.BytesRead, _ = strconv.ParseInt(temp, 10, 64)
	temp, raw, _ = util.PartitionString(raw, ' ')
	event.BytesWritten, _ = strconv.ParseInt(temp, 10, 64)
	return event
}

func (*BandwidthEvent) Code() EventCode { return EventCodeBandwidth }

type LogEvent struct {
	Severity EventCode
	Raw      string
}

func ParseLogEvent(severity EventCode, raw string) *LogEvent {
	return &LogEvent{Severity: severity, Raw: raw}
}

func (l *LogEvent) Code() EventCode { return l.Severity }

type NewDescEvent struct {
	Raw   string
	Descs []string
}

func ParseNewDescEvent(raw string) *NewDescEvent {
	return &NewDescEvent{Raw: raw, Descs: strings.Split(raw, " ")}
}

func (*NewDescEvent) Code() EventCode { return EventCodeNewDesc }

type AddrMapEvent struct {
	Raw        string
	Address    string
	NewAddress string
	ErrorCode  string
	// Zero if no expire
	Expires time.Time
	// Sans double quotes
	Cached string
}

func ParseAddrMapEvent(raw string) *AddrMapEvent {
	event := &AddrMapEvent{Raw: raw}
	event.Address, raw, _ = util.PartitionString(raw, ' ')
	event.NewAddress, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	// Skip local expiration, use UTC one later
	_, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "error":
			event.ErrorCode = val
		case "EXPIRES":
			val, _ = util.UnescapeSimpleQuotedString(val)
			event.Expires = parseISOTime(val)
		case "CACHED":
			event.Cached, _ = util.UnescapeSimpleQuotedStringIfNeeded(val)
		}
	}
	return event
}

func (*AddrMapEvent) Code() EventCode { return EventCodeAddrMap }

type DescChangedEvent struct {
	Raw string
}

func ParseDescChangedEvent(raw string) *DescChangedEvent {
	return &DescChangedEvent{Raw: raw}
}

func (*DescChangedEvent) Code() EventCode { return EventCodeDescChanged }

type StatusEvent struct {
	Raw       string
	Type      EventCode
	Severity  string
	Action    string
	Arguments map[string]string
}

func ParseStatusEvent(typ EventCode, raw string) *StatusEvent {
	event := &StatusEvent{Raw: raw, Type: typ, Arguments: map[string]string{}}
	event.Severity, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	event.Action, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		event.Arguments[key], _ = util.UnescapeSimpleQuotedStringIfNeeded(val)
	}
	return event
}

func (s *StatusEvent) Code() EventCode { return s.Type }

type GuardEvent struct {
	Raw    string
	Type   string
	Name   string
	Status string
}

func ParseGuardEvent(raw string) *GuardEvent {
	event := &GuardEvent{Raw: raw}
	event.Type, raw, _ = util.PartitionString(raw, ' ')
	event.Name, raw, _ = util.PartitionString(raw, ' ')
	event.Status, raw, _ = util.PartitionString(raw, ' ')
	return event
}

func (*GuardEvent) Code() EventCode { return EventCodeGuard }

type NetworkStatusEvent struct {
	Raw string
}

func ParseNetworkStatusEvent(raw string) *NetworkStatusEvent {
	return &NetworkStatusEvent{Raw: raw}
}

func (*NetworkStatusEvent) Code() EventCode { return EventCodeNetworkStatus }

type StreamBandwidthEvent struct {
	Raw          string
	BytesRead    int64
	BytesWritten int64
	Time         time.Time
}

func ParseStreamBandwidthEvent(raw string) *StreamBandwidthEvent {
	event := &StreamBandwidthEvent{Raw: raw}
	var temp string
	temp, raw, _ = util.PartitionString(raw, ' ')
	event.BytesRead, _ = strconv.ParseInt(temp, 10, 64)
	temp, raw, _ = util.PartitionString(raw, ' ')
	event.BytesWritten, _ = strconv.ParseInt(temp, 10, 64)
	temp, raw, _ = util.PartitionString(raw, ' ')
	temp, _ = util.UnescapeSimpleQuotedString(temp)
	event.Time = parseISOTime2Frac(temp)
	return event
}

func (*StreamBandwidthEvent) Code() EventCode { return EventCodeStreamBandwidth }

type ClientsSeenEvent struct {
	Raw            string
	TimeStarted    time.Time
	CountrySummary map[string]int
	IPVersions     map[string]int
}

func ParseClientsSeenEvent(raw string) *ClientsSeenEvent {
	event := &ClientsSeenEvent{Raw: raw}
	var temp string
	var ok bool
	temp, raw, ok = util.PartitionString(raw, ' ')
	temp, _ = util.UnescapeSimpleQuotedString(temp)
	event.TimeStarted = parseISOTime(temp)
	strToMap := func(str string) map[string]int {
		ret := map[string]int{}
		for _, keyVal := range strings.Split(str, ",") {
			key, val, _ := util.PartitionString(keyVal, '=')
			ret[key], _ = strconv.Atoi(val)
		}
		return ret
	}
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "CountrySummary":
			event.CountrySummary = strToMap(val)
		case "IPVersions":
			event.IPVersions = strToMap(val)
		}
	}
	return event
}

func (*ClientsSeenEvent) Code() EventCode { return EventCodeClientsSeen }

type NewConsensusEvent struct {
	Raw string
}

func ParseNewConsensusEvent(raw string) *NewConsensusEvent {
	return &NewConsensusEvent{Raw: raw}
}

func (*NewConsensusEvent) Code() EventCode { return EventCodeNewConsensus }

type BuildTimeoutSetEvent struct {
	Raw          string
	Type         string
	TotalTimes   int
	Timeout      time.Duration
	Xm           int
	Alpha        float32
	Quantile     float32
	TimeoutRate  float32
	CloseTimeout time.Duration
	CloseRate    float32
}

func ParseBuildTimeoutSetEvent(raw string) *BuildTimeoutSetEvent {
	event := &BuildTimeoutSetEvent{Raw: raw}
	var ok bool
	event.Type, raw, ok = util.PartitionString(raw, ' ')
	_, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	parseFloat := func(val string) float32 {
		f, _ := strconv.ParseFloat(val, 32)
		return float32(f)
	}
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "TOTAL_TIMES":
			event.TotalTimes, _ = strconv.Atoi(val)
		case "TIMEOUT_MS":
			if ms, err := strconv.ParseInt(val, 10, 64); err == nil {
				event.Timeout = time.Duration(ms) * time.Millisecond
			}
		case "XM":
			event.Xm, _ = strconv.Atoi(val)
		case "ALPHA":
			event.Alpha = parseFloat(val)
		case "CUTOFF_QUANTILE":
			event.Quantile = parseFloat(val)
		case "TIMEOUT_RATE":
			event.TimeoutRate = parseFloat(val)
		case "CLOSE_MS":
			if ms, err := strconv.ParseInt(val, 10, 64); err == nil {
				event.CloseTimeout = time.Duration(ms) * time.Millisecond
			}
		case "CLOSE_RATE":
			event.CloseRate = parseFloat(val)
		}
	}
	return event
}

func (*BuildTimeoutSetEvent) Code() EventCode { return EventCodeBuildTimeoutSet }

type SignalEvent struct {
	Raw string
}

func ParseSignalEvent(raw string) *SignalEvent {
	return &SignalEvent{Raw: raw}
}

func (*SignalEvent) Code() EventCode { return EventCodeSignal }

type ConfChangedEvent struct {
	Raw []string
}

func ParseConfChangedEvent(raw []string) *ConfChangedEvent {
	// TODO: break into KeyVal and unescape strings
	return &ConfChangedEvent{Raw: raw}
}

func (*ConfChangedEvent) Code() EventCode { return EventCodeConfChanged }

type CircuitMinorEvent struct {
	Raw         string
	CircuitID   string
	Event       string
	Path        []string
	BuildFlags  []string
	Purpose     string
	HSState     string
	RendQuery   string
	TimeCreated time.Time
	OldPurpose  string
	OldHSState  string
}

func ParseCircuitMinorEvent(raw string) *CircuitMinorEvent {
	event := &CircuitMinorEvent{Raw: raw}
	event.CircuitID, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	event.Event, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	first := true
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
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
		case "OLD_PURPOSE":
			event.OldPurpose = val
		case "OLD_HS_STATE":
			event.OldHSState = val
		default:
			if first {
				event.Path = strings.Split(val, ",")
			}
		}
		first = false
	}
	return event
}

func (*CircuitMinorEvent) Code() EventCode { return EventCodeCircuitMinor }

type TransportLaunchedEvent struct {
	Raw     string
	Type    string
	Name    string
	Address string
	Port    int
}

func ParseTransportLaunchedEvent(raw string) *TransportLaunchedEvent {
	event := &TransportLaunchedEvent{Raw: raw}
	event.Type, raw, _ = util.PartitionString(raw, ' ')
	event.Name, raw, _ = util.PartitionString(raw, ' ')
	event.Address, raw, _ = util.PartitionString(raw, ' ')
	var temp string
	temp, raw, _ = util.PartitionString(raw, ' ')
	event.Port, _ = strconv.Atoi(temp)
	return event
}

func (*TransportLaunchedEvent) Code() EventCode { return EventCodeTransportLaunched }

type ConnBandwidthEvent struct {
	Raw          string
	ConnID       string
	ConnType     string
	BytesRead    int64
	BytesWritten int64
}

func ParseConnBandwidthEvent(raw string) *ConnBandwidthEvent {
	event := &ConnBandwidthEvent{Raw: raw}
	ok := true
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "ID":
			event.ConnID = val
		case "TYPE":
			event.ConnType = val
		case "READ":
			event.BytesRead, _ = strconv.ParseInt(val, 10, 64)
		case "WRITTEN":
			event.BytesWritten, _ = strconv.ParseInt(val, 10, 64)
		}
	}
	return event
}

func (*ConnBandwidthEvent) Code() EventCode { return EventCodeConnBandwidth }

type CircuitBandwidthEvent struct {
	Raw          string
	CircuitID    string
	BytesRead    int64
	BytesWritten int64
	Time         time.Time
}

func ParseCircuitBandwidthEvent(raw string) *CircuitBandwidthEvent {
	event := &CircuitBandwidthEvent{Raw: raw}
	ok := true
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "ID":
			event.CircuitID = val
		case "READ":
			event.BytesRead, _ = strconv.ParseInt(val, 10, 64)
		case "WRITTEN":
			event.BytesWritten, _ = strconv.ParseInt(val, 10, 64)
		case "TIME":
			event.Time = parseISOTime2Frac(val)
		}
	}
	return event
}

func (*CircuitBandwidthEvent) Code() EventCode { return EventCodeCircuitBandwidth }

type CellStatsEvent struct {
	Raw             string
	CircuitID       string
	InboundQueueID  string
	InboundConnID   string
	InboundAdded    map[string]int
	InboundRemoved  map[string]int
	InboundTime     map[string]int
	OutboundQueueID string
	OutboundConnID  string
	OutboundAdded   map[string]int
	OutboundRemoved map[string]int
	OutboundTime    map[string]int
}

func ParseCellStatsEvent(raw string) *CellStatsEvent {
	event := &CellStatsEvent{Raw: raw}
	ok := true
	var attr string
	toIntMap := func(val string) map[string]int {
		ret := map[string]int{}
		for _, v := range strings.Split(val, ",") {
			key, val, _ := util.PartitionString(v, ':')
			ret[key], _ = strconv.Atoi(val)
		}
		return ret
	}
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "ID":
			event.CircuitID = val
		case "InboundQueue":
			event.InboundQueueID = val
		case "InboundConn":
			event.InboundConnID = val
		case "InboundAdded":
			event.InboundAdded = toIntMap(val)
		case "InboundRemoved":
			event.InboundRemoved = toIntMap(val)
		case "InboundTime":
			event.OutboundTime = toIntMap(val)
		case "OutboundQueue":
			event.OutboundQueueID = val
		case "OutboundConn":
			event.OutboundConnID = val
		case "OutboundAdded":
			event.OutboundAdded = toIntMap(val)
		case "OutboundRemoved":
			event.OutboundRemoved = toIntMap(val)
		case "OutboundTime":
			event.OutboundTime = toIntMap(val)
		}
	}
	return event
}

func (*CellStatsEvent) Code() EventCode { return EventCodeCellStats }

type TokenBucketEmptyEvent struct {
	Raw              string
	BucketName       string
	ConnID           string
	ReadBucketEmpty  time.Duration
	WriteBucketEmpty time.Duration
	LastRefil        time.Duration
}

func ParseTokenBucketEmptyEvent(raw string) *TokenBucketEmptyEvent {
	event := &TokenBucketEmptyEvent{Raw: raw}
	var ok bool
	event.BucketName, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, _ := util.PartitionString(attr, '=')
		switch key {
		case "ID":
			event.ConnID = val
		case "READ":
			i, _ := strconv.ParseInt(val, 10, 64)
			event.ReadBucketEmpty = time.Duration(i) * time.Millisecond
		case "WRITTEN":
			i, _ := strconv.ParseInt(val, 10, 64)
			event.WriteBucketEmpty = time.Duration(i) * time.Millisecond
		case "LAST":
			i, _ := strconv.ParseInt(val, 10, 64)
			event.LastRefil = time.Duration(i) * time.Millisecond
		}
	}
	return event
}

func (*TokenBucketEmptyEvent) Code() EventCode { return EventCodeTokenBucketEmpty }

type HSDescEvent struct {
	Raw        string
	Action     string
	Address    string
	AuthType   string
	HSDir      string
	DescID     string
	Reason     string
	Replica    int
	HSDirIndex string
}

func ParseHSDescEvent(raw string) *HSDescEvent {
	event := &HSDescEvent{Raw: raw}
	event.Action, raw, _ = util.PartitionString(raw, ' ')
	event.Address, raw, _ = util.PartitionString(raw, ' ')
	event.AuthType, raw, _ = util.PartitionString(raw, ' ')
	var ok bool
	event.HSDir, raw, ok = util.PartitionString(raw, ' ')
	var attr string
	first := true
	for ok {
		attr, raw, ok = util.PartitionString(raw, ' ')
		key, val, valOk := util.PartitionString(attr, '=')
		switch key {
		case "REASON":
			event.Reason = val
		case "REPLICA":
			event.Replica, _ = strconv.Atoi(val)
		case "HSDIR_INDEX":
			event.HSDirIndex = val
		default:
			if first && !valOk {
				event.DescID = attr
			}
		}
		first = false
	}
	return event
}

func (*HSDescEvent) Code() EventCode { return EventCodeHSDesc }

type HSDescContentEvent struct {
	Raw        string
	Address    string
	DescID     string
	HSDir      string
	Descriptor string
}

func ParseHSDescContentEvent(raw string) *HSDescContentEvent {
	event := &HSDescContentEvent{Raw: raw}
	event.Address, raw, _ = util.PartitionString(raw, ' ')
	event.DescID, raw, _ = util.PartitionString(raw, ' ')
	newlineIndex := strings.Index(raw, "\r\n")
	if newlineIndex != -1 {
		event.HSDir, event.Descriptor = raw[:newlineIndex], raw[newlineIndex+2:]
	}
	return event
}

func (*HSDescContentEvent) Code() EventCode { return EventCodeHSDescContent }

type NetworkLivenessEvent struct {
	Raw string
}

func ParseNetworkLivenessEvent(raw string) *NetworkLivenessEvent {
	return &NetworkLivenessEvent{Raw: raw}
}

func (*NetworkLivenessEvent) Code() EventCode { return EventCodeNetworkLiveness }
