package torutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartitionString(t *testing.T) {
	assert := func(str string, ch byte, expectedA string, expectedB string, expectedOk bool) {
		a, b, ok := PartitionString(str, ch)
		require.Equal(t, expectedA, a)
		require.Equal(t, expectedB, b)
		require.Equal(t, expectedOk, ok)
	}
	assert("foo:bar", ':', "foo", "bar", true)
	assert(":bar", ':', "", "bar", true)
	assert("foo:", ':', "foo", "", true)
	assert("foo", ':', "foo", "", false)
	assert("foo:bar:baz", ':', "foo", "bar:baz", true)
}

func TestPartitionStringFromEnd(t *testing.T) {
	assert := func(str string, ch byte, expectedA string, expectedB string, expectedOk bool) {
		a, b, ok := PartitionStringFromEnd(str, ch)
		require.Equal(t, expectedA, a)
		require.Equal(t, expectedB, b)
		require.Equal(t, expectedOk, ok)
	}
	assert("foo:bar", ':', "foo", "bar", true)
	assert(":bar", ':', "", "bar", true)
	assert("foo:", ':', "foo", "", true)
	assert("foo", ':', "foo", "", false)
	assert("foo:bar:baz", ':', "foo:bar", "baz", true)
}

func TestEscapeSimpleQuotedStringIfNeeded(t *testing.T) {
	assert := func(str string, shouldBeDiff bool) {
		maybeEscaped := EscapeSimpleQuotedStringIfNeeded(str)
		if shouldBeDiff {
			require.NotEqual(t, str, maybeEscaped)
		} else {
			require.Equal(t, str, maybeEscaped)
		}
	}
	assert("foo", false)
	assert(" foo", true)
	assert("f\\oo", true)
	assert("fo\"o", true)
	assert("f\roo", true)
	assert("fo\no", true)
}

func TestEscapeSimpleQuotedString(t *testing.T) {
	require.Equal(t, "\"foo\"", EscapeSimpleQuotedString("foo"))
}

func TestEscapeSimpleQuotedStringContents(t *testing.T) {
	assert := func(str string, expected string) {
		require.Equal(t, expected, EscapeSimpleQuotedStringContents(str))
	}
	assert("foo", "foo")
	assert("f\\oo", "f\\\\oo")
	assert("f\\noo", "f\\\\noo")
	assert("f\n o\ro", "f\\n o\\ro")
	assert("fo\r\\\"o", "fo\\r\\\\\\\"o")
}

func TestUnescapeSimpleQuotedStringIfNeeded(t *testing.T) {
	assert := func(str string, expectedStr string, expectedErr bool) {
		actualStr, actualErr := UnescapeSimpleQuotedStringIfNeeded(str)
		require.Equal(t, expectedStr, actualStr)
		require.Equal(t, expectedErr, actualErr != nil)
	}
	assert("foo", "foo", false)
	assert("\"foo\"", "foo", false)
	assert("\"f\"oo\"", "", true)
}

func TestUnescapeSimpleQuotedString(t *testing.T) {
	assert := func(str string, expectedStr string, expectedErr bool) {
		actualStr, actualErr := UnescapeSimpleQuotedString(str)
		require.Equal(t, expectedStr, actualStr)
		require.Equal(t, expectedErr, actualErr != nil)
	}
	assert("foo", "", true)
	assert("\"foo\"", "foo", false)
	assert("\"f\"oo\"", "", true)
}

func TestUnescapeSimpleQuotedStringContents(t *testing.T) {
	assert := func(str string, expectedStr string, expectedErr bool) {
		actualStr, actualErr := UnescapeSimpleQuotedStringContents(str)
		require.Equal(t, expectedStr, actualStr)
		require.Equal(t, expectedErr, actualErr != nil)
	}
	assert("foo", "foo", false)
	assert("f\\\\oo", "f\\oo", false)
	assert("f\\\\noo", "f\\noo", false)
	assert("f\\n o\\ro", "f\n o\ro", false)
	assert("fo\\r\\\\\\\"o", "fo\r\\\"o", false)
	assert("f\"oo", "", true)
	assert("f\roo", "", true)
	assert("f\noo", "", true)
	assert("f\\oo", "", true)
}
