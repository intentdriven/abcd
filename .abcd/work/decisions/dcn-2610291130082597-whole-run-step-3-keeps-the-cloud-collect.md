---
id: dcn-2610291130082597
slug: whole-run-step-3-keeps-the-cloud-collect
date: 2026-10-29
---
2026-10-29 — Whole-run step 3 keeps the cloud collect script's own waiter around make preflight. The laptop's step-3 list dropped it as no longer needed, but a plain make -C <worktree> preflight lets the collected tree's own load-gate code decide when its preflight starts on this machine; the outer waiter runs from the person's checkout, and the Makefile already holds that wrapping is harmless (the inner waiter opens at once inside the outer turn). Raised by the Opus reviewer of fbee3d246.
