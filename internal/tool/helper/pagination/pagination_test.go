package pagination

import (
	"testing"
)

func TestApplyPagination(t *testing.T) {
	tests := []struct {
		name           string
		items          []int
		offset         int
		limit          int
		wantLen        int
		wantTotalCount int
		wantTruncated  bool
		wantFirstItem  int // -1 if empty
	}{
		{
			name:           "normal pagination first page",
			items:          []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			offset:         0,
			limit:          3,
			wantLen:        3,
			wantTotalCount: 10,
			wantTruncated:  true,
			wantFirstItem:  1,
		},
		{
			name:           "normal pagination middle page",
			items:          []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			offset:         3,
			limit:          3,
			wantLen:        3,
			wantTotalCount: 10,
			wantTruncated:  true,
			wantFirstItem:  4,
		},
		{
			name:           "last page partial",
			items:          []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			offset:         8,
			limit:          5,
			wantLen:        2,
			wantTotalCount: 10,
			wantTruncated:  false,
			wantFirstItem:  9,
		},
		{
			name:           "last page exact",
			items:          []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			offset:         7,
			limit:          3,
			wantLen:        3,
			wantTotalCount: 10,
			wantTruncated:  false,
			wantFirstItem:  8,
		},
		{
			name:           "offset beyond end",
			items:          []int{1, 2, 3, 4, 5},
			offset:         100,
			limit:          10,
			wantLen:        0,
			wantTotalCount: 5,
			wantTruncated:  false,
			wantFirstItem:  -1,
		},
		{
			name:           "empty input",
			items:          []int{},
			offset:         0,
			limit:          10,
			wantLen:        0,
			wantTotalCount: 0,
			wantTruncated:  false,
			wantFirstItem:  -1,
		},
		{
			name:           "offset at exact end",
			items:          []int{1, 2, 3, 4, 5},
			offset:         5,
			limit:          10,
			wantLen:        0,
			wantTotalCount: 5,
			wantTruncated:  false,
			wantFirstItem:  -1,
		},
		{
			name:           "single item",
			items:          []int{42},
			offset:         0,
			limit:          10,
			wantLen:        1,
			wantTotalCount: 1,
			wantTruncated:  false,
			wantFirstItem:  42,
		},
		{
			name:           "limit equals total",
			items:          []int{1, 2, 3, 4, 5},
			offset:         0,
			limit:          5,
			wantLen:        5,
			wantTotalCount: 5,
			wantTruncated:  false,
			wantFirstItem:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, pagination := ApplyPagination(tt.items, tt.offset, tt.limit)

			if len(result) != tt.wantLen {
				t.Errorf("got len %d, want %d", len(result), tt.wantLen)
			}

			if pagination.TotalCount != tt.wantTotalCount {
				t.Errorf("got TotalCount %d, want %d", pagination.TotalCount, tt.wantTotalCount)
			}

			if pagination.Truncated != tt.wantTruncated {
				t.Errorf("got Truncated %v, want %v", pagination.Truncated, tt.wantTruncated)
			}

			if tt.wantFirstItem != -1 && len(result) > 0 && result[0] != tt.wantFirstItem {
				t.Errorf("got first item %d, want %d", result[0], tt.wantFirstItem)
			}
		})
	}
}

func TestApplyPagination_Strings(t *testing.T) {
	// Verify generic works with strings
	items := []string{"a", "b", "c", "d", "e"}
	result, pagination := ApplyPagination(items, 1, 2)

	if len(result) != 2 {
		t.Errorf("got len %d, want 2", len(result))
	}
	if result[0] != "b" || result[1] != "c" {
		t.Errorf("got %v, want [b c]", result)
	}
	if pagination.TotalCount != 5 {
		t.Errorf("got TotalCount %d, want 5", pagination.TotalCount)
	}
	if !pagination.Truncated {
		t.Error("expected Truncated=true")
	}
}
