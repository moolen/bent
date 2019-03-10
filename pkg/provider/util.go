package provider

import (
	"strconv"
	"strings"
	"time"
)

func parseIntWithFallback(val string, fallback int) int {
	num, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return num
}

func parseDurationWithFallback(val string, fallback time.Duration) time.Duration {
	num, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return time.Duration(num)
}

func parseInt64RangeWithFallback(val string, fallback1, fallback2 int64) (int64, int64) {
	list := strings.Split(val, "-")
	if len(list) != 2 {
		return fallback1, fallback2
	}
	num1, err := strconv.ParseInt(list[0], 10, 64)
	if err != nil {
		return fallback1, fallback2
	}
	num2, err := strconv.ParseInt(list[2], 10, 64)
	if err != nil {
		return fallback1, fallback2
	}
	return num1, num2
}
