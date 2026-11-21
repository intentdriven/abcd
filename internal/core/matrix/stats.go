package matrix

// stats.go is the run's statistics (spc-2610161949559050 step 8; decisions 8,
// 21 and 26; the SOTA note's repeats line): a code cell's pass@1 with its
// interval and its pass^k; the power estimate that says how many repeats a
// comparison needs and the smallest difference it could have detected at the
// repeats it had; a regularised Bradley–Terry rating of each model from the
// judges' rankings, with intervals bootstrapped over judges and copies and a
// model that cannot be told from another reported as not separable; and the
// judges' agreement as Krippendorff's alpha with its interval, or not
// estimable, beside plain percentage agreement. Every draw is seeded, so a
// report read twice from the same records says the same thing.

import (
	"math"
	"math/rand/v2"
	"slices"
	"sort"
)

// The two-sided 5% test at 80% power every power estimate uses.
const (
	zAlpha = 1.959963984540054
	zBeta  = 0.8416212335729143
)

// bootstrapDraws is how many resamples an interval is read from.
const bootstrapDraws = 1000

// Interval is a 95% interval.
type Interval struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

// PassRate is a cell's pass@1: of its attempts that count (passes and
// failures), how many passed, the rate and its 95% Wilson interval, both nil
// where nothing counts.
type PassRate struct {
	Passed   int       `json:"passed"`
	Attempts int       `json:"attempts"`
	Rate     *float64  `json:"rate"`
	Interval *Interval `json:"interval"`
}

// passAt1 is passed of attempts as a pass rate with its Wilson interval, which
// stays inside nought and one and says something at a rate of either.
func passAt1(passed, attempts int) PassRate {
	r := PassRate{Passed: passed, Attempts: attempts}
	if attempts <= 0 {
		return r
	}
	n := float64(attempts)
	p := float64(passed) / n
	z2 := zAlpha * zAlpha
	denom := 1 + z2/n
	centre := (p + z2/(2*n)) / denom
	half := zAlpha * math.Sqrt(p*(1-p)/n+z2/(4*n*n)) / denom
	r.Rate = &p
	r.Interval = &Interval{Low: math.Max(0, centre-half), High: math.Min(1, centre+half)}
	return r
}

// PassHat is pass^k: the chance that k of the cell's attempts, drawn without
// replacement, all pass.
type PassHat struct {
	K     int     `json:"k"`
	Value float64 `json:"value"`
}

// passHatK is pass^k for k of 1, 2, 4 and 8, as far as attempts reach, by
// τ-bench's estimator C(passed, k) / C(attempts, k).
func passHatK(passed, attempts int) []PassHat {
	var out []PassHat
	for _, k := range []int{1, 2, 4, 8} {
		if k > attempts {
			break
		}
		v := 1.0
		for i := range k {
			v *= float64(passed-i) / float64(attempts-i)
		}
		out = append(out, PassHat{K: k, Value: math.Max(0, v)})
	}
	return out
}

// spread is the variance a pooled rate p gives a pass or a failure: p(1-p),
// taken at its widest, one half, where p is nought or one and says nothing of
// the spread.
func spread(p float64) float64 {
	if p <= 0 || p >= 1 {
		p = 0.5
	}
	return p * (1 - p)
}

// minDetectable is the smallest difference between two pass rates, at a pooled
// rate p, that n1 and n2 attempts detect in a two-sided test at 5% with 80%
// power.
func minDetectable(n1, n2 int, p float64) float64 {
	if n1 <= 0 || n2 <= 0 {
		return 1
	}
	return (zAlpha + zBeta) * math.Sqrt(spread(p)*(1/float64(n1)+1/float64(n2)))
}

// repeatsNeeded is the attempts each of two cells needs, at a pooled rate p,
// to detect a difference of delta in pass rate by the same test.
func repeatsNeeded(delta, p float64) int {
	z := zAlpha + zBeta
	return int(math.Ceil(z * z * 2 * spread(p) / (delta * delta)))
}

// BTRating is one model's rating: the log of its Bradley–Terry strength, its
// bootstrapped 95% interval, and the models it is separable from (the
// interval of the two ratings' difference leaves out nought). A model
// separable from none is not separable.
type BTRating struct {
	Model         string    `json:"model"`
	Rating        float64   `json:"rating"`
	Interval      *Interval `json:"interval"`
	SeparableFrom []string  `json:"separable_from"`
	Separable     bool      `json:"separable"`
}

// btPrior is the regularisation: each model plays one win and one loss against
// a virtual model of strength one, so a model every judge puts first, or
// last, keeps a finite rating.
const btPrior = 1.0

// rateModels rates each model (a copy's cell) from the judges' rankings by a
// regularised Bradley–Terry fit, where every pair of copies of two models a
// judge ranked is one game the higher-ranked copy's model won, with intervals
// bootstrapped over judges and, within each model, over copies, seeded by
// seed.
func rateModels(rankings []JudgeRanking, seed [32]byte) []BTRating {
	models, copies := rankedModels(rankings)
	if len(models) == 0 {
		return nil
	}
	index := map[string]int{}
	for i, m := range models {
		index[m] = i
	}
	jw := make([]float64, len(rankings))
	for i := range jw {
		jw[i] = 1
	}
	cw := map[TrialKey]float64{}
	for _, c := range copies {
		cw[c] = 1
	}
	theta := fitBT(len(models), btGames(rankings, index, jw, cw))

	rng := rand.New(rand.NewChaCha8(seed))
	byModel := map[string][]TrialKey{}
	for _, c := range copies {
		byModel[c.Cell] = append(byModel[c.Cell], c)
	}
	draws := make([][]float64, 0, bootstrapDraws)
	for range bootstrapDraws {
		for i := range jw {
			jw[i] = 0
		}
		for range rankings {
			jw[rng.IntN(len(rankings))]++
		}
		for c := range cw {
			cw[c] = 0
		}
		for _, m := range models {
			cs := byModel[m]
			for range cs {
				cw[cs[rng.IntN(len(cs))]]++
			}
		}
		draws = append(draws, fitBT(len(models), btGames(rankings, index, jw, cw)))
	}
	out := make([]BTRating, len(models))
	for i, m := range models {
		col := make([]float64, len(draws))
		for d := range draws {
			col[d] = draws[d][i]
		}
		lo, hi := percentiles(col)
		out[i] = BTRating{Model: m, Rating: theta[i], Interval: &Interval{Low: math.Min(lo, theta[i]), High: math.Max(hi, theta[i])}, SeparableFrom: []string{}}
	}
	for i := range models {
		for j := range models {
			if i == j {
				continue
			}
			diff := make([]float64, len(draws))
			for d := range draws {
				diff[d] = draws[d][i] - draws[d][j]
			}
			if lo, hi := percentiles(diff); lo > 0 || hi < 0 {
				out[i].SeparableFrom = append(out[i].SeparableFrom, models[j])
			}
		}
		out[i].Separable = len(out[i].SeparableFrom) > 0
	}
	return out
}

// rankedModels is every model and every copy the rankings hold, sorted.
func rankedModels(rankings []JudgeRanking) ([]string, []TrialKey) {
	seenM, seenC := map[string]bool{}, map[TrialKey]bool{}
	var models []string
	var copies []TrialKey
	for _, r := range rankings {
		for _, c := range r.Ranked {
			if !seenM[c.Key.Cell] {
				seenM[c.Key.Cell] = true
				models = append(models, c.Key.Cell)
			}
			if !seenC[c.Key] {
				seenC[c.Key] = true
				copies = append(copies, c.Key)
			}
		}
	}
	slices.Sort(models)
	sort.Slice(copies, func(i, j int) bool { return copies[i].String() < copies[j].String() })
	return models, copies
}

// btGames is games[i][j], the weighted games model i won against model j:
// each pair of two models' copies one judge ranked, weighted by how often the
// judge and each copy were drawn.
func btGames(rankings []JudgeRanking, index map[string]int, jw []float64, cw map[TrialKey]float64) [][]float64 {
	n := len(index)
	games := make([][]float64, n)
	for i := range games {
		games[i] = make([]float64, n)
	}
	for r, rk := range rankings {
		if jw[r] == 0 {
			continue
		}
		for a := range rk.Ranked {
			for b := a + 1; b < len(rk.Ranked); b++ {
				hi, lo := rk.Ranked[a].Key, rk.Ranked[b].Key
				if hi.Cell == lo.Cell {
					continue
				}
				games[index[hi.Cell]][index[lo.Cell]] += jw[r] * cw[hi] * cw[lo]
			}
		}
	}
	return games
}

// fitBT is the log-strength of each of n models from games by Hunter's
// minorise–maximise iteration with btPrior's virtual games, centred so the
// models' mean is nought.
func fitBT(n int, games [][]float64) []float64 {
	p := make([]float64, n)
	for i := range p {
		p[i] = 1
	}
	next := make([]float64, n)
	for range 2000 {
		var moved float64
		for i := range n {
			wins := btPrior
			denom := 2 * btPrior / (p[i] + 1)
			for j := range n {
				if i == j {
					continue
				}
				wins += games[i][j]
				if g := games[i][j] + games[j][i]; g > 0 {
					denom += g / (p[i] + p[j])
				}
			}
			next[i] = wins / denom
			moved = math.Max(moved, math.Abs(math.Log(next[i])-math.Log(p[i])))
		}
		copy(p, next)
		if moved < 1e-10 {
			break
		}
	}
	theta := make([]float64, n)
	var mean float64
	for i := range p {
		theta[i] = math.Log(p[i])
		mean += theta[i]
	}
	mean /= float64(n)
	for i := range theta {
		theta[i] -= mean
	}
	return theta
}

// percentiles is the 2.5th and 97.5th percentiles of xs, which it sorts.
func percentiles(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return math.NaN(), math.NaN()
	}
	sort.Float64s(xs)
	at := func(q float64) float64 {
		i := int(math.Round(q * float64(len(xs)-1)))
		return xs[i]
	}
	return at(0.025), at(0.975)
}

// Agreement is how far the judges agree on one kind of judgement: over the
// units two judges or more judged, Krippendorff's alpha for nominal values
// with its bootstrapped 95% interval (nil where too few resamples could be
// estimated), and the share of pairs of their judgements that agree. Where
// alpha cannot be estimated (no unit judged twice, or every judgement the
// same), Alpha is nil and NotEstimable says why.
type Agreement struct {
	Units        int       `json:"units"`
	Alpha        *float64  `json:"alpha"`
	Interval     *Interval `json:"interval"`
	Percent      *float64  `json:"percent"`
	NotEstimable string    `json:"not_estimable,omitempty"`
}

// agreement is the judges' agreement on the good-enough marks (a unit is a
// copy, a value each judge's mark) and on the order of copies (a unit is a
// pair of copies two judges ranked, a value the copy each put higher),
// seeded by seed.
func agreement(rankings []JudgeRanking, seed [32]byte) (goodEnough, order Agreement) {
	marks := map[string][]string{}
	pairs := map[string][]string{}
	for _, r := range rankings {
		for a, c := range r.Ranked {
			v := "not good enough"
			if c.GoodEnough {
				v = "good enough"
			}
			marks[c.Key.String()] = append(marks[c.Key.String()], v)
			for b := a + 1; b < len(r.Ranked); b++ {
				x, y := c.Key.String(), r.Ranked[b].Key.String()
				unit := x + " " + y
				if y < x {
					unit = y + " " + x
				}
				pairs[unit] = append(pairs[unit], x)
			}
		}
	}
	rng := rand.New(rand.NewChaCha8(seed))
	return nominalAgreement(marks, rng), nominalAgreement(pairs, rng)
}

// nominalAgreement is the agreement over units, each the values its judges
// gave it; only a unit two judges or more judged counts.
func nominalAgreement(byUnit map[string][]string, rng *rand.Rand) Agreement {
	var units [][]string
	names := make([]string, 0, len(byUnit))
	for u := range byUnit {
		names = append(names, u)
	}
	slices.Sort(names)
	for _, u := range names {
		if vs := byUnit[u]; len(vs) >= 2 {
			units = append(units, vs)
		}
	}
	a := Agreement{Units: len(units)}
	if len(units) == 0 {
		a.NotEstimable = "no unit was judged by two judges or more"
		return a
	}
	agree, all := 0, 0
	for _, vs := range units {
		for i := range vs {
			for j := i + 1; j < len(vs); j++ {
				all++
				if vs[i] == vs[j] {
					agree++
				}
			}
		}
	}
	pct := float64(agree) / float64(all)
	a.Percent = &pct
	alpha, ok := krippendorff(units)
	if !ok {
		a.NotEstimable = "every judgement was the same, so the agreement expected by chance is nought and alpha is undefined"
		return a
	}
	a.Alpha = &alpha
	var draws []float64
	sample := make([][]string, len(units))
	for range bootstrapDraws {
		for i := range sample {
			sample[i] = units[rng.IntN(len(units))]
		}
		if v, ok := krippendorff(sample); ok {
			draws = append(draws, v)
		}
	}
	if len(draws) >= bootstrapDraws/2 {
		lo, hi := percentiles(draws)
		a.Interval = &Interval{Low: math.Min(lo, alpha), High: math.Max(hi, alpha)}
	}
	return a
}

// krippendorff is alpha for nominal values over units each judged twice or
// more, false where it is undefined (fewer than two values, or no
// disagreement expected by chance).
func krippendorff(units [][]string) (float64, bool) {
	o := map[[2]string]float64{}
	nc := map[string]float64{}
	var n float64
	for _, vs := range units {
		m := float64(len(vs))
		for i := range vs {
			for j := range vs {
				if i != j {
					o[[2]string{vs[i], vs[j]}] += 1 / (m - 1)
				}
			}
		}
	}
	for k, v := range o {
		nc[k[0]] += v
		n += v
	}
	if n <= 1 {
		return 0, false
	}
	var observed, expected float64
	for k, v := range o {
		if k[0] != k[1] {
			observed += v
		}
	}
	for c, x := range nc {
		for k, y := range nc {
			if c != k {
				expected += x * y
			}
		}
	}
	if expected == 0 {
		return 0, false
	}
	return 1 - (n-1)*observed/expected, true
}
