---
schema_version: 1
id: "iss-2610121210280182"
slug: "abcd-build-next-max-0-is-read-as-no"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "ruthless review of fix/build-next-followups"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/next.go"
remedy: "Tell an explicit --max 0 from an absent flag at the CLI (cmd.Flags().Changed) and refuse it with the negative-max refusal's words (a run count is 1 or more), exit 2; a test runs --max 0 with and without a live series and expects the refusal."
resolution: "build next refuses an explicit --max 0 with the negative count's words, exit 2"
impact: fix
resolved_by:
  commit: "cecf8a1f1b1bf5c06e019b16b08e7264f5f92944"
---

abcd build next --max 0 is read as no count at all: NextOptions.Max is 0 when the flag is not given, so an explicit --max 0 continues a live series (exit 0, a series move) and gives a single pick otherwise, while the refusal of a negative --max says a run count is '1 or more'. A person who types --max 0 is neither refused nor told it meant nothing. Found by the ruthless reviewer of fix/build-next-followups.
