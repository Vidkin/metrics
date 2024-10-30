package metric

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetric_ValueAsString(t *testing.T) {
	fVal := 42.1
	iVal := int64(42)

	tests := []struct {
		me   Metric
		name string
		want string
	}{
		{
			name: "test gauge ok",
			me: Metric{
				MType: "gauge",
				Value: &fVal,
				ID:    "gauge",
			},
			want: strconv.FormatFloat(fVal, 'g', -1, 64),
		},
		{
			name: "test counter ok",
			me: Metric{
				MType: "counter",
				Delta: &iVal,
				ID:    "counter",
			},
			want: strconv.FormatInt(iVal, 10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.me.ValueAsString())
		})
	}
}
