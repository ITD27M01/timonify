package cue

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestIndent(t *testing.T) {
	type args struct {
		content []byte
		n       int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "negative",
			args: args{[]byte("a"), -1},
			want: []byte("a"),
		},
		{
			name: "none",
			args: args{[]byte("a"), 0},
			want: []byte("a"),
		},
		{
			name: "one",
			args: args{[]byte("a"), 1},
			want: []byte(" a"),
		},
		{
			name: "two",
			args: args{[]byte("a"), 2},
			want: []byte("  a"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Indent(tt.args.content, tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Indent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseStringLits(t *testing.T) {
	type args struct {
		node ast.Node
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test with valid string literal",
			args: args{
				node: &ast.StructLit{
					Elts: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("key"),
							Value: &ast.BasicLit{Kind: token.STRING, Value: `"\\(string)"`},
						},
					},
				},
			},
			want: `{
	key: "\(string)"
}`,
			wantErr: false,
		},
		{
			name: "Test with invalid string literal",
			args: args{
				node: &ast.StructLit{
					Elts: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("key"),
							Value: &ast.BasicLit{Kind: token.STRING, Value: "\"value"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Test with nested struct literal",
			args: args{
				node: &ast.StructLit{
					Elts: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("key"),
							Value: &ast.StructLit{
								Elts: []ast.Decl{
									&ast.Field{
										Label: ast.NewIdent("nestedKey"),
										Value: &ast.BasicLit{Kind: token.STRING, Value: `"\\(string)"`},
									},
								},
							},
						},
					},
				},
			},
			want: `{
	key: {
		nestedKey: "\(string)"
	}
}`,
			wantErr: false,
		},
		{
			name: "Test with list literal",
			args: args{
				node: &ast.StructLit{
					Elts: []ast.Decl{
						&ast.Field{
							Label: ast.NewIdent("key"),
							Value: &ast.ListLit{
								Elts: []ast.Expr{
									&ast.BasicLit{Kind: token.STRING, Value: `"\\(string)"`},
									&ast.BasicLit{Kind: token.STRING, Value: `"\\(anotherstring)"`},
									&ast.StructLit{
										Elts: []ast.Decl{
											&ast.Field{
												Label: ast.NewIdent("key"),
												Value: &ast.BasicLit{Kind: token.STRING, Value: `"\\(string)"`},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: `{
	key: ["\(string)", "\(anotherstring)", {
		key: "\(string)"
	}]
}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseStringLits(tt.args.node); (err != nil) != tt.wantErr {
				t.Errorf("parseStringLits() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				res, err := format.Node(tt.args.node)
				assert.NoError(t, err)
				assert.Equal(t, tt.want, string(res))
			}
		})
	}
}
