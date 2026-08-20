package leaky

func dropWater(w float64) float64 {
	return w
}

func applyFill(b *Bucket, n float64) {
	b.water += dropWater(n)
}
