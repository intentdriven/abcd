---
id: dcn-2609230000000013
slug: a-load-experiment-is-trusted-only-under
date: 2026-09-23
---
2026-09-23 — A load experiment is trusted only under four conditions (product thinker, ruling on iss-2609210828122412, after orphaned load processes from another repository starved the machine into a crash): background load processes start as one owned process group and are killed together; "clean" is proven by checking what is actually running, never by reading a list of what was started; an experiment on a live development machine needs explicit consent and a cap below the core count; and abcd's own test lanes check machine load before they start and refuse when it is too high, naming what is running. The first three hold as rules now; the last is an intent to plan.
