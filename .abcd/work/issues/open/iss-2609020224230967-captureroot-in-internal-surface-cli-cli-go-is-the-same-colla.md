---
schema_version: 1
id: "iss-2609020224230967"
slug: "captureroot-in-internal-surface-cli-cli-go-is-the-same-colla"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

captureRoot (`internal/surface/cli/cli.go:3794`) is the same collapse the rules
root just had: it asks git for `rev-parse --show-toplevel` and falls back to cwd
on any error, so a repository git will not answer for (an ownership refusal under
the isolated env, git absent from the host's PATH, a corrupt `.git`) is treated as
no repository at all. The sibling three-state walk already exists as
`gitutil.RepoShapedRoot`; `capture.discoverRepoRoot` and `site.checkoutRoot` both
do this correctly. That much is unchanged and is still worth fixing.

RE-GRADED 2026-09-09, major -> minor, on reachability. The stated harm path is
GATED and the record did not say so.

The mechanism is real where it is reached: `internal/core/history/history.go:140`
calls `scanner.New(repoRoot)` with the root captureRoot returned, and the scanner
resolves the per-repo redaction override at `<root>/.abcd/config/pii.json`
without walking up — so a wrong root does silently redact with defaults only,
which is the B12 failure.

But the history verbs never reach it in the failure modes the record names.
`abcd history capture` (`cli.go:3557`) and `abcd history drain` (`cli.go:3693`)
each call `repoRootSHA()` FIRST and return its error, and `repoRootSHA`
(`cli.go:3822`) fails whenever `det.RootSHA == ""`. RootSHA comes from
`gitutil.RootCommit` (`internal/core/ahoy/store.go:61`), which runs
`rev-list -n 1 --max-parents=0 HEAD` through the SAME isolated-git environment
`gitutil.Run` uses (`internal/gitutil/repo.go:300`, `:345`). Every failure mode
the record names — the ownership refusal under `GIT_CONFIG_GLOBAL=/dev/null`,
git absent from PATH, a corrupt `.git` — fails both calls, so the verb refuses
before captureRoot is consulted. The asymmetry does not run the other way
either: `rev-list HEAD` is the stricter call, so there is no state where
`--show-toplevel` fails and RootSHA still resolves. The `hook session-start`
drain (`cli.go:1283-1284`) is gated identically, by `ahoy.Detect` succeeding AND
`det.RootSHA != ""`.

The one currently-ungated caller set is `internal/surface/cli/reading.go` — the
cold-reading verbs, at `:56` (`reading.Describe`), `:121` (`reading.Assemble`)
and `:237` (`reading.Ingest`). None is preceded by a root-SHA gate, and the
ingest path reaches a scanner the same way: `internal/core/reading/ingest.go:601`
calls `newPayloadField(repoRoot)`, which is `scanner.New(repoRoot)` at
`internal/core/reading/redact.go:51`. So the reachable exposure today is `abcd
reading ingest`/`assemble` run from a subdirectory of a repository git will not
answer for. That is the cold-reading workstream, which is not being carried here.

So: a real latent collapse, correctly identified, whose named harm path cannot
currently be walked, and whose one walkable path belongs to another workstream.
`minor` rather than `major`. The fix is unchanged and still wanted — three-state
through `gitutil.RepoShapedRoot` like the two siblings that already do — and it
should land before any new caller is added without a root-SHA gate in front of
it, because the gate is what is holding this closed, not the function.
