package matrix

import (
	"fmt"
	"math"
	"testing"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// pass@1 is the pass rate with a Wilson interval, and pass^k the chance that k
// attempts drawn without replacement all pass (τ-bench's estimator,
// C(c,k)/C(n,k)), given for every k from 1 to 8 the attempts reach.
func TestPassAt1HasAnIntervalAndPassHatKIsTheAllPassEstimator(t *testing.T) {
	r := passAt1(7, 10)
	if r.Rate == nil || !near(*r.Rate, 0.7, 1e-12) || r.Interval == nil {
		t.Fatalf("pass@1 of 7 in 10 = %+v; want 0.7 with an interval", r)
	}
	// Wilson's 95% interval for 7 of 10.
	if !near(r.Interval.Low, 0.3968, 1e-3) || !near(r.Interval.High, 0.8922, 1e-3) {
		t.Fatalf("pass@1 interval = %+v; want Wilson's (0.397, 0.892)", *r.Interval)
	}
	if z := passAt1(0, 0); z.Rate != nil || z.Interval != nil {
		t.Fatalf("pass@1 of no attempt = %+v; want no rate and no interval", z)
	}
	if all := passAt1(5, 5); all.Interval.High != 1 || all.Interval.Low >= 1 {
		t.Fatalf("pass@1 of 5 in 5 = %+v; want an interval below 1 up to 1", all)
	}
	hk := passHatK(7, 10)
	want := map[int]float64{1: 0.7, 2: 21.0 / 45, 4: 35.0 / 210, 8: 0}
	if len(hk) != len(want) {
		t.Fatalf("pass^k = %+v; want k of 1, 2, 4 and 8", hk)
	}
	for _, h := range hk {
		if !near(h.Value, want[h.K], 1e-12) {
			t.Errorf("pass^%d = %v; want %v", h.K, h.Value, want[h.K])
		}
	}
	if got := passHatK(2, 3); len(got) != 2 || got[1].K != 2 || !near(got[1].Value, 1.0/3, 1e-12) {
		t.Errorf("pass^k of 2 in 3 = %+v; want k of 1 and 2 only, pass^2 a third", got)
	}
}

// The smallest difference two cells' pass rates could show at their repeats,
// and the repeats a declared difference needs, come from one two-sided test at
// 5% with 80% power.
func TestThePowerEstimateGivesRepeatsNeededAndTheSmallestDetectableDifference(t *testing.T) {
	// 2 × (1.96 + 0.84)^2 × 0.25 / 0.2^2 ≈ 98.1.
	if n := repeatsNeeded(0.2, 0.5); n != 99 {
		t.Fatalf("repeats for a difference of 0.2 at 0.5 = %d; want 99", n)
	}
	// A difference too small for any run asks for one past the most a cell
	// may be sized to, never a count an int cannot hold.
	for _, tiny := range []float64{1e-6, 1e-200, 5e-324} {
		if n := repeatsNeeded(tiny, 0.5); n != MaxRepeats+1 {
			t.Fatalf("repeats for a difference of %v = %d; want %d", tiny, n, MaxRepeats+1)
		}
	}
	if d := minDetectable(99, 99, 0.5); !near(d, 0.199, 2e-3) {
		t.Fatalf("the smallest difference at 99 each = %v; want about 0.2", d)
	}
	if d := minDetectable(5, 5, 0.5); d <= 0.5 {
		t.Fatalf("the smallest difference at 5 each = %v; want it far past 0.5", d)
	}
	// A pooled rate of nought or one says nothing of the spread: the
	// estimate takes the widest, at one half.
	if minDetectable(10, 10, 1) != minDetectable(10, 10, 0.5) {
		t.Fatal("a pooled rate of one does not take the widest spread")
	}
}

// writingRanking is one judge's ranking of copies, best first, each named
// <cell>/<repeat> of task doc-page, every copy marked good enough when good
// names it.
func writingRanking(judge string, good map[string]bool, copies ...string) JudgeRanking {
	r := JudgeRanking{Judge: judge}
	for _, c := range copies {
		var cell string
		var repeat int
		if _, err := fmt.Sscanf(c, "%1s/%d", &cell, &repeat); err != nil {
			panic(err)
		}
		r.Ranked = append(r.Ranked, RankedCopy{Key: TrialKey{Task: "doc-page", Cell: "cell-" + cell, Repeat: repeat}, GoodEnough: good[c], Words: 100})
	}
	return r
}

// Ratings come from a regularised Bradley–Terry fit of every judge's ranking,
// with intervals bootstrapped over judges and copies: a model every judge puts
// first is rated above and separable from the one every judge puts last, while
// two models the judges split on evenly are not separable.
func TestBradleyTerryRatesModelsWithBootstrappedIntervalsAndSaysNotSeparable(t *testing.T) {
	var rankings []JudgeRanking
	for j := range 6 {
		judge := fmt.Sprintf("j%d", j+1)
		mid := []string{"b/1", "c/1", "c/2", "b/2"}
		if j%2 == 1 {
			mid = []string{"c/1", "b/1", "b/2", "c/2"}
		}
		rankings = append(rankings, writingRanking(judge, nil, append(append([]string{"a/1", "a/2"}, mid...), "d/1", "d/2")...))
	}
	ratings := rateModels(rankings, [32]byte{9})
	by := map[string]BTRating{}
	for _, r := range ratings {
		if r.Interval == nil || r.Interval.Low > r.Rating || r.Interval.High < r.Rating {
			t.Fatalf("model %s's rating %v carries no interval around it: %+v", r.Model, r.Rating, r.Interval)
		}
		by[r.Model] = r
	}
	if len(by) != 4 {
		t.Fatalf("ratings = %+v; want one per model", ratings)
	}
	a, b, c, d := by["cell-a"], by["cell-b"], by["cell-c"], by["cell-d"]
	if !(a.Rating > b.Rating && b.Rating > d.Rating && c.Rating > d.Rating) {
		t.Fatalf("ratings a %v, b %v, c %v, d %v; want a first and d last", a.Rating, b.Rating, c.Rating, d.Rating)
	}
	if !contains(a.SeparableFrom, "cell-d") || !a.Separable {
		t.Errorf("a = %+v; want it separable from d", a)
	}
	if contains(b.SeparableFrom, "cell-c") || contains(c.SeparableFrom, "cell-b") {
		t.Errorf("b = %+v, c = %+v; the judges split evenly on them, so they are not separable", b, c)
	}
	// Regularised: a model every judge puts first still has a finite rating.
	if math.IsInf(a.Rating, 0) || a.Rating > 10 {
		t.Errorf("a's rating = %v; want the regularised fit to keep it finite", a.Rating)
	}

	// Two models no judge tells apart are reported as not separable.
	even := []JudgeRanking{
		writingRanking("j1", nil, "a/1", "b/1"),
		writingRanking("j2", nil, "b/1", "a/1"),
		writingRanking("j3", nil, "a/2", "b/2"),
		writingRanking("j4", nil, "b/2", "a/2"),
	}
	for _, r := range rateModels(even, [32]byte{9}) {
		if r.Separable || len(r.SeparableFrom) != 0 {
			t.Errorf("model %s = %+v; want it not separable", r.Model, r)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// Agreement is Krippendorff's alpha with a bootstrapped interval and plain
// percentage agreement, on the good-enough marks and on the order of every
// pair of copies two judges ranked; where no unit is judged twice, or every
// mark is the same, it is not estimable, said so, never zero.
func TestAgreementIsAlphaWithAnIntervalOrNotEstimable(t *testing.T) {
	good := map[string]bool{"a/1": true, "b/1": true}
	var same []JudgeRanking
	for j := range 4 {
		same = append(same, writingRanking(fmt.Sprintf("j%d", j+1), good, "a/1", "b/1", "c/1", "d/1"))
	}
	ge, order := agreement(same, [32]byte{3})
	for name, a := range map[string]Agreement{"good enough": ge, "order": order} {
		if a.Alpha == nil || !near(*a.Alpha, 1, 1e-12) || a.Percent == nil || !near(*a.Percent, 1, 1e-12) {
			t.Errorf("%s agreement of four identical rankings = %+v; want alpha 1 and full agreement", name, a)
		}
		if a.Interval == nil {
			t.Errorf("%s agreement carries no interval: %+v", name, a)
		}
	}

	// Judges who split: alpha below zero's neighbourhood of one.
	split := []JudgeRanking{
		writingRanking("j1", map[string]bool{"a/1": true}, "a/1", "b/1", "c/1"),
		writingRanking("j2", map[string]bool{"b/1": true}, "c/1", "b/1", "a/1"),
		writingRanking("j3", map[string]bool{"c/1": true}, "b/1", "a/1", "c/1"),
	}
	ge, order = agreement(split, [32]byte{3})
	if ge.Alpha == nil || *ge.Alpha >= 0.5 || order.Alpha == nil || *order.Alpha >= 0.5 {
		t.Errorf("split judges' agreement = %+v / %+v; want alpha well below one", ge, order)
	}

	// No copy judged twice: not estimable.
	apart := []JudgeRanking{
		writingRanking("j1", good, "a/1", "b/1"),
		writingRanking("j2", good, "c/1", "d/1"),
	}
	ge, order = agreement(apart, [32]byte{3})
	for name, a := range map[string]Agreement{"good enough": ge, "order": order} {
		if a.Alpha != nil || a.NotEstimable == "" {
			t.Errorf("%s agreement with no unit judged twice = %+v; want it not estimable, saying why", name, a)
		}
	}
	// Every mark the same: alpha is not estimable, though the judges agree.
	allGood := map[string]bool{"a/1": true, "b/1": true}
	ge, _ = agreement([]JudgeRanking{writingRanking("j1", allGood, "a/1", "b/1"), writingRanking("j2", allGood, "b/1", "a/1")}, [32]byte{3})
	if ge.Alpha != nil || ge.NotEstimable == "" || ge.Percent == nil || *ge.Percent != 1 {
		t.Errorf("agreement when every mark is the same = %+v; want alpha not estimable and full percentage agreement", ge)
	}
}

// Alpha read twice from the same marks is the same to the last bit: its sums
// never follow a map's order, so a report read twice says the same thing even
// where units are judged by three, four or more judges, whose terms are
// thirds and sixths.
func TestKrippendorffsAlphaIsTheSameEveryTimeItIsRead(t *testing.T) {
	cats := []string{"a", "b", "c", "d", "e", "f", "g"}
	var units [][]string
	for u := range 40 {
		n := 3 + u%5
		var vs []string
		for j := range n {
			vs = append(vs, cats[(u*7+j*j*3+j)%len(cats)])
		}
		units = append(units, vs)
	}
	first, ok := krippendorff(units)
	if !ok {
		t.Fatal("alpha is not estimable for the marks")
	}
	for i := range 500 {
		if got, _ := krippendorff(units); math.Float64bits(got) != math.Float64bits(first) {
			t.Fatalf("read %d, alpha = %v; read first, %v: the same marks give another alpha", i+2, got, first)
		}
	}
}
