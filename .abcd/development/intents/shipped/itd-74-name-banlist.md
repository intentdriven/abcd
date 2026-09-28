---
id: itd-74
slug: name-banlist
spec_id: spc-20
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
impact: additive
---

# abcd Keeps the Names You Ban Out of Everything You Publish

## Press Release

> **Name a thing once as off-limits; abcd keeps it out of every published surface — and keeps the truly private ones off the machine's commits entirely.** A repo abcd configures often must not name certain things in what it publishes: a specific agent harness (so the surface stays host-agnostic), a partner's product, a *private* project whose very name is confidential, or — most sensitively — the user's own machine identifiers: hostnames, device names, and the IPs or address prefixes of their private network. abcd manages this as a two-layer banlist. The **public banlist** is enforced deterministically in CI: named tokens in user-facing content (README, `docs/`, the shipped artefact) fail the build, with a per-line escape hatch for the rare deliberate mention. The **private banlist** is enforced by a **local, untracked guard** — because a name that must never appear *anywhere public* cannot be written into public CI config to ban it there. Its patterns live only on the developer's machine; a pre-commit guard refuses to stage any content that matches, so the string never enters tracked history in the first place.
>
> "The names I can't afford to leak are exactly the ones I can't put in a public linter rule," said Kira, a maintainer. "abcd solved that by splitting it: public names get a CI gate, private names get a local guard whose list never leaves my machine. I stopped worrying that a stray paste would ship a name I'd promised to keep quiet."

## Why This Matters

Two failure modes share one root. First, a tool that claims to be *host-agnostic* undermines itself the moment its published docs name a specific harness — the naming dates the content and couples the surface. Second, and worse, a private collaborator's or project's name leaking into a public repo is a confidentiality breach that a history rewrite alone cannot fully undo (merged-PR diffs and cached views persist server-side). Both are cheap to prevent and expensive to remediate. The lesson learned the hard way on abcd's own repo: **prevent at authoring time, and never let the sensitive string reach a public artefact — including the linter config meant to catch it.**

The design tension is the interesting part: a deterministic CI gate is the right tool for *public* banned names, but it is the *wrong* place for a *private* one, because the rule would have to contain the very string it forbids. Splitting enforcement by sensitivity — public names in CI, private names in a local untracked guard — resolves it without compromise.

## What It Looks Like

- **`abcd` manages both layers as first-class config.** A public banlist (patterns + per-token severity + the allow-context escape) compiles into the deterministic docs-currency lint family; a private banlist is scaffolded as an untracked, per-machine file plus a committed guard hook that reads it. The literal private strings never enter tracked content or CI config.
- **Wired into install.** `abcd ahoy` scaffolds both surfaces for any repo it configures: the public family in the docs-lint config, the guard hook in the repo's committed hooks, and a gitignored banlist stub with instructions. A repo becomes name-safe by being abcd-managed, not by hand-rolling hooks.
- **Verbs to maintain it.** Add, list, and remove banned patterns; the public ones are visible and reviewable, the private ones addressed by reference (never printed into a shared artefact).
- **Reports what it cannot enforce.** The private guard is local by construction, so abcd states plainly that CI cannot enforce it — the guard protects the machine that opted in, and the public flip is gated on a from-scratch name scan.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Confirmed by the maintainer in the 2026-07-29 planning interview._

- **Given** a repo whose public banlist bans token X, **when** docs-lint runs over user-facing content (README, `docs/`, the shipped artefact) containing X without the allow escape, **then** a blocker finding names the file and line; **given** the line carries the escape, **then** no finding is raised.
- **Given** a private banlist entry with key K and pattern P, **when** a commit stages content matching P, **then** the commit is refused and the refusal message names only K — the matched string and the pattern value are never echoed to any output.
- **Given** a private banlist containing a hostname, an IP or CIDR prefix, or a device name, **when** staged content matches one, **then** the commit is refused exactly as for a name entry — machine identifiers are first-class private entries.
- **Given** a fresh clone where the private banlist file does not exist, **when** the guard hook runs, **then** it prints a loud warning that the private layer is inactive on this machine and does not block the commit — never a silent pass that looks like protection.
- **Given** a repo configured by `abcd ahoy`, **when** scaffolding completes, **then** the public family is present in the docs-lint config, the guard hook is committed, the private banlist stub exists and is gitignored, and the stub's seeded examples use only reserved documentation values (RFC 5737/3849/2606/7042 ranges, persona-derived hostnames) per the `examples-use-reserved-identifiers` principle.
- **Given** the banlist maintenance verbs (add, list, remove), **when** entries are rendered, **then** public entries render in full and private entries render by key only.
- **Given** any status or report surface describing the private layer, **when** it renders, **then** it states plainly that CI cannot enforce the private list — it protects only machines that have opted in.

## Resolved Questions (2026-07-29 planning interview)

- **Config shape: two files.** The public banlist is committed beside the docs-lint config; the private banlist is a separate gitignored file with its own tooling. A single split file makes the whole file's visibility load-bearing — accidental commit of the private half is the exact failure mode this intent exists to prevent.
- **Public-layer home: generalise the docs-lint family.** The existing `harness/*` banned-token family is lifted into a general banned-names family — one canonical primitive, ratchet not big-bang. A dedicated rule kind with richer reporting is future work, not this intent.
- **Private-stub seeding: reserved documentation values only.** The `examples-use-reserved-identifiers` principle answers the seeding question — the stub's examples come from RFC-reserved ranges and persona-derived hostnames, so the scaffold can never suggest a plausible real value.
- **Scope: machine identifiers included.** Hostnames, device names, and IP/CIDR entries are first-class private-banlist entries (evidence: iss-158, the 2026-07-29 managed-repo NEXT.md privacy-leak investigation).

## Dogfood (already running on this repo)

The concrete prototype this intent generalises is live in abcd-cli itself: a `harness/*` banned-token family in the docs-currency lint (public agent-harness names, blocker, README + `docs/` scanned), a committed `.githooks/pre-commit` guard that reads an untracked banlist for the private name, and the gitignored banlist that holds it. The feature is to lift that from a hand-wired arrangement into an abcd capability every managed repo inherits. Relates to the host-agnostic documentation principle and to the install surface (`ahoy`).

## Open Questions

_None — all three original questions resolved in the 2026-07-29 planning interview (see Resolved Questions)._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-3ceed52bdb99 -->
Fidelity review — receipt rcp-3ceed52bdb99 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:adb1f626f42968e7eed5c70958482350603309715288f8683540ecbcd62ef5c7
Input attestations: diff:internal/core/banlist, internal/core/ahoy/banlist_scaffold*.go, internal/surface/cli/banlist.go, .githooks/pre-commit, internal/core/lint/config.go and .abcd/docs-lint.json at chore/audit-run-a-1 5b4a43b6 (git ls-tree -r; spc-20 closed, itd-74 shipped)@sha256:cf94fb55003ac6ae6116bab8ea45d59a0f568e0de0a7e005bb9f2a4472190b8a;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: the banned_tokens family compiles per-token patterns with an allow_context escape and a blocker finding names file and line, as TestBannedTokens (bad.md:1) and TestDocsLintHarnessNameGate (docs/named.md:3, the docs-lint:allow line silent) hold; the concern is the reach: the configured roots are docs and README.md only, the payload render reuses those same roots, and the shipped artefact (commands/, agents/, skills/) is not a scanned root
  evidence: internal/core/lint/config.go:47 — "type BannedToken struct {"
  evidence: internal/core/lint/config.go:59 — "AllowContext []string `json:"allow_context"`"
  evidence: internal/core/lint/lint.go:279 — "findings = append(findings, tokenChecks.lintLines(rel, lines, mask)...)"
  evidence: internal/core/lint/lint_test.go:238 — "func TestBannedTokens(t *testing.T) {"
  evidence: internal/core/lint/lint_test.go:262 — "if !hasFinding(fs, filepath.Join("rec", "bad.md"), "py", 1) {"
  evidence: internal/core/lint/lint_test.go:289 — "if !hasFinding(fs, filepath.Join("docs", "named.md"), "harness/claude-code", 3) {"
  evidence: .abcd/docs-lint.json:2 — ""roots": [ "docs", "README.md" ]"
  evidence: internal/core/launch/gates.go:105 — "// Findings are the docs-lint findings over the configured doc roots."
- ac-2 — MET: the committed guard refuses a staged match naming only the entry key and withholds the matched text and the pattern by design; TestPreCommitHook_RefusesByKeyOnly asserts the key is printed and neither the pattern, its upper-case form nor the matched line appears
  evidence: .githooks/pre-commit:877 — "echo "pre-commit: BLOCKED — staged content matches private banlist entry '$key'." >&2"
  evidence: .githooks/pre-commit:886 — "(the matched text and the pattern are withheld by design — only the key is named)"
  evidence: internal/core/banlist/hook_test.go:195 — "func TestPreCommitHook_RefusesByKeyOnly(t *testing.T) {"
- ac-3 — MET: machine identifiers are ordinary keyed patterns in the store format, and the keyed corpus carries a hostname, an IPv4 address, a CIDR prefix, a MAC address and an IPv6 address that TestPreCommitHook_KeyedCorpus proves block exactly as the name entry does, naming the key only
  evidence: .githooks/pre-commit:27 — "hostnames, IPv4/IPv6 addresses, CIDR prefixes, MAC addresses,"
  evidence: internal/core/banlist/testdata/parse-corpus.txt:19 — "lab-host alice-laptop\.example\.com"
  evidence: internal/core/banlist/testdata/parse-corpus.txt:21 — "lab-cidr 203\.0\.113\.0/24"
  evidence: internal/core/banlist/hook_test.go:215 — "func TestPreCommitHook_KeyedCorpus(t *testing.T) {"
- ac-4 — MET: an absent store prints a starred WARNING that the private name guard is INACTIVE and the commit proceeds; an entryless store warns as loudly; both are pinned by TestPreCommitHook_AbsentBanlistWarnsLoudly and TestPreCommitHook_EntrylessStoreWarnsLoudly
  evidence: .githooks/pre-commit:462 — "echo "pre-commit: WARNING — the private name guard is INACTIVE on this machine." >&2"
  evidence: .githooks/pre-commit:681 — "echo "pre-commit: WARNING — the private name guard has NO ENTRIES in this store." >&2"
  evidence: internal/core/banlist/hook_test.go:155 — "func TestPreCommitHook_AbsentBanlistWarnsLoudly(t *testing.T) {"
  evidence: internal/core/banlist/hook_test.go:171 — "func TestPreCommitHook_EntrylessStoreWarnsLoudly(t *testing.T) {"
- ac-5 — MET_WITH_CONCERNS: Install writes the executable guard hook, a docs-lint config whose public family ListPublic reports present, and a keyed private stub with only commented examples (TestInstallScaffoldsTheBanlistArtefacts); the stub's values are judged reserved by the repo's own network detector with an armed RFC 1918 control (TestBanlistStubSeedsOnlyReservedIdentifiers) and real git ignores it (TestInstalledStubIsIgnoredByRealGit); the concern is that where the docs-lint config path is gitignored (the public-visibility case) no public family is written and the gap is reported unresolvable, so the scaffold leaves that repo with no CI-enforced family
  evidence: internal/core/ahoy/banlist_scaffold_test.go:19 — "func TestInstallScaffoldsTheBanlistArtefacts(t *testing.T) {"
  evidence: internal/core/ahoy/banlist_scaffold_test.go:73 — "func TestBanlistStubSeedsOnlyReservedIdentifiers(t *testing.T) {"
  evidence: internal/core/ahoy/banlist_scaffold_test.go:346 — "func TestInstalledStubIsIgnoredByRealGit(t *testing.T) {"
  evidence: internal/core/ahoy/banlist_scaffold.go:164 — "# lab-ipv4 192\.0\.2\.17"
  evidence: internal/core/ahoy/banlist_scaffold_test.go:495 — "t.Error("install wrote a docs-lint config into a path git ignores, delivering no enforcement")"
  evidence: internal/core/ahoy/banlist_scaffold_test.go:509 — "t.Error("the gap offers a fix that would produce a config CI never sees")"
- ac-6 — MET: renderPublicLayer prints id, severity, owner and the pattern in full; renderPrivateLayer prints each entry's key and line and states that pattern values never reach any output, and TestAddPrivateCreatesTheStoreAndListsKeysOnly asserts the marshalled private report carries no pattern
  evidence: internal/surface/cli/banlist.go:478 — "fmt.Fprintf(w, " %-32s %-8s %-13s %s\n", termsafe.Sanitize(e.ID), termsafe.Sanitize(e.Severity), owner, termsafe.Sanitize(e.Pattern))"
  evidence: internal/surface/cli/banlist.go:269 — "fmt.Fprintf(w, " %s (line %d)\n", termsafe.Sanitize(e.Key), e.Line)"
  evidence: internal/surface/cli/banlist.go:274 — "keys only: the pattern values never reach any output, by design"
  evidence: internal/core/banlist/private_test.go:225 — "func TestAddPrivateCreatesTheStoreAndListsKeysOnly(t *testing.T) {"
- ac-7 — MET: one PrivateReachNote constant states that CI cannot enforce the layer and it protects only opted-in machines; the banlist render, the ahoy status board, its JSON envelope and the scaffolded stub all derive from it, with tests on the board and the envelope
  evidence: internal/core/banlist/banlist.go:92 — "const PrivateReachNote = "CI cannot enforce this layer — it protects only machines that have opted in, " +"
  evidence: internal/surface/cli/banlist.go:298 — "fmt.Fprintln(w, " reach: "+banlist.PrivateReachNote)"
  evidence: internal/core/ahoy/banlist_scaffold.go:468 — "Reach: banlist.PrivateReachNote,"
  evidence: internal/surface/cli/ahoy_banlist_reach_test.go:20 — "func TestAhoyStatusStatesThePrivateBanlistReach(t *testing.T) {"
  evidence: internal/surface/cli/ahoy_banlist_reach_test.go:86 — "func TestAhoyEnvelopeCarriesTheReach(t *testing.T) {"

Gap audit:
- honoured:
  - two layers, public in CI config and private in an untracked store the committed guard reads
    evidence: .githooks/pre-commit:6 — "Refuses to commit any name listed in an UNTRACKED banlist."
    evidence: internal/core/banlist/banlist.go:157 — "func validPublicPattern(stored string) bool {"
  - a refusal names the key alone
    evidence: .githooks/pre-commit:886 — "only the key is named"
  - an absent private layer warns loudly and never impersonates protection
    evidence: .githooks/pre-commit:462 — "WARNING — the private name guard is INACTIVE on this machine."
  - ahoy scaffolds hook, public family and gitignored stub seeded with reserved values
    evidence: internal/core/ahoy/banlist_scaffold_test.go:19 — "func TestInstallScaffoldsTheBanlistArtefacts(t *testing.T) {"
  - every surface states what CI cannot enforce
    evidence: internal/core/banlist/banlist.go:92 — "CI cannot enforce this layer"
- diverged:
  - the public family gates README, docs/ and the shipped artefact
    evidence: .abcd/docs-lint.json:2 — ""roots": [ "docs", "README.md" ]"
    evidence: internal/core/launch/gates.go:105 — "the docs-lint findings over the configured doc roots"
  - a repo becomes name-safe by being abcd-managed: the public family is present after scaffolding
    evidence: internal/core/ahoy/banlist_scaffold_test.go:485 — "func TestPublicFamilyUnderPublicVisibility(t *testing.T) {"
    evidence: internal/core/ahoy/banlist_scaffold_test.go:509 — "the gap offers a fix that would produce a config CI never sees"
- missing: (none)
