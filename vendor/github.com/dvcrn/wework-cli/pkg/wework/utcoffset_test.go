package wework

import (
	"testing"
	"time"
)

func TestGetUTCOffsetForLocation_EmptyOffset(t *testing.T) {
	date := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		location     *Location
		expectedUTC  string
		expectError  bool
	}{
		{
			name: "Sao Paulo with empty offset",
			location: &Location{
				TimezoneOffset: "",
				TimeZone:       "America/Sao_Paulo",
			},
			expectedUTC: "-03:00",
			expectError: false,
		},
		{
			name: "Tokyo with empty offset",
			location: &Location{
				TimezoneOffset: "",
				TimeZone:       "Asia/Tokyo",
			},
			expectedUTC: "+09:00",
			expectError: false,
		},
		{
			name: "New York with empty offset",
			location: &Location{
				TimezoneOffset: "",
				TimeZone:       "America/New_York",
			},
			expectedUTC: "-04:00",
			expectError: false,
		},
		{
			name: "Location with existing offset (passthrough)",
			location: &Location{
				TimezoneOffset: "+10:00",
				TimeZone:       "Asia/Tokyo",
			},
			expectedUTC: "+10:00",
			expectError: false,
		},
		{
			name: "Location with both empty (error)",
			location: &Location{
				TimezoneOffset: "",
				TimeZone:       "",
			},
			expectedUTC: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getUTCOffsetForLocation(tt.location, date)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.expectedUTC {
				t.Errorf("getUTCOffsetForLocation() = %q, want %q", got, tt.expectedUTC)
			}
		})
	}
}

func TestGetUTCOffsetForLocation_DSTTransition(t *testing.T) {
	loc := &Location{
		TimezoneOffset: "",
		TimeZone:       "America/New_York",
	}

	winter := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	winterOffset, err := getUTCOffsetForLocation(loc, winter)
	if err != nil {
		t.Fatalf("unexpected error for winter date: %v", err)
	}
	if winterOffset != "-05:00" {
		t.Errorf("winter offset = %q, want -05:00 (EST)", winterOffset)
	}

	summer := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	summerOffset, err := getUTCOffsetForLocation(loc, summer)
	if err != nil {
		t.Fatalf("unexpected error for summer date: %v", err)
	}
	if summerOffset != "-04:00" {
		t.Errorf("summer offset = %q, want -04:00 (EDT)", summerOffset)
	}
}
