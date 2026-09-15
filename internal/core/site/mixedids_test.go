package site

import (
	"reflect"
	"testing"
)

// TestDevelopmentDeckOrdersMixedIdVintagesByRecency proves the development
// deck's newest-first order survives the ledger's two id vintages (adr-45): on
// one date a timestamp-numeric id is the most recent mint and leads, the
// ordinals follow in descending numeric order, and the date still dominates the
// id. Nothing here parses the id as anything but a number, so a 16-digit id
// and a 1-digit one are judged on the same axis.
func TestDevelopmentDeckOrdersMixedIdVintagesByRecency(t *testing.T) {
	nodes := []ExportNode{
		{ID: "itd-3", Date: "2026-08-22"},
		{ID: "itd-2608221126066632", Date: "2026-08-22"},
		{ID: "itd-200", Date: "2026-08-22"},
		{ID: "itd-10", Date: "2026-08-22"},
		{ID: "itd-1", Date: "2026-09-01"},
	}
	sortDevelopmentNodes(nodes)
	var got []string
	for _, n := range nodes {
		got = append(got, n.ID)
	}
	want := []string{"itd-1", "itd-2608221126066632", "itd-200", "itd-10", "itd-3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("development deck order:\n got %v\nwant %v", got, want)
	}
}

// TestDecisionIndexOrdersOrdinalsBeforeStamps is the ADR store's own version of
// the case above (the 2026-09-01 ruling): the decisions index carries the
// hand-numbered ordinals 0001–0058 and the minted timestamp ids together, and
// one sort orders them — ordinals first ascending, stamps after, with the record's
// own date still dominating. handleNum reads a 16-digit id and a 2-digit one on
// the same axis, so the deck needs no second comparator for the new vintage.
func TestDecisionIndexOrdersOrdinalsBeforeStamps(t *testing.T) {
	nodes := []ExportNode{
		{ID: "adr-58", Date: "2026-08-31"},
		{ID: "adr-2609012206053814", Date: "2026-08-31"},
		{ID: "adr-1", Date: "2026-08-31"},
		{ID: "adr-12", Date: "2026-09-02"},
	}
	sortDevelopmentNodes(nodes)
	var got []string
	for _, n := range nodes {
		got = append(got, n.ID)
	}
	// Newest date first; within the date the largest number leads, which puts the
	// stamps ahead of the ordinals and the ordinals in descending order — the
	// exact inverse of the ascending index order.
	want := []string{"adr-12", "adr-2609012206053814", "adr-58", "adr-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decision deck order:\n got %v\nwant %v", got, want)
	}
	if handleNum("adr-2609012206053814") <= handleNum("adr-58") {
		t.Fatal("handleNum reads a minted ADR id as no larger than an ordinal")
	}
}
