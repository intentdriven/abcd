package site

import (
	"math"
	"testing"
)

// BenchmarkSettle measures the packing pass over the corpus shape the
// overlap test uses — a tight spiral of very different sizes at the size of
// the live record — since settle runs twice per site build.
func BenchmarkSettle(b *testing.B) {
	const n = 600
	seed := make([]Point, n)
	radii := make([]float64, n)
	for i := range seed {
		ang := float64(i) * 0.7
		rho := float64(i) * 0.4
		seed[i] = Point{X: math.Cos(ang) * rho, Y: math.Sin(ang) * rho}
		radii[i] = 3 + float64(i%7)*3
	}
	members := make([]int, n)
	for i := range members {
		members[i] = i
	}
	pos := make([]Point, n)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(pos, seed)
		settle(pos, radii, members, false)
	}
}
