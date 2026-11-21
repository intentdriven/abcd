---
id: dcn-2611201515310051
slug: the-drain-s-declared-order-in
date: 2026-11-20
---
2026-11-20 — The drain's declared order in adr-2609291342092738 (decision 4) is amended, not kept, for itd-2611201450052326: from step 4 of spc-2611201455051897 the order chooses each lane's lead, and the lead's members are drawn after it in drain order from its category and area at either fixable severity, so a member may be taken ahead of an older or a nitpick issue elsewhere. Rejected: letting members join only at the lead's severity, which would not keep the order either, since a member would still pass older issues of its own tier in other areas, and which would split the person's one kind in one area (dcn-2611201429107457) by a field the person did not name. Grounds: every member is eligible, judged and reserved on its own, so no member is taken that the drain would not take alone, and a bundle opens no extra lane under the drain's lane cap, so the order still decides which lanes open first. The ADR carries a dated amendment saying so, as its consequences require of a change to the order. Decided by the technical facilitator on delegation, 20 Nov 2026.
