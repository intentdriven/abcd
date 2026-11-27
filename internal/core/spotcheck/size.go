package spotcheck

// size.go is the sample size (spc-2610301925429250 approach 5): it adapts
// over fully reviewed cycles only.

// sampleSize is the size a new cycle draws per group, over the cycles before
// it newest first (tallyCycles): half of perGroup, rounded up and never below
// one, when the three most recent closed cycles each found nothing; perGroup
// otherwise, so a closed cycle with a finding makes the next size full. An
// open cycle is skipped, counting as neither clean nor a finding. counted
// names the closed cycles the size was decided over, newest first.
func sampleSize(perGroup int, newestFirst []cycleTally) (size int, why string, counted []string) {
	counted = []string{}
	clean := true
	for _, c := range newestFirst {
		if !c.closed {
			continue
		}
		counted = append(counted, c.id)
		if c.findings > 0 {
			clean = false
		}
		if len(counted) == 3 {
			break
		}
	}
	if len(counted) == 3 && clean {
		return max((perGroup+1)/2, 1), SizeHalved, counted
	}
	return perGroup, SizeFull, counted
}
