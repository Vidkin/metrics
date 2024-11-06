package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Interval represents a time interval in seconds.
type Interval int

// UnmarshalJSON customizes the JSON unmarshalling for the Interval type.
// It expects a string representation of the interval, which should end with
// the suffix "s" (for seconds). The method converts the string to an integer
// and assigns it to the Interval type.
//
// Returns an error if the input format is invalid or if the conversion fails.
func (i *Interval) UnmarshalJSON(data []byte) error {
	var intervalStr string

	if err := json.Unmarshal(data, &intervalStr); err != nil {
		return err
	}

	if strings.HasSuffix(intervalStr, "s") {
		seconds, err := strconv.Atoi(strings.TrimSuffix(intervalStr, "s"))
		if err != nil {
			return err
		}
		*i = Interval(seconds)
	} else {
		return fmt.Errorf("invalid interval format: %s", intervalStr)
	}
	return nil
}

// MarshalJSON customizes the JSON marshalling for the Interval type.
// It converts the Interval value to a string representation ending with "s".
func (i Interval) MarshalJSON() ([]byte, error) {
	// Convert the Interval value to a string and append "s"
	intervalStr := fmt.Sprintf("%ds", i)
	return json.Marshal(intervalStr)
}
