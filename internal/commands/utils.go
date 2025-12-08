package commands

import (
	"fmt"
	"strconv"
	"strings"
)

func parseDuration(str string) (int64, error) {
	str = strings.ToLower(str)
	if len(str) < 2 {
		return 0, fmt.Errorf("invalid length")
	}
	unit := str[len(str)-1]
	valStr := str[:len(str)-1]

	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 's':
		return int64(val * 1000), nil
	case 'm':
		return int64(val * 60 * 1000), nil
	case 'h':
		return int64(val * 60 * 60 * 1000), nil
	case 'd':
		return int64(val * 24 * 60 * 60 * 1000), nil
	default:
		return 0, fmt.Errorf("unknown unit")
	}
}
