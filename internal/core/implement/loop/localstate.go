package loop

import (
	"errors"

	"github.com/intentdriven/abcd/internal/localtier"
)

// tierRefusal is the loop's refusal, at stage, of a local-tier read
// localtier refused: state git tracks came from whoever made the commit, not
// this checkout. State git cannot say it does not track is refused at the
// config stage instead, where the loop says the repository's git
// configuration cannot be read: a git that cannot answer for the checkout is
// that. It is nil for any other error, which the reader reports as its own.
func tierRefusal(stage, lane string, err error) *Refusal {
	var r *localtier.Refused
	if !errors.As(err, &r) {
		return nil
	}
	if r.Cause != nil {
		return refuse(StageConfig, "", lane, "the repository's git configuration cannot be read, so "+r.Reason(), r.Remedy())
	}
	return refuse(stage, "", lane, r.Reason(), r.Remedy())
}
