package paginationutil

// PaginationResult holds pagination metadata.
type PaginationResult struct {
	TotalCount int
	Truncated  bool
}

// ApplyPagination returns the paginated slice and metadata.
// It handles bounds checking to prevent panics and clamps offset/limit.
func ApplyPagination[T any](items []T, offset, limit int) ([]T, PaginationResult) {
	totalCount := len(items)
	start := offset
	end := offset + limit
	truncated := end < totalCount

	if start > totalCount {
		start = totalCount
	}
	if end > totalCount {
		end = totalCount
	}

	return items[start:end], PaginationResult{
		TotalCount: totalCount,
		Truncated:  truncated,
	}
}
