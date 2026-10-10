---
schema_version: 1
id: "iss-164"
slug: "the-ahoy-install-completion-summary-is"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "marketplace-install smoke test"
found_at: "internal/core/ahoy"
blocked_by: [iss-163]
resolution: "The install returns a plain-language headline and summary from core (what/why/action per write kind, declined category, outstanding work and optional step), each write carrying its kind; the CLI leads with it and the plugin page relays it."
impact: additive
resolved_by:
  commit: "f0e7c837"
---

The ahoy install completion summary is written for abcd's implementers, not its personas: it leads with insider vocabulary ('managed repo', 'canonical marker blocks', 'the identity gate', 'plugin root not resolvable (machine-scope)') and raw internals (ABCD_PLUGIN_ROOT/CLAUDE_PLUGIN_ROOT, the history-index path) without saying what any of it means for the user's work. An Iris (product thinker) or Nia (facilitator) cannot tell what just changed, which items need action, or why they'd care. Like iss-163, core emits result data with no user-facing explanation layer; each reported item needs a plain-language 'what this is / why it matters / what, if anything, you should do' framing at every front door.

## Grounds

- pursued: every item an install reports reaches the person with core's what/why/action framing and no raw environment names; shown wrong if a reported write, declined category, outstanding step or optional step has no summary item, which the install summary tests catch
