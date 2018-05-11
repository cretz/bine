package controltest

import (
	"io/ioutil"
	"testing"

	"github.com/cretz/bine/control"
)

func TestGetSetAndResetConf(t *testing.T) {
	ctx, conn := NewTestContextAuthenticated(t)
	defer ctx.CloseConnected(conn)
	// Simple get conf
	assertConfVals := func(val string) {
		entries, err := conn.GetConf("LogMessageDomains", "ProtocolWarnings")
		ctx.Require.NoError(err)
		ctx.Require.Len(entries, 2)
		ctx.Require.Contains(entries, control.NewKeyVal("LogMessageDomains", val))
		ctx.Require.Contains(entries, control.NewKeyVal("ProtocolWarnings", val))
	}
	assertConfVals("0")
	// Change em both to 1
	one := "1"
	err := conn.SetConf(control.KeyVals("LogMessageDomains", "1", "ProtocolWarnings", "1")...)
	ctx.Require.NoError(err)
	// Check again
	assertConfVals(one)
	// Reset em both
	err = conn.ResetConf(control.KeyVals("LogMessageDomains", "", "ProtocolWarnings", "")...)
	ctx.Require.NoError(err)
	// Make sure both back to zero
	assertConfVals("0")
}

func TestLoadConf(t *testing.T) {
	ctx, conn := NewTestContextAuthenticated(t)
	defer ctx.CloseConnected(conn)
	// Get entire conf text
	vals, err := conn.GetInfo("config-text")
	ctx.Require.NoError(err)
	ctx.Require.Len(vals, 1)
	ctx.Require.Equal("config-text", vals[0].Key)
	confText := vals[0].Val
	// Append new conf val and load
	ctx.Require.NotContains(confText, "LogMessageDomains")
	confText += "\r\nLogMessageDomains 1"
	ctx.Require.NoError(conn.LoadConf(confText))
	// Check the new val
	vals, err = conn.GetInfo("config-text")
	ctx.Require.NoError(err)
	ctx.Require.Len(vals, 1)
	ctx.Require.Equal("config-text", vals[0].Key)
	ctx.Require.Contains(vals[0].Val, "LogMessageDomains 1")
}

func TestSaveConf(t *testing.T) {
	ctx, conn := NewTestContextAuthenticated(t)
	defer ctx.CloseConnected(conn)
	// Get conf filename
	vals, err := conn.GetInfo("config-file")
	ctx.Require.NoError(err)
	ctx.Require.Len(vals, 1)
	ctx.Require.Equal("config-file", vals[0].Key)
	confFile := vals[0].Val
	// Save it
	ctx.Require.NoError(conn.SaveConf(false))
	// Read and make sure, say, the DataDirectory is accurate
	confText, err := ioutil.ReadFile(confFile)
	ctx.Require.NoError(err)
	ctx.Require.Contains(string(confText), "DataDirectory "+ctx.TestTor.DataDir)
}
