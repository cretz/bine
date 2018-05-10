package util

import (
	"fmt"
	"strings"
)

func PartitionString(str string, ch byte) (string, string, bool) {
	index := strings.IndexByte(str, ch)
	if index == -1 {
		return str, "", false
	}
	return str[:index], str[index+1:], true
}

func ParseSimpleQuotedString(str string) (string, error) {
	if len(str) < 2 || str[0] != '"' || str[len(str)-1] != '"' {
		return "", fmt.Errorf("Missing quotes")
	}
	return UnescapeSimpleQuoted(str[1 : len(str)-1])
}

func UnescapeSimpleQuoted(str string) (string, error) {
	ret := ""
	escaping := false
	for _, c := range str {
		switch c {
		case '\\':
			if escaping {
				ret += "\\"
			}
			escaping = !escaping
		case '"':
			if !escaping {
				return "", fmt.Errorf("Unescaped quote")
			}
			ret += "\""
			escaping = false
		default:
			if escaping {
				return "", fmt.Errorf("Unexpected escape")
			}
			ret += string(c)
		}
	}
	return ret, nil
}
