---
schema_version: 1
id: "iss-2611011010338815"
slug: "a-back-off-leaves-no-trace-when-abcd"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "a person asked who an over-wide hold held back and whether they restarted after it was released"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Record each back-off where every copy can read it (who backed off, which record and paths, against which hold, when), and when that hold is released or narrowed, list the copies it held back so abcd peers names them and a backed-off build or drain can be told to start again"
---

A back-off leaves no trace: when abcd reserve or a build backs off another copy's live hold (exit 3), nothing is written that the holder or another copy can read (a build lane's widening back-off is noted only in that lane's own local run record), so neither the holder nor a person can later tell who was held back, from which paths, or whether they tried again once the hold ended. On 2026-10-31 a hand reservation of iss-2610311224444576 held eleven record files it never changed for almost four hours (23:09:38Z to 03:06:16Z); answering who that held back took a hand search of every reservation chain on origin, every local run record and origin's history, which found none, and could not rule out a refused attempt on another machine or account, since a refusal pushes nothing. Two open records touch neighbouring parts of the same remedy: iss-2610241424428420 (nothing tells a backed-off build or drain when the hold is released) and iss-2610311604123786 (a widening's back-off is re-handed with no wait); a fix here should build on them, not repeat them.
