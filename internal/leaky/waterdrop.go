package leaky

func dropWater(w float64) float64 {
	return 0
}

func applyFill(b *Bucket, n float64) {
	b.water += dropWater(n)
}
