---
schema_version: 1
id: "iss-2611050035192370"
slug: "nothing-reads-a-failed-scheduled-ci-run"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "building the release path's macOS gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Have abcd status name a failed latest scheduled ci.yml run on the default branch with its URL, in the verb and in the prompt's STATUS block: a step outside the prompt reads it through gh into the local tier, the prompt reads only that cache so its deadline is not spent on the network, an unknown result says nothing, and a repository names its own scheduled workflow."
resolution: "A repository names its workflow as status.scheduled_workflow on its default branch, and abcd status, and the prompt hook in the background once the answer is an hour old, asks the forge for its latest scheduled runs and keeps what they say in the machine's store, shared by every checkout. The verb and the prompt's STATUS block read only that, and name a red verdict with its URL and age to each new session until a green replaces it, reading back past cancelled and skipped runs; a workflow with no verdict for a few of its cycles, and an answer that is missing, stale or unreadable, each say so rather than read like green, as does a refresh that failed once one was due."
impact: additive
---

Nothing reads a failed scheduled run of ci.yml: the nightly macOS suite and the nightly Linux checks can stay red on the default branch with no session told, and since the release path now runs its own macOS suite the first anyone hears of a macOS regression can be a cut that stops. The alert is not small enough to add beside the gate: the prompt's STATUS block is read from disk alone and promises never to touch the network, so the cache needs a refresh outside the prompt, or the alert stays silent in any session where nobody ran the verb; the workflow's name is abcd's own, while abcd status serves every managed repository; and the verb's JSON, its command page, its brief chapter and the generated reference all change with it.
