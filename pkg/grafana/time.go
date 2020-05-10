package grafana

import (
	"errors"
	"regexp"
)

func ParseTimeRange(s string) (string, error) {
	regex := regexp.MustCompile(`^\d+([mhdyM])$`)
	if regex.MatchString(s) {
		return "now-" + s, nil
	}
	return "", errors.New("this time range is invalid")
}
