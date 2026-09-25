---
schema_version: 1
id: "iss-2609120447483698"
slug: "ahoy-install-the-scan-deep-true-false-prompt-sits-in-the-sam"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
resolution: "already fixed at tip: an out-of-set scan_deep answer refuses rather than disabling the scan (10c4ac01; resolved iss-2609090642031172)"
impact: internal
resolved_by:
  commit: "10c4ac01a6c85247383b1b07f8e4101752274f0c"
---

ahoy install: the scan_deep (true/false) prompt sits in the same stdin stream as the y/N approval prompts, so the documented 'yes | abcd ahoy install' form feeds it 'y', which is silently treated as the default (false). Either accept y/n on boolean config prompts, skip the prompt when --yes or the flag is given, or refuse an unparseable answer loudly. Observed: user chose 'approve everything', scan.deep landed as false with no warning.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._

## Grounds

- pursued: a piped yes can no longer silently turn deep scanning off; shown wrong if an unparseable scan_deep answer lands as false without a refusal
