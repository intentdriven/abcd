---
schema_version: 1
id: "iss-316"
slug: "readme-process-section-asserts-two"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "README.md"
resolution: "README process section drops the unshipped 'skill' and 'Socratic interview' claims."
impact: fix
---

README process section asserts two unshipped capabilities a brief-drafting skill and a Socratic interview the framework provides
## Evidence
`README.md:77` (§ Process, present tense, unmarked): "abcd has a skill that ingests that material and produces a plain-language draft of your project's brief; the parts that feel fuzzy, you sharpen together with a Socratic interview the framework provides." Both clauses are unshipped:
- abcd ships ZERO skills by design (`brief/05-internals/08-skills.md`; `04-surfaces/README.md:34`; `docs/reference/terminology.md:47` REJECTS the skills format). No `skills/` dir; not in `launch-payload.json`. No command turns discovery material into a brief draft (`ingest` → sources corpus; `disembark`/`embark` → a repository).
- The Socratic interview is the grill sub-verb, `intents/planned/` (itd-27), "no grill sub-verb ships yet" (`04-surfaces/05-intent.md:198`); `abcd intent --help` lists audit/link/new/plan/ready.

## Adversarial verdict: CONFIRMED (major)
Two refuters, independently. It is the first instruction of the documented workflow ("It starts with the brief"), so a new reader asks their agent for a skill that does not exist. Severity major by the repo's own precedent for this class (iss-43, iss-181 documentation/major; iss-244 files the identical class on an internal record as major). Fix: reword to the human process that is true today, dropping both surface claims; the noun "skill" is forbidden for any abcd surface, and the ingest-to-brief-draft half has no backing intent (itd-90 is a different capability). Not prior art: iss-43 (resolved) closed three OTHER README claims and dropped its detector; this line was written 2026-08-16, 11 days after that closure. iss-181 records this exact class as knowingly ungated on README.
