---
id: dcn-2610161907326838
slug: a-forge-record-of-pushes-whose-page-is
date: 2026-10-16
---
2026-10-16 — A forge record of pushes whose page is not full but that names no push to the tip this checkout holds is read as a record not read in full: nothing it leaves open is kept, and the next fetch asks again (spc-2610150728396006 step 3 fix round 4, on review). The forge records a push a little after it lands, so such a record is one that has not caught up yet, and keeping no pusher for those commits would have left them claimed for good.
