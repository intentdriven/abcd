package lint

import (
	"reflect"
	"sort"
	"testing"
)

// TestHandleLessOrdersOrdinalsBeforeTimestampIds proves the one canonical
// record-list order holds across the ledger's two id vintages (adr-45): the
// ordinals sort numerically among themselves, every timestamp-numeric id sorts
// after every ordinal because it is numerically larger, and two native ids sort
// by mint instant. Families stay grouped by prefix. A plain string sort would
// interleave the vintages (itd-2608… between itd-2 and itd-3), which is exactly
// the inversion a reader of a rendered list notices first.
func TestHandleLessOrdersOrdinalsBeforeTimestampIds(t *testing.T) {
	ids := []string{
		"spc-2609011200000001", "itd-2608221126066632", "itd-3", "spc-69",
		"itd-200", "itd-2608210737260468", "itd-10", "spc-7", "itd-1",
	}
	sort.SliceStable(ids, func(i, j int) bool { return HandleLess(ids[i], ids[j]) })
	want := []string{
		"itd-1", "itd-3", "itd-10", "itd-200",
		"itd-2608210737260468", "itd-2608221126066632",
		"spc-7", "spc-69", "spc-2609011200000001",
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("HandleLess order:\n got %v\nwant %v", ids, want)
	}
	// Stable under a second pass: the order is a total one, not a shuffle.
	again := append([]string{}, want...)
	sort.SliceStable(again, func(i, j int) bool { return HandleLess(again[i], again[j]) })
	if !reflect.DeepEqual(again, want) {
		t.Fatalf("HandleLess is not stable over its own output: %v", again)
	}
}
