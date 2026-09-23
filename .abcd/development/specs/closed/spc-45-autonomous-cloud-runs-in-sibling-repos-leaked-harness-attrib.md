---
id: spc-45
slug: autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib
intent: itd-152
---
# autonomous-cloud-runs-in-sibling-repos-leaked-harness-attrib

## Summary

Adds a shared privacy-pattern class for two harness-leak shapes — a live session
URL and a harness-appended "Generated with …" attribution footer — and enforces
it on committed and stored text via `abcd audit` and docs-lint, and exposes it
for outbound artefacts (PR bodies, issue comments) as a primitive the posting
routine calls: abcd owns no forge client, so the outbound half gains no front
door of its own here. It also carries the operational half into the routine-prompt policy:
every autonomous routine prompt bans session URLs and harness footers in public
text and mandates a post-create re-read-and-strip of every PR, issue and comment
the loop creates, because the append happens outside the model's own text.

## Scope

In:

- `internal/adapter/scanner/patterns.go` — two new patterns in the canonical
  `DefaultPatterns()` set (session-URL, harness-footer), so they flow into every
  `ScanText`/`ScanBundle` consumer automatically.
- `internal/core/repolint/rule_privacy.go` — widen `abcd audit`'s privacy rule
  to iterate the new class over tracked files (today it reads only
  `NetworkPatterns()`).
- A new outbound-artefact scan seam: `scanner.New(repoRoot).ScanText(text,
  label)` + `scanner.Redact` + a `blockingResidual`-style fail-closed re-scan,
  callable before a PR/issue/comment is posted.
- The docs-lint path: thread the new class into `internal/core/lint` so posted
  text under the lint roots is flagged.
- The routine-prompt assembly used by the bughunt/autonomous-run scaffolding —
  the policy text and the mandated post-create strip step.

Out:

- The full secret set (`DefaultPatterns()` API keys, PATs) is **not** newly added
  to the audit privacy rule beyond the two footer/URL patterns — widening audit
  to every secret pattern is a separate behavioural change and is not in scope.
- No new forge client: abcd does not itself open PRs. The outbound scan is a
  primitive the routine calls, not a posting path abcd owns.

## Approach

**The canonical set is the scanner's `DefaultPatterns()`** (`patterns.go:41`),
already the source for `ScanText`/`ScanBundle` used by capture
(`capture/redact.go`), memory (`memory/redact.go`), history
(`history/history.go`) and launch (`launch/dryrun.go`). Two `Pattern` entries
(struct at `patterns.go:12`) are added:

- `harness_session_url` — matches the live-session URL shape the harness appends
  (a claude.ai session path), `Severity` high, with a `Skip`/`SkipAt` guard so a
  bare unrelated URL does not false-positive.
- `harness_attribution_footer` — matches the "Generated with …"/"Co-authored-by"
  harness footer block the append adds outside the model's own text.

Because these live in the canonical set, all four store-before-commit consumers
gain them for free; the fail-closed residual re-scan those paths already run
(`history.go:160`, `memory/redact.go:75`) treats a surviving match as blocking.

**Audit** (`repolint/rule_privacy.go`). `privacyHygiene.Eval` (rule_privacy.go:85)
today builds its pattern list from `sc.NetworkPatterns()` (rule_privacy.go:134)
and scans `gitutil.TrackedFiles`. The change appends the two new patterns to
that list (a small `SecretSubset()` / explicit selection so only these two, not
the whole secret set, join the network patterns), so `privacyLeak`
(rule_privacy.go:203) flags a footer or session URL in any committed file.

**docs-lint** (`internal/core/lint`). The two patterns are registered as banned
tokens via `textlint.go`'s `NewTokenChecker` (textlint.go:28) so committed/posted
prose under the lint roots is flagged with a clear message. This is the seam that
makes the class apply "in any committed or posted text, not only in freshly
created PR bodies".

**Outbound scan.** A new helper (e.g. `scanner`-backed `ScrubOutbound(repoRoot,
text)`) runs `ScanText` + `Redact` and returns the scrubbed text plus a
fail-closed error if any target pattern survives — the same
`ScanText → Redact → blockingResidual` shape capture/memory/history use. The
routine that holds the forge credentials calls it on every PR body and comment
before posting; abcd itself exposes no verb onto it, by the scope decision above,
so the primitive ships without a front door and the posting-time control stays
the re-read-and-strip policy below.

**Routine-prompt policy.** The autonomous-run prompt assembly gains a fixed
policy block: (a) never emit session URLs or harness footers in public text; (b)
after creating any PR, issue or comment, re-read it and strip the
harness-appended footer/URL, because the append lands outside the model's own
output. This mirrors the pattern already recorded in the user's operating notes
(strip the auto-appended footer post-write).

## How it satisfies each acceptance criterion

- *Outbound PR body with a harness footer is caught and stripped* — `ScrubOutbound`
  runs the footer pattern and `Redact` before posting. Test: feed a PR body with
  the footer, assert the returned text carries only the repo's `Assisted-by:`
  attribution and the footer is gone.
- *Outbound issue comment with a live session URL is caught, model- or
  harness-authored* — the `harness_session_url` pattern matches regardless of
  origin. Test: a comment containing a session URL fails the scan / is redacted.
- *Already-clean artefact passes unchanged* — no target pattern matches, so
  `ScrubOutbound` returns the input verbatim with no finding. Test asserts byte
  equality and empty findings.
- *The shared set is applied by `abcd audit` and docs-lint over committed/posted
  text* — the audit widening (rule_privacy.go) and the docs-lint token
  registration (textlint.go). Tests: a tracked file carrying a footer fails
  `audit`; a doc under the lint root carrying a session URL fails `docs-lint`.
- *An assembled routine prompt carries the policy* — the prompt-assembly change.
  Test: assemble a routine prompt for a managed repo and assert the policy block
  (ban + post-create strip) is present.

## Decisions

Only the two new shapes join the audit privacy rule, not the whole secret set:
the leak this closes is specifically the harness footer/URL, and widening audit
to every API-key pattern would change unrelated behaviour and risk noise on
legitimately redacted fixtures. The class is defined once, in the scanner's
canonical set, so the four store-before-commit paths, audit, docs-lint and the
outbound scan all read one definition rather than drifting copies.
