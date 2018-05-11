package control

import (
	"strconv"
)

func (c *Conn) AttachStream(streamID string, circuitID string, hopNum int) error {
	if circuitID == "" {
		circuitID = "0"
	}
	cmd := "ATTACHSTREAM " + streamID + " " + circuitID
	if hopNum > 0 {
		cmd += " HOP=" + strconv.Itoa(hopNum)
	}
	return c.sendRequestIgnoreResponse(cmd)
}

func (c *Conn) RedirectStream(streamID string, address string, port int) error {
	cmd := "REDIRECTSTREAM " + streamID + " " + address
	if port > 0 {
		cmd += " " + strconv.Itoa(port)
	}
	return c.sendRequestIgnoreResponse(cmd)
}

func (c *Conn) CloseStream(streamID string, reason string) error {
	return c.sendRequestIgnoreResponse("CLOSESTREAM %v %v", streamID, reason)
}
