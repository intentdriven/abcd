---
schema_version: 1
id: "iss-2610162030024268"
slug: "hand-back-sh-decides-whether-the"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "merge-queue failure of the cloud-sessions pull request"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/cloud/hand-back.sh"
remedy: "Match the line in the shell, case $'\\n'\"$out\" in *$'\\n'pre-push:*), with a test that makes the push print more than a pipe buffer of lines after the hook's refusal."
resolution: "hand-back.sh now matches the hook's refusal in the shell instead of printf piped into grep -q, and cuts only the output beneath the reason sentence; a stand-in hook refusing with 12000 lines drives both."
impact: internal
resolved_by:
  commit: "4e58ded3a"
---

hand-back.sh decides whether the committed pre-push hook refused a push with printf of the push output piped into grep -q '^pre-push:', under set -o pipefail. grep -q stops reading at its first match, so when the output runs to many lines past the pipe's buffer after that line, printf is killed by SIGPIPE, pipefail fails the condition, and the report's reason drops the words saying the hook refused it. A reproduction with 400KB of multi-line output missed five times in five; ordinary push output is far smaller, so no test drives it yet.
