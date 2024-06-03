package cue

import (
	"bytes"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	"cuelang.org/go/cue/token"
	"fmt"
	"strconv"
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
func Marshal(object interface{}, indent int, parse bool) (string, error) {
	ctx := cuecontext.New()
	objectValue := ctx.Encode(object)
	if objectValue.Err() != nil {
		return "", objectValue.Err()
	}
	node := objectValue.Syntax()

	if parse {
		if _, err := parseStringLits(node); err != nil {
			return "", fmt.Errorf("%w: failed to parse string literals", err)
		}
	}
	objectBytes, err := format.Node(node)
	if err != nil {
		return "", err
	}
	objectBytes = Indent(objectBytes, indent)
	objectBytes = bytes.TrimRight(objectBytes, "\n ")
	return string(objectBytes), nil
}

// parseStringLits checks every field recursively if it has an ast.BasicLit with kind token.STRING call parser.ParseExpr result
func parseStringLits(node ast.Node) (ast.Node, error) {
	switch node := node.(type) {
	case *ast.StructLit:
		for _, decl := range node.Elts {
			_, err := parseStringLits(decl)
			if err != nil {
				return nil, err
			}
		}
	case *ast.ListLit:
		for i, decl := range node.Elts {
			if isStringLit(decl) {
				parsed, err := parseStringLit(decl.(*ast.BasicLit))
				if err == nil {
					node.Elts[i] = parsed
				}
				continue
			}
			decl, err := parseStringLits(decl)
			if err != nil {
				return node, err
			}
			node.Elts[i] = decl.(ast.Expr)
		}

	case *ast.Field:
		if isStringLit(node.Value) {
			parsed, err := parseStringLit(node.Value.(*ast.BasicLit))
			if err != nil {
				return node, nil
			}
			node.Value = parsed
			return node, nil
		}
		_, err := parseStringLits(node.Value)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

// isStringLit checks if the value is a BasicLit with kind token.STRING
func isStringLit(v ast.Expr) bool {
	if v, ok := v.(*ast.BasicLit); ok && v.Kind == token.STRING {
		return true
	}
	return false
}

func parseStringLit(v *ast.BasicLit) (ast.Expr, error) {
	value, err := strconv.Unquote(v.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to unquote string", err)
	}
	return parser.ParseExpr("", value)
}

func MustParse(src string) ast.Expr {
	expr, err := parser.ParseExpr("", []byte(src))
	if err != nil {
		// This should never happen and shows that something is wrong with the cue parsing
		panic(fmt.Errorf("%w: failed to parse expression", err))
	}
	return expr
}
