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
remedy: "Once the maintainer-gate record defines who maintains a repository, have the remote apply write .github/CODEOWNERS on a yes when no CODEOWNERS file exists, owning the publish-surface paths that exist under the maintainers' team or the maintainers; first check on a live repository whether a code-owner team member's own pull request needs another member's approval, and record the answer"
---

abcd ahoy remote apply (itd-2610290746077716) reports a missing or partial CODEOWNERS file as advisory and does not write it, because the file must name the repository's maintainers and who is a maintainer is defined by the maintainer-gate record iss-2610290730539347, which is not yet on the default branch. A team where the organization has one, otherwise the maintainers themselves; GitHub's admin role is not a maintainer by default. Unconfirmed: whether a pull request authored by a member of a code-owner team satisfies that team's code-owner rule without another member's approval; it must be checked on a live repository before the write is built, because a wrong answer makes every maintainer's own pull request wait for an approval. <!-- record-lint: forward-looking -->
