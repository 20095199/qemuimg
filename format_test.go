package qemuimg

import (
	"errors"
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Format
		wantErr bool
	}{
		{name: "empty means infer", input: "", want: ""},
		{name: "raw", input: "raw", want: FormatRaw},
		{name: "qcow2", input: "qcow2", want: FormatQcow2},
		{name: "unsupported", input: "vmdk", wantErr: true},
		{name: "case sensitive", input: "RAW", wantErr: true},
		{name: "whitespace", input: " raw", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseFormat(%q) returned no error", tt.input)
				}
				if !errors.Is(err, ErrInvalidFormat) {
					t.Fatalf("ParseFormat(%q) error %v does not wrap ErrInvalidFormat", tt.input, err)
				}
				var invalid *InvalidFormatError
				if !errors.As(err, &invalid) {
					t.Fatalf("ParseFormat(%q) error is not *InvalidFormatError: %T", tt.input, err)
				}
				if invalid.Value != tt.input {
					t.Fatalf("InvalidFormatError.Value = %q, want %q", invalid.Value, tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFormat(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if got.String() != string(tt.want) {
				t.Fatalf("Format.String() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestFormatValidate(t *testing.T) {
	tests := []struct {
		name    string
		format  Format
		wantErr bool
	}{
		{name: "empty", format: ""},
		{name: "raw", format: FormatRaw},
		{name: "qcow2", format: FormatQcow2},
		{name: "invalid", format: "vdi", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.format.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Validate() returned no error for %q", tt.format)
				}
				if !errors.Is(err, ErrInvalidFormat) {
					t.Fatalf("Validate() error %v does not wrap ErrInvalidFormat", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() returned error for %q: %v", tt.format, err)
			}
		})
	}
}
