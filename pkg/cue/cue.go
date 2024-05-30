package cue

import (
	"bytes"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
)

// Indent - adds indentation to given content.
func Indent(content []byte, n int) []byte {
	if n < 0 {
		return content
	}
	prefix := append([]byte("\n"), bytes.Repeat([]byte(" "), n)...)
	content = append(prefix[1:], content...)
	return bytes.ReplaceAll(content, []byte("\n"), prefix)
}

// Marshal object to cue string with indentation.
func Marshal(object interface{}, indent int) (string, error) {
	ctx := cuecontext.New()
	objectValue := ctx.Encode(object)
	if objectValue.Err() != nil {
		return "", objectValue.Err()
	}
	objectBytes, err := format.Node(objectValue.Syntax())
	if err != nil {
		return "", err
	}
	objectBytes = Indent(objectBytes, indent)
	objectBytes = bytes.TrimRight(objectBytes, "\n ")
	return string(objectBytes), nil
}
