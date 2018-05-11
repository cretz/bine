package controltest

import "testing"

func TestAuthenticateNull(t *testing.T) {
	ctx, conn := NewTestContextConnected(t)
	defer ctx.CloseConnected(conn)
	// Verify auth methods before auth
	info, err := conn.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.ElementsMatch([]string{"NULL"}, info.AuthMethods)
	ctx.Require.NoError(conn.Authenticate(""))
}

func TestAuthenticateSafeCookie(t *testing.T) {
	ctx, conn := NewTestContextConnected(t, "--CookieAuthentication", "1")
	defer ctx.CloseConnected(conn)
	// Verify auth methods before auth
	info, err := conn.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.ElementsMatch([]string{"COOKIE", "SAFECOOKIE"}, info.AuthMethods)
	ctx.Require.NoError(conn.Authenticate(""))
}

func TestAuthenticateHashedPassword(t *testing.T) {
	// "testpass" - 16:5417AE717521511A609921392778FFA8518EC089BF2162A199241AEB4A
	ctx, conn := NewTestContextConnected(t, "--HashedControlPassword",
		"16:5417AE717521511A609921392778FFA8518EC089BF2162A199241AEB4A")
	defer ctx.CloseConnected(conn)
	// Verify auth methods before auth
	info, err := conn.ProtocolInfo()
	ctx.Require.NoError(err)
	ctx.Require.ElementsMatch([]string{"HASHEDPASSWORD"}, info.AuthMethods)
	ctx.Require.NoError(conn.Authenticate("testpass"))
}
