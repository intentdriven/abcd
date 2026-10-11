---
schema_version: 1
id: "iss-176"
slug: "public-banlist-family-unenforceable"
severity: "major"
category: "inconsistency"
source: "impl-review"
found_during: "itd-74-increment-2-review"
found_at: ".abcd/docs-lint.json"
resolution: "already mitigated on HEAD: effectiveVisibilityEntries (internal/core/ahoy/gitignore.go, iss-255) narrows the public fence so that when the repo commits any .abcd/ files the wholesale /.abcd/ ignore is replaced by just .abcd/.work.local/ — so .abcd/docs-lint.json is tracked and CI-visible in a normal abcd-managed public repo. The scaffolder (stepBanlist, publicPathIsWritable) additionally withholds the public-family write where the path would be ignored, never writing a config abcd would immediately declare unenforceable. The premise (public family unenforceable under public visibility) no longer holds; no file move needed. Verified during the 2026-08-28 ledger-decision pass; the proposed relocation was built and then abandoned as unnecessary."
impact: internal
---

The public banned-names family lives at .abcd/docs-lint.json, which the installed .gitignore fence ignores under visibility public, so the layer that claims to be committed and CI-enforced is untracked exactly where public exposure is the risk

The two sides of the conflict are each defensible and both are in the record.
spc-20 places the public banlist "committed beside the docs-lint config"; the
iss-169 fence (brief §1) ignores the anchored `/.abcd/` wholesale under public
visibility, with no per-subdirectory exceptions, so one switch decides the whole
namespace. Under that fence a repo abcd configures as public carries a
`.abcd/docs-lint.json` git never tracks, and the family it holds reaches no CI
run anywhere.

itd-74 increment 2 does not resolve it: moving the file amends the iss-169
design record, and carving an exception into the fence gives up the
one-switch property that record chose deliberately. What it does instead is stop
claiming otherwise — detection reports `banlist.public_family_ignored`, and the
status board reads "public family NOT ENFORCEABLE (git ignores …, so CI never
sees it)".

Candidate reconciliations for a maintainer to pick between:

- Move the public banlist (and the docs-lint config generally) to a committed
  path outside `.abcd/`, accepting a second config location.
- Add the single un-ignore the public fence would need, accepting one
  per-subdirectory exception in a table that has none today.
- Keep the placement and declare the public layer private-visibility-only,
  making the private layer the answer for a public repo.

Since the second review round abcd also declines to CREATE the config where git
would ignore it: the absent-family gap is non-resolvable there and its fix hint
points here, because offering to write a file abcd would immediately report as
unenforceable is an offer that fixes nothing. A public-visibility repo therefore
has no public layer at all until this question is settled — which is the honest
state, and the reason it needs settling.

