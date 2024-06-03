package format

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuoteStringsInStruct(t *testing.T) {
	type args struct {
		s interface{}
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "Test with single string field",
			args: args{
				s: &struct {
					A string
				}{
					A: "Hello",
				},
			},
			want: &struct {
				A string
			}{
				A: "\"Hello\"",
			},
		},
		{
			name: "Test with nested struct",
			args: args{
				s: &struct {
					A string
					B struct {
						C string
					}
				}{
					A: "Hello",
					B: struct {
						C string
					}{
						C: "World",
					},
				},
			},
			want: &struct {
				A string
				B struct {
					C string
				}
			}{
				A: "\"Hello\"",
				B: struct {
					C string
				}{
					C: "\"World\"",
				},
			},
		},
		{
			name: "Test with slice",
			args: args{
				s: &[]*struct {
					A string
					B struct {
						C string
					}
				}{{
					A: "Hello",
					B: struct {
						C string
					}{
						C: "World",
					},
				}, {
					A: "Hello",
					B: struct {
						C string
					}{
						C: "World",
					},
				}},
			},
			want: &[]*struct {
				A string
				B struct {
					C string
				}
			}{{
				A: "\"Hello\"",
				B: struct {
					C string
				}{
					C: "\"World\"",
				},
			}, {
				A: "\"Hello\"",
				B: struct {
					C string
				}{
					C: "\"World\"",
				},
			}},
		},
		{
			name: "Test with empty strings",
			args: args{
				s: &struct {
					A string
					B string
				}{
					A: "",
					B: "",
				},
			},
			want: &struct {
				A string
				B string
			}{
				A: "",
				B: "",
			},
		},
		{
			name: "Test with pointer in struct",
			args: args{
				s: &struct {
					A *string
				}{
					A: func() *string {
						s := "Hello"
						return &s
					}(),
				},
			},
			want: &struct {
				A *string
			}{
				A: func() *string {
					s := "\"Hello\""
					return &s
				}(),
			},
		},
		{
			name: "Test with unexported field",
			args: args{
				s: &struct {
					a string
				}{
					a: "Hello",
				},
			},
			want: &struct {
				a string
			}{
				a: "Hello",
			},
		},
		{
			name: "Test with map[string]string",
			args: args{
				s: &map[string]string{
					"key1": "Hello",
					"key2": "World",
				},
			},
			want: &map[string]string{
				"key1": "\"Hello\"",
				"key2": "\"World\"",
			},
		},
		{
			name: "Test with map[string]struct",
			args: args{
				s: &map[string]struct {
					A string
					B string
				}{
					"key1": {
						A: "Hello",
						B: "World",
					},
					"key2": {
						A: "Foo",
						B: "Bar",
					},
				},
			},
			want: &map[string]struct {
				A string
				B string
			}{
				"key1": {
					A: "\"Hello\"",
					B: "\"World\"",
				},
				"key2": {
					A: "\"Foo\"",
					B: "\"Bar\"",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			QuoteStringsInStruct(tt.args.s)
			assert.Equal(t, tt.want, tt.args.s)
		})
	}
}
