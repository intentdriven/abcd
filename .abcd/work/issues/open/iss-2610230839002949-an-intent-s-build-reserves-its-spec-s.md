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
remedy: "Make the reservation rule a per-repository setting with two values, as dcn-2611012049050703 rules: release each step's files once that step lands, the default, so a later step's files are never unreserved mid-run; or reserve only the steps in flight, widening with a plain reserve of the union (keeping the tip's term) as each further lane opens and narrowing as each step lands."
---

An intent's build reserves its spec's whole Footprint packages line once, for every lane of the run, and never reads a step's own packages line, so the reservation cannot start at the first step's files and widen step by step. Building that needs a reservation per open lane (the union of the steps in flight, since steps marked needs: none run beside each other), a widening act as each lane opens (a reserve with the union and the tip's term, as widenReservation already does, since claim.Request.Narrow refuses to widen), and a narrowing as each step lands. That is more than the directory-warning fix for iss-2610230016544142 could carry, so it is filed on its own.
