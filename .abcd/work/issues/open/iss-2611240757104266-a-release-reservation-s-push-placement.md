---
schema_version: 1
id: "iss-2611240757104266"
slug: "a-release-reservation-s-push-placement"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/withdrawpush.go"
remedy: "Place every push of one fetch with a single walk of the first-parent history (one rev-list over the deepest tip, matched against each pushed commit) instead of one git process per push, so the cost is one walk, not one per push."
---

A release reservation's push placement runs one git rev-list per push under the one fetch-wide pusherBudget (ten seconds; place in internal/core/claim/withdrawpush.go), so a release branch whose last several hundred commits each arrived as their own push can outrun the budget on a slow machine. It fails safe: every unplaced push reads unverified, with the reason named. The claim tests' walk cases hit it on the macOS runners until their TestMain widened the budget.
