package domain

import (
	"errors"
	"testing"
)

func TestParseMBID(t *testing.T) {
	tests := []struct {
		in      string
		want    MBID
		wantErr bool
	}{
		{"5b11f4ce-a62d-471e-81fc-a69a8278c7da", "5b11f4ce-a62d-471e-81fc-a69a8278c7da", false},
		{" 5B11F4CE-A62D-471E-81FC-A69A8278C7DA ", "5b11f4ce-a62d-471e-81fc-a69a8278c7da", false},
		{"", "", true},
		{"5b11f4ce-a62d-471e-81fc-a69a8278c7d", "", true},  // too short
		{"5b11f4cea62d-471e-81fc-a69a8278c7da0", "", true}, // hyphen misplaced
		{"5b11f4ce-a62d-471e-81fc-a69a8278c7dz", "", true}, // not hex
	}
	for _, tt := range tests {
		got, err := ParseMBID(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseMBID(%q) error = %v, wantErr %t", tt.in, err, tt.wantErr)
			continue
		}
		if err != nil && !errors.Is(err, ErrValidation) {
			t.Errorf("ParseMBID(%q) error %v is not ErrValidation", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ParseMBID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
