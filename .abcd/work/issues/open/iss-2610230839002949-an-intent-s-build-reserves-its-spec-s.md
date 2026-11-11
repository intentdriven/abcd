---
schema_version: 1
id: "iss-2610230839002949"
slug: "an-intent-s-build-reserves-its-spec-s"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Make the reservation rule a per-repository setting with two values, as dcn-2611012049050703 rules: as each step lands, narrow the reservation to the files the steps not yet landed still name, the default, so a later step's files are never unreserved mid-run; or reserve only the steps in flight, widening with a plain reserve of the union (keeping the tip's term) as each further lane opens and narrowing as each step lands."
---

An intent's build reserves its spec's whole Footprint packages line once, for every lane of the run, and never reads a step's own packages line, so the reservation cannot start at the first step's files and widen step by step. Building that needs a reservation per open lane (the union of the steps in flight, since steps marked needs: none run beside each other), a widening act as each lane opens (a reserve with the union and the tip's term, as widenReservation already does, since claim.Request.Narrow refuses to widen), and a narrowing as each step lands. That is more than the directory-warning fix for iss-2610230016544142 could carry, so it is filed on its own.

2026-11-09 and 2026-11-10: seen again in the autonomous run of 7 Nov. On 9 Nov at 08:27 a build of itd-2610240922461281 was refused at its reservation because the spec's footprint names the whole internal/core/implement/loop package and another holder had reserved one test file in it; by 12:27 that one-file hold, renewed twice at the same stage, had blocked the whole-package footprint for four hours, and the step's own files were reserved by hand instead. Later on 9 Nov two lanes of one run blocked each other because the checklist spec names the whole internal/surface/cli package, and one lane was narrowed to its step under a hand reservation. On 10 Nov at 12:23 a one-package step collided with lanes in other packages for the same reason.
