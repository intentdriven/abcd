---
schema_version: 1
id: "iss-2609290405451613"
slug: "the-release-cut-accepts-a-third-party-s-home-path-and-any"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release/ingest.go"
remedy: "Waits on ruling BC; in checkPrivacy (internal/core/release/ingest.go): if (a), keep the launch bar, state it in the brief's release chapter and pin a test that a home_path_other line is accepted; if (b), refuse every identity kind the store-before-commit redactors refuse, the owner's bare name included; if (c), refuse home_path_other and admit github_username. Any refusal names the kind and the line, never the path, proven by a test on a changelog payload carrying a third-party home path."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (rulings-owed BC; lane reanchor, run A 2026-09-29): which bar public release text is held to, the launch bar (hard_fail only, as checkPrivacy in internal/core/release/ingest.go applies it now), the store-before-commit bar (every identity kind, refusing the repository owner's bare name as well), or a middle bar (third-party home paths refused, usernames allowed)"
---

The release cut accepts a third party's home path, and any other identity finding the scanner grades warn, in public release text. After the hard_fail refusal of iss-2609290405381338, a changelog entries[].text or press_release headline carrying another account's home path (home_path_other) or a GitHub username (github_username) is still written to CHANGELOG.md and RELEASE.md as it stands, because the cut refuses at the bar the launch scan applies to the same files, hard_fail only. Refusing these too is a policy choice, not a mechanical fix: the store-before-commit redactors (history, memory, ideate) redact every identity kind and refuse any that survives, whatever its severity, but github_username fires on the repository owner's name standing as a bare word (a URL and the owner/repo slug are exempt), which a release note may legitimately carry, and the scanner grades that kind review-only, warn, for that reason. Ruling owed to the product thinker: whether public release text is held to the launch bar (hard_fail only, as now), to the store-before-commit bar (every identity kind, accepting refusals of the owner's bare name), or to a middle bar (third-party home paths refused, usernames allowed). Found by the security review of lane drainEcho2 and split from iss-2609290405381338 by lane drainEcho3. Detector, once ruled: a payload carrying a third-party home path in a changelog line is refused or accepted as the ruling says, and the refusal does not carry the path.

## Remedy grounds (2026-09-29)

- Why: ruling BC's three bars differ only in which scanner kinds checkPrivacy treats as refusals, so each answer is one filter change and one test; the ruling is unanswered and none is picked. The never-echo clause holds under every answer, since a refusal that printed the path would leak it to the terminal and the log.
- Rejected: a per-line escape for release text, which would add a way past the refusal rather than decide its bar.
