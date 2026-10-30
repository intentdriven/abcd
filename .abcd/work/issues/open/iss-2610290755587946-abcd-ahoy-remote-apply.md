---
schema_version: 1
id: "iss-2610290755587946"
slug: "abcd-ahoy-remote-apply"
severity: "minor"
category: "security"
source: "plan-review"
found_during: "planning itd-2610290746077716 (lane F, 2026-10-29)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ahoy remote apply (CODEOWNERS)"
remedy: "Once the maintainer-gate record settles whether a managed repository keeps a CODEOWNERS file, either have the remote apply write .github/CODEOWNERS on a yes, owning the publish-surface paths that exist, or retire the advisory CODEOWNERS finding; the record's ruling decides which"
---

abcd ahoy remote apply (itd-2610290746077716) reports a missing or partial CODEOWNERS file as advisory and does not write it. Whether a managed repository keeps a CODEOWNERS file at all is now part of the maintainer-gate record iss-2610290730539347, which is not yet on the default branch. A maintainer is anyone with Write access or above, and GitHub counts approvals only from those, so the approval gate does not depend on CODEOWNERS naming a team. <!-- record-lint: forward-looking -->
