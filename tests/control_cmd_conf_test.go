package tests

import (
	"io/ioutil"
	"testing"

	"github.com/cretz/bine/control"
)

func TestGetSetAndResetConf(t *testing.T) {
	ctx := NewTestContext(t, nil)
	defer ctx.Close()
	// Simple get conf
	assertConfVals := func(val string) {
		entries, err := ctx.Control.GetConf("LogMessageDomains", "ProtocolWarnings")
		ctx.Require.NoError(err)
		ctx.Require.Len(entries, 2)
		ctx.Require.Contains(entries, control.NewKeyVal("LogMessageDomains", val))
		ctx.Require.Contains(entries, control.NewKeyVal("ProtocolWarnings", val))
	}
	assertConfVals("0")
	// Change em both to 1
	one := "1"
	err := ctx.Control.SetConf(control.KeyVals("LogMessageDomains", "1", "ProtocolWarnings", "1")...)
	ctx.Require.NoError(err)
	// Check again
	assertConfVals(one)
	// Reset em both
	err = ctx.Control.ResetConf(control.KeyVals("LogMessageDomains", "", "ProtocolWarnings", "")...)
	ctx.Require.NoError(err)
	// Make sure both back to zero
	assertConfVals("0")
}

func TestLoadConf(t *testing.T) {
	ctx := NewTestContext(t, nil)
	defer ctx.Close()
	// Get entire conf text
	vals, err := ctx.Control.GetInfo("config-text")
	ctx.Require.NoError(err)
	ctx.Require.Len(vals, 1)
	ctx.Require.Equal("config-text", vals[0].Key)
	confText := vals[0].Val
	// Append new conf val and load
	ctx.Require.NotContains(confText, "LogMessageDomains")
	confText += "\r\nLogMessageDomains 1"
	ctx.Require.NoError(ctx.Control.LoadConf(confText))
	// Check the new val
	vals, err = ctx.Control.GetInfo("config-text")
	ctx.Require.NoError(err)
	ctx.Require.Len(vals, 1)
	ctx.Require.Equal("config-text", vals[0].Key)
	ctx.Require.Contains(vals[0].Val, "LogMessageDomains 1")
}

func TestSaveConf(t *testing.T) {
	ctx := NewTestContext(t, nil)
	defer ctx.Close()
	// Get conf filename
	vals, err := ctx.Control.GetInfo("config-file")
	ctx.Require.NoError(err)
	ctx.Require.Len(vals, 1)
	ctx.Require.Equal("config-file", vals[0].Key)
	confFile := vals[0].Val
	// Save it
	ctx.Require.NoError(ctx.Control.SaveConf(false))
	// Read and make sure, say, the DataDirectory is accurate
	confText, err := ioutil.ReadFile(confFile)
	ctx.Require.NoError(err)
	ctx.Require.Contains(string(confText), "DataDirectory "+ctx.DataDir)
}
