package ahoy

import "github.com/intentdriven/abcd/internal/gitutil"

// scrubRemoteUserinfo drops any credential from a git remote URL before it
// enters RepoIdentity — the one value every registry sink and every JSON
// surface reads (GHSA-qc3w-8pv5-crc3). An operator who configured
// `https://user:token@github.com/owner/repo.git` as origin should not find the
// token at rest in ~/.abcd.noindex/history or on stdout, and scrubbing at the single
// derivation site means no downstream writer or renderer has to remember to.
//
// The rule distinguishes a credential from a route; it lives in gitutil
// (ScrubURLUserinfo), where `abcd land` shares it.
func scrubRemoteUserinfo(s string) string { return gitutil.ScrubURLUserinfo(s) }
