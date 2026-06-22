package repository

import "testing"

func TestNewPageNormalizesBounds(t *testing.T) {
	page := NewPage(-5, 1000)

	if page.Limit != MaxLimit {
		t.Fatalf("limit = %d, want %d", page.Limit, MaxLimit)
	}
	if page.Offset != 0 {
		t.Fatalf("offset = %d, want 0", page.Offset)
	}
}
