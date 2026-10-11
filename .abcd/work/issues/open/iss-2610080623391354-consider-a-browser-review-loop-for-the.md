---
schema_version: 1
id: "iss-2610080623391354"
slug: "consider-a-browser-review-loop-for-the"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "the guard-refusal session of 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "the product thinker's dashboard and abcd's interviews (commands/dashboard.md)"
remedy: "Prototype on the dashboard a review page for one HTML artifact with element-anchored annotations, a feedback queue the agent collects by long-polling, and revision markers; trial it on one real interview or criteria walk against the terminal questions, measuring the person's effort and the rounds needed, before deciding whether interviews hand long material to it."
---

Consider a browser review loop for the product thinker, seen in an outside open-source agent tool: the agent writes an HTML artifact (a mockup, a report, a criteria walk, an intent's press release); the person opens it in the browser, annotates exact elements or text, attaches images, and queues feedback; the agent collects the whole batch with one long-polling call and writes the next revision, with the changed parts marked as revisions. The same tool audits each render for layout faults (clipped text, unreachable controls) and lists them as issues the person can send back. Today abcd's only channel from the person back to the agent is the terminal question tool, held to twenty-four rows at eighty columns, so long material is split across many questions or pushed into an HTML report read separately, with the answers still typed in the terminal. abcd's dashboard already runs a server reachable only from the person's own devices over Tailscale, but serves nothing of the project yet, so it is the natural host for such a loop. Feedback arriving from the browser is a new path into the agent's context and must be treated as data, never instruction.
