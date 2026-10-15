package session

import "time"

// bucket is a token bucket: it holds up to burst tokens, refills at rate
// tokens a second, and each message takes one (or, counted in bytes, its
// size). Its zero value is full on first use. Called under the lock that
// guards its owner, with the relay's clock.
type bucket struct {
	tokens float64
	last   time.Time
	used   bool
}

// take spends a token if one is left, refilling first for the time since the
// last call.
func (b *bucket) take(now time.Time, rate, burst int) bool {
	return b.takeN(now, rate, burst, 1)
}

// takeN spends n tokens if that many are left, refilling first.
func (b *bucket) takeN(now time.Time, rate, burst, n int) bool {
	if !b.used {
		b.used, b.tokens, b.last = true, float64(burst), now
	}
	if elapsed := now.Sub(b.last); elapsed > 0 {
		b.tokens = min(float64(burst), b.tokens+elapsed.Seconds()*float64(rate))
		b.last = now
	}
	if b.tokens < float64(n) {
		return false
	}
	b.tokens -= float64(n)
	return true
}
