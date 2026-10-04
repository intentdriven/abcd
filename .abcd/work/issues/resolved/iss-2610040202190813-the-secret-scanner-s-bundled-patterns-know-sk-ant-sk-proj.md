---
schema_version: 1
id: "iss-2610040202190813"
slug: "the-secret-scanner-s-bundled-patterns-know-sk-ant-sk-proj"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25, lane e8connect3fix (security review finding 2 of the guided connect)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/patterns.go"
remedy: "Add one bundled pattern for a plain sk- key to scanner.DefaultPatterns, after checking its exact form against a primary source (gitleaks' openai-api-key rule and the issuing services' published key formats) so a model name such as sk-tuned/7b is not taken for one; pin it with runtime-built fixtures (testsecret) in the scanner's tests and in TestGuideNeverEchoesATypedValueThatIsNotAName. It would be shown wrong by a committed file the new pattern flags that is not a key (run the payload scan over the tree first)."
resolution: "The canonical pattern set learns OpenRouter (sk-or-v1-, hard_fail), the legacy OpenAI shape (gitleaks openai-api-key, hard_fail) and a plain sk- key of 32 or more alphanumerics at warn, so every redactor masks it and the guided connect refuses it as a typed name, while lint and the launch scan cannot fail a committed file on it."
impact: fix
resolved_by:
  commit: "c6e34252829b0e34bba63e236b4f73810a35f868"
---

The secret scanner's bundled patterns know sk-ant-, sk-proj- and sk-svcacct- keys but no plain sk- key, the shape OpenAI's older keys and many OpenAI-compatible services issue (sk- followed by a long alphanumeric run). The guided connect refuses a typed value as a key only by those patterns (keyLines in internal/core/oracle/connect_guide.go), so a plain sk- key pasted where the guide asks for a model name still passes validModel and is printed in the command; every store-before-commit redactor has the same gap.

## Grounds

- pursued: a plain, OpenRouter or legacy sk- key typed where the guided connect asks a name is refused and never quoted back, and every store-before-commit redactor masks it; shown wrong by such a key reaching a record or a guide turn raw, or by a committed non-key the new rules flag (the tree sweep found none).
