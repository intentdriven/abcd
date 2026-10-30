---
id: dcn-2610292236161673
slug: decided-by-the-technical-facilitator-on
date: 2026-10-29
---
2026-10-29 — Decided by the technical facilitator on delegation (coordinator rulings on the review of the merge that landed step 2 of spc-2610240951109420, 2026-10-29), two rules for a model asked per request, for step 3 to build. (a) When a request asks the claude runner for a model the machine pins, the dispatch is refused before anything launches and nothing falls back, to the host or to any other runner, with the words the refusal of a --route to a pinned model on claude already uses: claude can hold no pin, and a run on another model is a run nobody asked for. As merged, the request is refused at launch and the role is handed back to the host; step 3 makes that refusal final. (b) When a request names a model and a --route names a model too, and the two differ, the dispatch is refused before anything launches, naming both; when they are the same model it runs, and the up-front pin check vets that one model.
