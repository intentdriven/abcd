---
id: dcn-2610241450155347
slug: reservations-refuse-while-any-embargo
date: 2026-10-24
---
2026-10-24 — Reservations refuse while any embargo store under the home cannot be read, by design: abcd reserve and the build and drain reservations ask the same outbound embargo check abcd announce does, which reads every embargo store under this account's home, so one store whose state cannot be read refuses every reservation, renewal and take-over on the machine until it is settled. A release is not checked, so a reservation that stands can always be ended. The refusal names the store's path and the reason's own repair, so the fault is found from any checkout. Failing open would read a store that cannot be read as holding nothing, as every other outbound check refuses to; the cost of failing closed is one repair, and it lands on the person who owns every store under that home
