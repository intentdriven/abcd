package matrix

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// CapKey is the person's machine-wide spending cap for a matrix run with a
// metered cell (decision 17), read from ~/.abcd.noindex/config.json. A
// repository's .abcd/config.json may not set it: a committed file never sets
// anybody's spending.
const CapKey = "matrix.spend_cap"

// ResolvedCap is the cap a run spends under and where it came from: the run's
// own declaration, which overrides the person's machine-wide figure, or that
// figure. Both empty means no cap is named anywhere; abcd never supplies one.
type ResolvedCap struct {
	Cap    *Money
	Origin string // CapFromRun, CapFromMachine, or ""
}

// resolveCap resolves the run's spending cap from the declaration's own
// figure and the layered configuration at r. The matrix namespace is claimed,
// so a misspelt key under it is refused naming its file; a figure in the
// repository layer is refused naming its file; a machine figure that is not a
// positive amount in usd is refused naming its file.
func resolveCap(r layered.Roots, run *Money) (ResolvedCap, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return ResolvedCap{}, err
	}
	if err := s.Claim("matrix", "spend_cap"); err != nil {
		return ResolvedCap{}, err
	}
	found, err := s.Lookup(CapKey)
	if err != nil {
		return ResolvedCap{}, err
	}
	var machine *Money
	for _, f := range found {
		switch f.Layer {
		case layered.Repo:
			return ResolvedCap{}, fmt.Errorf("%s (repo layer): %s sets a spending cap, which only the person's own %s or the run's declaration may; a committed file never sets anybody's spending",
				f.Origin, CapKey, layered.Config.MachineOrigin())
		case layered.Machine:
			m, err := layered.Decode[Money](f.Raw)
			if err != nil {
				return ResolvedCap{}, fmt.Errorf("%s (machine layer): %s: %v", f.Origin, CapKey, err)
			}
			if err := m.validate(); err != nil {
				return ResolvedCap{}, fmt.Errorf("%s (machine layer): %s: %v", f.Origin, CapKey, err)
			}
			machine = &m
		}
	}
	if run != nil {
		c := *run
		return ResolvedCap{Cap: &c, Origin: CapFromRun}, nil
	}
	if machine != nil {
		return ResolvedCap{Cap: machine, Origin: CapFromMachine}, nil
	}
	return ResolvedCap{}, nil
}
