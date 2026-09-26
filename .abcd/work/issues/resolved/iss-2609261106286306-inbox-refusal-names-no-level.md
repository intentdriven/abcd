---
schema_version: 1
id: "iss-2609261106286306"
slug: "inbox-refusal-names-no-level"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-cutfix item 2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report/inbox.go"
resolution: "The inbox refusal unwraps fsutil's *os.PathError and names the refused level, home-redacted, at exit 2 (TestAnInboxRefusalNamesTheLevelItRefused covers the home, ~/.abcd, the inbox and promoted/)."
impact: fix
resolved_by:
  commit: "7170858f8acd09be35106c551b243fae77600603"
---

The inbox's not-a-real-directory refusal names no level: errInboxNotRealDir (internal/core/report/inbox.go) says 'the inbox path is not a real directory' whatever level fsutil.EnsureRealDirAll refused — the home directory, ~/.abcd, ~/.abcd/inbox or ~/.abcd/inbox/promoted — and drops the *os.PathError path the earlier message carried. With ~/.abcd a dotfiles symlink, abcd report and abcd inbox promote say the inbox path is a symlink, the user looks at ~/.abcd/inbox, which does not exist, and is stuck. The refusal should name the refused level, home-redacted, and keep exit 2.

## Grounds

- pursued: a symlink at any level of the inbox is refused naming that level as ~/…; a refusal that names a level other than the one a symlink occupies, or carries the absolute home, would show it wrong
