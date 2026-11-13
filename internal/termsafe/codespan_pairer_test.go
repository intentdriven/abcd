package termsafe

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// TestCodeSpanPairerPairsAsPairCodeSpan: walked left to right, the pairer
// pairs every run of a string exactly as PairCodeSpan does, over strings drawn
// from backticks, text and line breaks.
func TestCodeSpanPairerPairsAsPairCodeSpan(t *testing.T) {
	alphabet := []string{"`", "``", "```", "a", " ", "\n"}
	rng := rand.New(rand.NewPCG(3, 4))
	for n := 0; n < 2000; n++ {
		var b strings.Builder
		for k := rng.IntN(24); k >= 0; k-- {
			b.WriteString(alphabet[rng.IntN(len(alphabet))])
		}
		s := b.String()
		p := NewCodeSpanPairer(s)
		for i := 0; i < len(s); {
			if s[i] != '`' {
				i++
				continue
			}
			end := backtickRunEnd(s, i)
			want, wok := PairCodeSpan(s, i)
			got, gok := p.Pair(i, end)
			if gok != wok || got != want {
				t.Fatalf("over %q at %d: pairer = %+v %v, PairCodeSpan = %+v %v", s, i, got, gok, want, wok)
			}
			if gok {
				i = got.End
			} else {
				i = end
			}
		}
	}
}

// TestCodeSpanPairerOpensAnEscapedRunPastItsFirstBacktick: an opener that
// starts one backtick into a run closes on a later run of its own length.
func TestCodeSpanPairerOpensAnEscapedRunPastItsFirstBacktick(t *testing.T) {
	s := "``a` b"
	sp, ok := NewCodeSpanPairer(s).Pair(1, 2)
	if !ok || sp.Start != 1 || sp.End != 4 || sp.Raw(s) != "a" {
		t.Fatalf("Pair(1, 2) over %q = %+v %v, want the span `a`", s, sp, ok)
	}
}

// TestCodeSpanPairerStepsInLineWithTheString: pairing every run of a string
// of runs that never close costs steps in line with the string, so a string
// four times longer takes at most about four times the steps. A scan from each
// run to the end of the string takes about eight times as many here.
func TestCodeSpanPairerStepsInLineWithTheString(t *testing.T) {
	count := func(units int) (int, int) {
		// Runs of every length from 1 to units, each once: none closes, and
		// the string's length grows with the square of units.
		var b strings.Builder
		for k := 1; k <= units; k++ {
			b.WriteString(strings.Repeat("`", k))
			b.WriteString(" a ")
		}
		s := b.String()
		p := NewCodeSpanPairer(s)
		for i := 0; i < len(s); {
			if s[i] != '`' {
				i++
				continue
			}
			end := backtickRunEnd(s, i)
			if sp, ok := p.Pair(i, end); ok {
				i = sp.End
			} else {
				i = end
			}
		}
		return p.Steps(), len(s)
	}
	small, ns := count(100)
	large, nl := count(200)
	if nl < 4*ns-4*200 || small == 0 || large > 5*small {
		t.Fatalf("pairing took %d steps over %d bytes and %d over %d, want at most five times as many", small, ns, large, nl)
	}
}
