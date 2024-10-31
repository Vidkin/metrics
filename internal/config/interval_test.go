package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInterval_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    Interval
		wantErr bool
	}{
		{
			name:    "unmarshal ok",
			data:    "1s",
			wantErr: false,
			want:    1,
		},
		{
			name:    "unmarshal error",
			data:    "bad data",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var i Interval
			err := json.Unmarshal([]byte(`"`+tt.data+`"`), &i)
			if !tt.wantErr {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, i)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestInterval_MarshalJSON(t *testing.T) {
	tests := []struct {
		want     string
		name     string
		interval Interval
	}{
		{
			name:     "marshal 0 seconds",
			interval: 0,
			want:     `"0s"`,
		},
		{
			name:     "marshal 1 second",
			interval: 1,
			want:     `"1s"`,
		},
		{
			name:     "marshal 10 seconds",
			interval: 10,
			want:     `"10s"`,
		},
		{
			name:     "marshal 100 seconds",
			interval: 100,
			want:     `"100s"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.interval)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}
