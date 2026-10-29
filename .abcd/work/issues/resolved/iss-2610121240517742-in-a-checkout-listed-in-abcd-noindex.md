---
schema_version: 1
id: "iss-2610121240517742"
slug: "in-a-checkout-listed-in-abcd-noindex"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/user-prompt-submit.sh"
remedy: "Give each non-guard shim the PreToolUse fall-back: run the dogfood build without exec, and when it exits with a failure code (or dies by a signal) run the same event through the installed release, with one stderr line naming the build that failed; add a shim test per event that a dying dogfood stub leaves the release's verb run once. If any event cannot be replayed safely (stdin already consumed, side effects half-done), say so in the brief instead of claiming the hooks never run weaker than the release."
resolution: "every non-guard hook shim falls back to the installed release when the dogfood build dies"
impact: fix
resolved_by:
  commit: "3b02a0d10133e93681f8dcd414e8a50233b125b0"
---

In a checkout listed in ~/.abcd.noindex/dogfood-roots, every hook shim but the PreToolUse one hands its event to the dogfood build with no fall-back to the installed release if that build dies. hooks/user-prompt-submit.sh, subagent-stop.sh, pre-compact.sh, post-tool-use.sh and session-end.sh replace the installed binary with the build that hook dogfood-binary names and exec it, and session-start.sh runs it in the release's place; a build that crashes, is killed or exits with an error leaves the rules loader, the router reset, the transcript capture or the session-start checks undone for that event, with nothing run in its stead. The brief's dogfood store section says a hook never runs weaker than the release, and only hooks/pre-tool-use.sh implements that, by handing the command to the installed release when the dogfood guard answers exit 1 or dies. Found in the round-2 security review of the dogfood remote fix (iss-2610070547057816).
