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
