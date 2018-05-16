package tests

import (
	"testing"

	"github.com/cretz/bine/tor"
)

func TestAuthenticateNull(t *testing.T) {
	ctx := NewTestContext(t, &tor.StartConf{DisableCookieAuth: true, DisableEagerAuth: true})
	defer ctx.Close()
	// Verify auth methods before auth
	info, err := ctx.Control.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.ElementsMatch([]string{"NULL"}, info.AuthMethods)
	ctx.Require.NoError(ctx.Control.Authenticate(""))
}

func TestAuthenticateSafeCookie(t *testing.T) {
	ctx := NewTestContext(t, &tor.StartConf{DisableEagerAuth: true})
	defer ctx.Close()
	// Verify auth methods before auth
	info, err := ctx.Control.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.ElementsMatch([]string{"COOKIE", "SAFECOOKIE"}, info.AuthMethods)
	ctx.Require.NoError(ctx.Control.Authenticate(""))
}

func TestAuthenticateHashedPassword(t *testing.T) {
	// "testpass" - 16:5417AE717521511A609921392778FFA8518EC089BF2162A199241AEB4A
	ctx := NewTestContext(t, &tor.StartConf{
		DisableCookieAuth: true,
		DisableEagerAuth:  true,
		ExtraArgs:         []string{"--HashedControlPassword", "16:5417AE717521511A609921392778FFA8518EC089BF2162A199241AEB4A"},
	})
	defer ctx.Close()
	// Verify auth methods before auth
	info, err := ctx.Control.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.ElementsMatch([]string{"HASHEDPASSWORD"}, info.AuthMethods)
	ctx.Require.NoError(ctx.Control.Authenticate("testpass"))
}
