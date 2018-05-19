package torutil

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/cretz/bine/torutil/ed25519"
	"github.com/stretchr/testify/require"
)

func genRsa(t *testing.T, bits int) *rsa.PrivateKey {
	k, e := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, e)
	return k
}

func genEd25519(t *testing.T) ed25519.KeyPair {
	k, e := ed25519.GenerateKey(nil)
	require.NoError(t, e)
	return k
}

func TestOnionServiceIDFromPrivateKey(t *testing.T) {
	assert := func(key interface{}, shouldPanic bool) {
		if shouldPanic {
			require.Panics(t, func() { OnionServiceIDFromPrivateKey(key) })
		} else {
			require.NotPanics(t, func() { OnionServiceIDFromPrivateKey(key) })
		}
	}
	assert(nil, true)
	assert("bad type", true)
	assert(genRsa(t, 512), true)
	assert(genRsa(t, 1024), false)
	assert(genEd25519(t), false)
}

func TestOnionServiceIDFromPublicKey(t *testing.T) {
	assert := func(key interface{}, shouldPanic bool) {
		if shouldPanic {
			require.Panics(t, func() { OnionServiceIDFromPublicKey(key) })
		} else {
			require.NotPanics(t, func() { OnionServiceIDFromPublicKey(key) })
		}
	}
	assert(nil, true)
	assert("bad type", true)
	assert(genRsa(t, 512).Public(), true)
	assert(genRsa(t, 1024), true)
	assert(genRsa(t, 1024).Public(), false)
	assert(genEd25519(t), true)
	assert(genEd25519(t).Public(), false)
}

func TestOnionServiceIDFromV2PublicKey(t *testing.T) {
	keyNs := []string{
		// Good:
		"146324450904690776677821409274235322093351159271459646966603918354799061259062657420293876128692345182038558188684966983800327732948675482476772223940488746110835191444662533167597590461666544121677987412778085089886835490778554764504249900341150942052002951429704745527158573712253866271451928082512868548761",
		"128765593328258418045179848773717342016715415508670816023595649916344640363150769867803188216600032423256896646578968925175093584252663716464819945657362904808776223191437446288878991355616138261909405164010386485361833909203128674413041630645284466111155610996017814867550636519109125461589251061597106654453",
		"142816677715940797613971689484332187730646681999601531244837211468050734148365138492918019219363903243436898624689103727294808675158556496441738627693945143098034304873441312947712853824963023184593797741228534339590785521072446422663170639163836372239933736851693970208563926767141632739068954958552435402293",
		"145115352997770387235216741368218582671004692828327605746752722031765658311649572143281396789387808261614671508963042791801662334421789227429337599249357503724975792005849908733936522427330824294880823009884401313371327997012363609954851207630328042324027016587584799514594101157535904741483269310276131442141",
		"147719637109219630754585551462675301139659936682064979504052824885582296579356301771435242063159743126441027484306731955552256555531866636211668612294755914990702770530441483651548916585382488692381916953093261634746890673551241873307767188168965986976533243218915179497387875035829308609534245761833108189053",
		// Bad (512 bit):
		"12406459612976799354275054531003074054562219068852891594185203203668633138039185159716483674833390801567933368800574140712590387835931746258315639847176501",
	}
	matchingIDs := []string{
		"kqxsrkmm272hqvbj",
		"75rzoc3nxzucidqb",
		"l2vxsdecx6yita6r",
		"hzma3bmo7mtyr5mq",
		"prek6tayypvteljb",
		"",
	}
	for i, keyN := range keyNs {
		pubKey := &rsa.PublicKey{E: 65537, N: new(big.Int)}
		pubKey.N, _ = pubKey.N.SetString(keyN, 10)
		matchingID := matchingIDs[i]
		if matchingID == "" {
			require.Panics(t, func() { OnionServiceIDFromV2PublicKey(pubKey) })
		} else {
			require.Equal(t, matchingID, OnionServiceIDFromV2PublicKey(pubKey))
		}
	}
}

func TestOnionServiceIDFromV3PublicKey(t *testing.T) {
	base64Keys := []string{
		"SLne6D/uawqUj23619GbeYCd6HnzYPqyUvF8/xyz/3XNVpkgnonQI+J5NQVSGkppD1b0M87+qOtUBmVXsd7H3w",
		"kPUs5aPoqISZVbg0q7coW+mNCODlcL4O7k2QWFOCC0gOQBiDm+g4Xz48lqucA7o2HIQ3gBdL5rlB6+q1tFdJwQ",
		"YGzw/EwpcqfWb5UWIw652Ps4vTKu38VgX7Qo16XvOWjNWQK9YmfgARYiGQ1XYXEAKBJvoq8x+rKFbQN3FG1F6w",
		"IJIZcWE57n5WCvHU2x7GkpBCIw0S0vWd+QyrE5RifGwPtYsbtxjyOxlb754Z0zXLZc+yQUp9hMQt5dt/YNpMag",
		"SD7d4I6ZOjNlcqR2g4ptFJUw0tUHPQvfk92sExvnJ1uofPw9T9LUaaEs3rE/1yoGWKI4YejAzaTJXF9wrWQyuA",
	}
	matchingIDs := []string{
		"2s2wk473fmotzgh6l2ycigrwegnurlzufatjm3bglrb36zbvlerskxad",
		"tmcpdbgklpbywqyjpr7fijvjl7qjihd7pyubosbeohefec2m2thvzoqd",
		"nrcan5uye2fwazixubug6pzrzp6ofjez43bjcyfoxhgxyygxbhgs4zqd",
		"g2csv4kavhunvs45vxxc5ljz775d5a4ycqo4m4nrwpk3b4gryvz2zdyd",
		"jviaiibaz7r6wqxttj5i2bi4zjfilmsevplwwtxdfyjph2sdmq5osdid",
	}
	for i, base64Key := range base64Keys {
		key, err := base64.RawStdEncoding.DecodeString(base64Key)
		require.NoError(t, err)
		pubKey := ed25519.PrivateKey(key).PublicKey()
		matchingID := matchingIDs[i]
		require.Equal(t, matchingID, OnionServiceIDFromV3PublicKey(pubKey))
		// Check verify here too
		derivedPubKey, err := PublicKeyFromV3OnionServiceID(matchingID)
		require.NoError(t, err)
		require.Equal(t, pubKey, derivedPubKey)
		// Let's mangle the matchingID a bit
		tooLong := matchingID + "ddddd"
		_, err = PublicKeyFromV3OnionServiceID(tooLong)
		require.EqualError(t, err, "Invalid id length")
		badVersion := matchingID[:len(matchingID)-1] + "e"
		_, err = PublicKeyFromV3OnionServiceID(badVersion)
		require.EqualError(t, err, "Invalid version")
		badChecksum := []byte(matchingID)
		badChecksum[len(badChecksum)-3] = 'q'
		_, err = PublicKeyFromV3OnionServiceID(string(badChecksum))
		require.EqualError(t, err, "Invalid checksum")
	}
}
