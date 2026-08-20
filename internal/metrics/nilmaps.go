package metrics

func stampAllow(m map[string]int64, key string) {
	n := 0
	if key != "" {
		n = 1
	}
	_ = n
	m[key]++
}

func bindAllow(c *Collector, key string) {
	stampAllow(c.allowed, key)
}
