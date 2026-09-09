---
schema_version: 1
id: "iss-2609090951279243"
slug: "manifest-quote-path-fields-get-only-the-git-refusal"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/manifest.go"
---

The site manifest now holds its two whole-file page sources to a closed allowlist of the documentation and site-source roots, on the stated ground that a denylist of the git directory and the environment file leaves the gitignored local tier, the private record and every file a future contributor adds still reachable. The quote-type path fields did not get that treatment: identity.file, ui_strings, the deferred feature quote paths, the contributors policy file and the unresolved-reference baseline each pass the relative-path check and then only the git-directory refusal, so a committed manifest naming a dotenv file or a path under the gitignored local tier loads without complaint. The policy file is the one that renders: when its part is anything other than first-bullet, policyQuote renders the ENTIRE matched section of the named file, not one bullet, into the published contributors page, so a manifest pointing it at a local scratch file with a matching heading publishes that file verbatim to the public site. Verified by reading the five call sites and policyQuote. It matters because the manifest travels with a clone and the site is public, so the weaker gate is the one an unreviewed manifest edit reaches first, and the closed set was adopted next door precisely because a denylist cannot anticipate the next reachable file. Fix direction: put every path field the build reads through the same closed page-root set the whole-file sources use, or state per field why a wider set is safe there. Detector: a manifest whose contributors policy file names a path outside the page roots must be refused at load, while the committed manifest still loads.
