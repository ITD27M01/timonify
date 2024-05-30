package timonify

import (
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/token"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

//func TestValues_Add(t *testing.T) {
//	t.Run("quote func added for string values", func(t *testing.T) {
//		testVal := Values{}
//		res, err := testVal.Add("abc", "a", "b")
//		assert.NoError(t, err)
//		assert.Contains(t, res, "quote")
//	})
//	t.Run("quote func not added for not string values", func(t *testing.T) {
//		testVal := Values{}
//		res, err := testVal.Add(int64(1), "a", "b")
//		assert.NoError(t, err)
//		assert.NotContains(t, res, "quote")
//		res, err = testVal.Add(true, "a", "b")
//		assert.NoError(t, err)
//		assert.NotContains(t, res, "quote")
//		res, err = testVal.Add(420.69, "a", "b")
//		assert.NoError(t, err)
//		assert.NotContains(t, res, "quote")
//	})
//	t.Run("name path is dot formatted", func(t *testing.T) {
//		testVal := Values{}
//		res, err := testVal.Add(int64(1), "a", "b")
//		assert.NoError(t, err)
//		assert.Contains(t, res, " .Values.a.b ")
//	})
//	t.Run("snake names camel cased", func(t *testing.T) {
//		testVal := Values{}
//		snake := "my_name"
//		camel := "myName"
//		res, err := testVal.Add(420.69, snake)
//		assert.NoError(t, err)
//		assert.NotContains(t, res, snake)
//		assert.Contains(t, res, camel)
//	})
//	t.Run("upper snake names camel cased", func(t *testing.T) {
//		testVal := Values{}
//		upSnake := "MY_NAME"
//		camel := "myName"
//		res, err := testVal.Add(420.69, upSnake)
//		assert.NoError(t, err)
//		assert.NotContains(t, res, upSnake)
//		assert.Contains(t, res, camel)
//	})
//	t.Run("kebab names camel cased", func(t *testing.T) {
//		testVal := Values{}
//		kebab := "my-name"
//		camel := "myName"
//		res, err := testVal.Add(420.69, kebab)
//		assert.NoError(t, err)
//		assert.NotContains(t, res, kebab)
//		assert.Contains(t, res, camel)
//	})
//	t.Run("dot names camel cased", func(t *testing.T) {
//		testVal := Values{}
//		dot := "my.name"
//		camel := "myName"
//		res, err := testVal.Add(420.69, dot)
//		assert.NoError(t, err)
//		assert.NotContains(t, res, dot)
//		assert.Contains(t, res, camel)
//	})
//}
//func TestValues_AddSecret(t *testing.T) {
//	t.Run("add base64 enc secret", func(t *testing.T) {
//		testVal := Values{}
//		res, err := testVal.AddSecret(true, "a", "b")
//		assert.NoError(t, err)
//		assert.Contains(t, res, "b64enc")
//	})
//	t.Run("add not encoded secret", func(t *testing.T) {
//		testVal := Values{}
//		res, err := testVal.AddSecret(false, "a", "b")
//		assert.NoError(t, err)
//		assert.NotContains(t, res, "b64enc")
//	})
//}

func Test_setNestedCueField(t *testing.T) {
	type args struct {
		config ast.Node
		value  ast.Expr
		name   []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"empty config",
			args{
				&ast.StructLit{},
				ast.NewLit(token.INT, "1"),
				[]string{"a", "b"},
			},
			`{
	a: {
		b: 1
	}
}`,
			assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, setNestedCueField(tt.args.config, tt.args.value, tt.args.name...), fmt.Sprintf("setNestedCueField(%v, %v, %v)", tt.args.config, tt.args.value, tt.args.name))
			b, err := format.Node(tt.args.config)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(b))
		})
	}
}
