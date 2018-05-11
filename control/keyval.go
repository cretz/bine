package control

type KeyVal struct {
	Key string
	Val string
	// If true and the associated command supports nil vals, then an empty string for val is NOT considered nil like it
	// otherwise would. This is ignored for commands that don't support nil vals.
	ValSetAndEmpty bool
}

func NewKeyVal(key string, val string) *KeyVal {
	return &KeyVal{Key: key, Val: val}
}

func KeyVals(keysAndVals ...string) []*KeyVal {
	if len(keysAndVals)%2 != 0 {
		panic("Expected multiple of 2")
	}
	ret := make([]*KeyVal, len(keysAndVals)/2)
	for i := 0; i < len(ret); i++ {
		ret[i] = NewKeyVal(keysAndVals[i*2], keysAndVals[i*2+1])
	}
	return ret
}

func (k *KeyVal) ValSet() bool {
	return len(k.Val) > 0 || k.ValSetAndEmpty
}
