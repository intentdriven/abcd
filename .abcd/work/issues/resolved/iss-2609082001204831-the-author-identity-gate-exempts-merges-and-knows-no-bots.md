---
schema_version: 1
id: "iss-2609082001204831"
slug: "the-author-identity-gate-exempts-merges-and-knows-no-bots"
severity: "major"
category: "process"
source: "user-observation"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-attribution.sh"
resolution: "The commits arm now reads the author and committer of every commit in the range, merges included, and refuses a machine identity on the forge's structural markers while a human at a forge privacy address passes."
impact: internal
---

The attribution gate refuses an AI git identity by name and by vendor address (AI_IDENT_NAME_RE, AI_IDENT_MAIL_RE), which closes the route AGENTS.md cares about: the contributor graph is built from commit authorship, so an AI author puts a tool in the graph and a squash merge then re-appends it as a co-author. Two holes let four commits through anyway. First, the commits arm walks every NON-MERGE commit in the range, so a merge commit's identity is never read; commit 23f0a891 is authored and committed as Claude at a vendor noreply address and stands in main's history today, arriving from an autonomous bug-hunt round whose routine ran with the tool's own git identity. Second, a machine identity that is not an AI vendor matches neither list: three dependabot[bot] commits author dependency bumps directly. The maintainer's rule is that a commit is authored by a human and machine assistance is disclosed by the Assisted-by trailer alone, so the gate's shape is right and its coverage is not. The rule to enforce is refuse-machines-allow-humans, not an allowlist of one name: the repository takes outside contributions through .abcd/work/intake.md and carries one such commit already, and an allowlist would refuse the next one. Fix: read the identity of every commit in the range including merges, and refuse a machine author on a structural signal — the [bot] name suffix the forge stamps, the vendor noreply addresses already enumerated, and the forge bot address shape — while leaving any human author to pass. Detector: a range containing a merge commit authored by an AI identity must fail the gate, a range containing a bot-suffixed author must fail it, a range whose commits are authored by two different humans must pass, and the repository's own history from the gate's introduction forward must still pass. Note the consequence to state in the fix, not to solve here: dependabot pull requests will stop being mergeable as authored, so a dependency bump has to be landed by a human.

## Grounds

- pursued: the [bot] marker in a name or mailbox, the vendor noreply domains, and a noreply mailbox in the author role together name every machine that lands a commit here while naming no human — shown wrong by a real contributor the gate refuses (a person whose forge name ends [bot], or a human-authored commit refused for its committer), or by a machine landing a commit carrying none of those markers.
