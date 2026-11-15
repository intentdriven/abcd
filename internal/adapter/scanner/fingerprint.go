package scanner

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// Fingerprint is a digest of everything this scanner was built from that
// decides what ScanText finds: the pattern set (bundled and per-repository),
// the identity it redacts and the identity severities, the skip and exclusion
// sets, the degraded state, and whether an augmenter runs and the gap one left.
// Two scanners with the same fingerprint, in one binary, find the same spans
// in the same text.
//
// It exists for a caller that keeps work done with one scanner and reuses it
// under another (the history store's redaction parts): a kept result is
// trusted only while the fingerprint is unchanged. A pattern's Skip and SkipAt
// functions are code, so only their presence is in the digest; the code itself
// changes with the binary, which a caller that keeps results across binaries
// fingerprints itself. The digest is a one-way hash, so the identity values in
// it are not recoverable from it.
func (s *Scanner) Fingerprint() string {
	type pattern struct {
		Name, Kind, Label, Re, Severity, Suggestion string
		Skip, SkipAt                                bool
	}
	type fingerprint struct {
		Patterns           []pattern
		Identity           Identity
		IdentitySeverities [][2]string
		SkipExtensions     []string
		SkipFilenames      []string
		RepoSkipExtensions []string
		RepoSkipFilenames  []string
		SkipFragments      []string
		Exclusions         []Exclusion
		Unavailable        bool
		UnavailableReason  string
		Augmented          bool
		AugmenterExplicit  bool
		AugmenterGap       string
	}
	f := fingerprint{
		Identity:           s.identity,
		SkipExtensions:     sortedKeys(s.skipExtensions),
		SkipFilenames:      sortedKeys(s.skipFilenames),
		RepoSkipExtensions: sortedKeys(s.repoSkipExtensions),
		RepoSkipFilenames:  sortedKeys(s.repoSkipFilenames),
		SkipFragments:      append([]string(nil), s.skipFragments...),
		Exclusions:         append([]Exclusion(nil), s.exclusions...),
		Unavailable:        s.unavailable,
		UnavailableReason:  s.unavailReason,
		Augmented:          s.aug != nil,
		AugmenterExplicit:  s.augExplicit,
		AugmenterGap:       s.AugmenterGap(),
	}
	for _, p := range s.patterns {
		re := ""
		if p.Re != nil {
			re = p.Re.String()
		}
		f.Patterns = append(f.Patterns, pattern{Name: p.Name, Kind: p.Kind, Label: p.Label, Re: re,
			Severity: string(p.Severity), Suggestion: p.Suggestion, Skip: p.Skip != nil, SkipAt: p.SkipAt != nil})
	}
	for _, k := range sortedKeys(s.identSev) {
		f.IdentitySeverities = append(f.IdentitySeverities, [2]string{k, string(s.identSev[k])})
	}
	f.Identity.OtherGitUserNames = sortedCopy(f.Identity.OtherGitUserNames)
	f.Identity.OtherGitUserEmails = sortedCopy(f.Identity.OtherGitUserEmails)
	data, err := json.Marshal(f)
	if err != nil {
		// Every field is a string, a bool or a slice of them; Marshal cannot
		// fail on it. Were it to, the fingerprint is unique to this call, so
		// it matches nothing a caller kept, nor any other call's.
		var nonce [16]byte
		_, _ = rand.Read(nonce[:])
		return "unfingerprinted:" + hex.EncodeToString(nonce[:])
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
