package db

import (
	"strconv"
	"strings"
)

func rebind(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteByte('$')
			b.Write(strconv.AppendInt(nil, int64(n), 10))
			n++
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}
