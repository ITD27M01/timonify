package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	type fields struct {
		ModuleName  string
		Verbose     bool
		VeryVerbose bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "valid", fields: fields{ModuleName: ""}, wantErr: false},
		{name: "valid", fields: fields{ModuleName: "my.module123"}, wantErr: false},
		{name: "valid", fields: fields{ModuleName: "my-module123"}, wantErr: false},
		{name: "invalid", fields: fields{ModuleName: "my_module123"}, wantErr: true},
		{name: "invalid", fields: fields{ModuleName: "my char123t"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				ModuleName:  tt.fields.ModuleName,
				Verbose:     tt.fields.Verbose,
				VeryVerbose: tt.fields.VeryVerbose,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	t.Run("module name not set", func(t *testing.T) {
		c := &Config{}
		err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, defaultModuleName, c.ModuleName)
	})
	t.Run("module name set", func(t *testing.T) {
		c := &Config{ModuleName: "test"}
		err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, "test", c.ModuleName)
	})
}
