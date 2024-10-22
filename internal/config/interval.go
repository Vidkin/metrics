package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Interval int

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
