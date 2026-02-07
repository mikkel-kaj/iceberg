package hetzner

import "strconv"

func itoa64(v int64) string {
	return strconv.FormatInt(v, 10)
}
