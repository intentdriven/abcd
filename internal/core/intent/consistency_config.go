package intent

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// consistency_config.go — the request limit of the consistency pass
// (itd-2609291923559186, A4). The emit measures the assembled corpus against
// it, and reviews a corpus larger than it in chunks.

// The request limit's bundled default and its bounds, in bytes of the corpus
// file. The upper bound is the guarded-read cap a provider dispatch puts on
// every input file it sends (1 MiB): a corpus file above it cannot be sent at
// all. The lower bound keeps a chunk large enough to hold more than a page or
// two beside its framing.
const (
	DefaultConsistencyRequestBytes = 512 << 10
	MinConsistencyRequestBytes     = 64 << 10
	MaxConsistencyRequestBytes     = 1 << 20
)

// ConsistencyConfig is the resolved consistency configuration, with where the
// limit came from (the layered resolver's origin: "bundled" or a file).
type ConsistencyConfig struct {
	MaxRequestBytes       int    `json:"max_request_bytes"`
	MaxRequestBytesOrigin string `json:"max_request_bytes_origin"`
}

// LoadConsistencyConfig resolves consistency.max_request_bytes through the one
// layered configuration reader (.abcd/config.json, then ~/.abcd.noindex/config.json,
// then the bundled 512 KiB). It claims the `consistency` namespace, with
// on_finding (the pre-commit gate's setting, read by the gate) beside the
// limit, so a misspelt key is refused rather than letting the default apply
// unannounced, and a limit outside 64 KiB to 1 MiB is refused naming its file.
func LoadConsistencyConfig(r layered.Roots) (ConsistencyConfig, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return ConsistencyConfig{}, err
	}
	if err := s.Claim("consistency", "max_request_bytes", "on_finding"); err != nil {
		return ConsistencyConfig{}, err
	}
	v, err := layered.Get(s, "consistency.max_request_bytes", DefaultConsistencyRequestBytes, func(n int) error {
		if n < MinConsistencyRequestBytes || n > MaxConsistencyRequestBytes {
			return fmt.Errorf("want a byte count from %d (64 KiB) to %d (1 MiB, the most a provider dispatch sends)",
				MinConsistencyRequestBytes, MaxConsistencyRequestBytes)
		}
		return nil
	})
	if err != nil {
		return ConsistencyConfig{}, err
	}
	return ConsistencyConfig{MaxRequestBytes: v.V, MaxRequestBytesOrigin: v.Origin}, nil
}
