# Dependencies

abcd has no required external tools. It ships as a single Go binary
([adr-21](../../decisions/adrs/0021-rebuild-in-go.md)) and runs standalone; every
capability abcd once delegated to a bundled tool now has a **native default**,
with the external tool available as an **opt-in adapter** over the same seam
([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)). A
present tool is an upgrade, never a prerequisite — a missing or misbehaving
adapter degrades to the native path rather than breaking abcd.

## Adapters over native defaults

- **Oracle / review** — the LLM is **host-delegated by default**
  ([adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)): abcd's core
  does the deterministic work, hands a prompt to the host's subagent dispatch,
  and consumes the structured result. Concrete oracle backends — **native, CLI,
  API, MCP** — are opt-in adapters behind one seam, selected when an operator
  wants abcd to reach a model directly. RepoPrompt and codex are two such
  adapters; a **direct API oracle is a legitimate adapter** too. There is no
  fixed RP→codex→in-session cascade.
- **Spec / task** — a **native minimal store** is the default
  ([adr-26](../../decisions/adrs/0026-native-spec-layer-ccpm-backend.md)):
  directories whose location encodes status, plus a dependency graph over them.
  the companion harness `ccpm` is the primary deeper backend at the convention level.
  **flow-next is not a dependency and is not built.**
- **Autonomous run** — a **pluggable `run` seam**
  ([adr-27](../../decisions/adrs/0027-autonomous-run-pluggable-seam.md)): Claude
  Workflows, the companion harness's agent loop, or a thin native Go loop behind one
  contract. Not a Ralph port; the native loop is the always-available fallback.
- **Transcript capture** — a **native local redacted store**
  ([adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)) is the
  default. specstory is an optional capture source, not a requirement.
- **Secret / PII scanning** — a **native Go scanner** is the default (see below);
  gitleaks and trufflehog are opt-in adapters that deepen the scan when wired.

## Secret / PII scanning

The launch preflight and the write-time redaction stages run a **native Go
secret/PII scan by default** — no external tool required. **gitleaks** (secret
scanning) and **trufflehog** (deep secret scan) are opt-in adapters: when wired
they run behind the same seam and harden the scan; when absent, the native
default still runs and still gates. Scanning never depends on a tool being
installed.

gitleaks is the one adapter wired. The scanner declares the seam it plugs into,
`scanner.Augmenter` (`Available() error` and `Scan(text, file) []Finding`),
and never imports gitleaks; the composition root (`cmd/abcd`) registers the
gitleaks augmenter, and every scanner the core builds picks it up for the
repository it scans. The opt-in is that repository's own
`.abcd/config/gitleaks.json`: absent or `enabled: false`, nothing is looked up
and nothing runs. Armed, gitleaks runs over every text the scanner reads —
the launch payload, transcripts, issue captures, memory pages, the privacy
lint's tracked files, a lifeboat's planned bytes — and its findings are appended
to the native ones, deduplicated on file, line and span. Its output is untrusted
input: a report is bounded in size and count, a finding is kept only when its
bytes sit at the line and column it names, and the scanner rebuilds every other
field itself, so a report or record carries the rule's kind and a masked
fingerprint, never the value in clear. The binary runs in the isolated child
environment every abcd subprocess gets, its own output discarded.

Armed with no gitleaks binary installed is one state with one consequence per
consumer. A release (`launch`) and a lifeboat (`disembark pack`) refuse on it,
the privacy lint reports it as an error, and the write paths — transcript
capture, issue capture, memory ingest — write with the native scanner and name
the gap in their receipt. A gitleaks run that fails, a report the scanner cannot
place, or a configured path the adapter refuses degrades the scanner instead:
every consumer treats that as it treats a broken `pii.json`.

## Plugin interop

abcd interoperates with peer tools — notably the companion harness
([adr-24](../../decisions/adrs/0024-companion-harness-peer-via-conventions-and-mcp.md)) —
over shared conventions and MCP, with **no code dependency in either
direction**. Interop is a capability, never a prerequisite: abcd runs fully with
no peer present.

## Out of scope

- **Cross-version schema migration** — abcd stamps `schema_version: 1`
  everywhere; migrators are added if a later intent changes the shape (itd-9,
  a draft).
