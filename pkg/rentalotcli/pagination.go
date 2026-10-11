package rentalotcli

import (
	"fmt"
	"strconv"
)

// CursorParams selects a cursor page. Cursor is opaque and is URL-encoded by Get.
// A zero Limit uses the server's default; an empty Cursor selects the first page.
type CursorParams struct {
	Limit  int
	Cursor string
}

// QueryParams returns a new map that callers can extend with endpoint filters.
func (p CursorParams) QueryParams() (QueryParams, error) {
	if p.Limit < 0 {
		return nil, fmt.Errorf("cursor pagination: limit must be non-negative")
	}
	params := QueryParams{}
	if p.Limit > 0 {
		params["limit"] = strconv.Itoa(p.Limit)
	}
	if p.Cursor != "" {
		params["cursor"] = p.Cursor
	}
	return params, nil
}

// CursorPagination describes metadata in a cursor page's pagination object.
// A missing/null/empty nextCursor ends pagination. hasMore is optional; an
// explicit false also ends pagination, and true requires a non-empty nextCursor.
type CursorPagination struct {
	NextCursor *string `json:"nextCursor"`
	HasMore    *bool   `json:"hasMore"`
}

// CursorPage is a typed data array with cursor pagination metadata.
type CursorPage[T any] struct {
	Data       []T              `json:"data"`
	Pagination CursorPagination `json:"pagination"`
}

// Next returns parameters for the next request, preserving the requested limit.
// The bool is false on the last page. An unchanged cursor is rejected to detect
// stalled pagination. Callers control requests and cancellation.
func (p CursorPage[T]) Next(current CursorParams) (CursorParams, bool, error) {
	if _, err := current.QueryParams(); err != nil {
		return CursorParams{}, false, err
	}
	meta := p.Pagination
	if meta.HasMore != nil && !*meta.HasMore {
		return CursorParams{}, false, nil
	}
	if meta.NextCursor == nil || *meta.NextCursor == "" {
		if meta.HasMore != nil && *meta.HasMore {
			return CursorParams{}, false, fmt.Errorf("cursor pagination: hasMore requires nextCursor")
		}
		return CursorParams{}, false, nil
	}
	if *meta.NextCursor == current.Cursor {
		return CursorParams{}, false, fmt.Errorf("cursor pagination: nextCursor did not advance")
	}
	return CursorParams{Limit: current.Limit, Cursor: *meta.NextCursor}, true, nil
}

// OffsetParams selects an offset page. A zero Limit uses the server's default.
type OffsetParams struct {
	Limit  int
	Offset int
}

// QueryParams returns a new map that callers can extend with endpoint filters.
// Offset zero is sent explicitly to select the first page.
func (p OffsetParams) QueryParams() (QueryParams, error) {
	if p.Limit < 0 || p.Offset < 0 {
		return nil, fmt.Errorf("offset pagination: limit and offset must be non-negative")
	}
	params := QueryParams{"offset": strconv.Itoa(p.Offset)}
	if p.Limit > 0 {
		params["limit"] = strconv.Itoa(p.Limit)
	}
	return params, nil
}

// OffsetPagination describes metadata in an offset page's pagination object.
// Limit is the effective server page size, Offset is the current offset, and
// Total is optional. Without Total, a short or empty page ends pagination.
type OffsetPagination struct {
	Limit  int  `json:"limit"`
	Offset int  `json:"offset"`
	Total  *int `json:"total"`
}

// OffsetPage is a typed data array with offset pagination metadata.
type OffsetPage[T any] struct {
	Data       []T              `json:"data"`
	Pagination OffsetPagination `json:"pagination"`
}

// Next returns parameters for the next request, using the effective server
// limit and advancing by the number of returned items. The bool is false on
// the last page. An empty page always ends pagination, even if Total is stale.
func (p OffsetPage[T]) Next() (OffsetParams, bool, error) {
	meta := p.Pagination
	count := len(p.Data)
	if meta.Limit < 0 || meta.Offset < 0 || (meta.Total != nil && *meta.Total < 0) {
		return OffsetParams{}, false, fmt.Errorf("offset pagination: metadata must be non-negative")
	}
	if count == 0 {
		return OffsetParams{}, false, nil
	}
	if meta.Limit == 0 || count > meta.Limit {
		return OffsetParams{}, false, fmt.Errorf("offset pagination: limit must cover the returned items")
	}
	maxInt := int(^uint(0) >> 1)
	if meta.Offset > maxInt-count {
		return OffsetParams{}, false, fmt.Errorf("offset pagination: next offset overflows int")
	}
	nextOffset := meta.Offset + count
	if meta.Total != nil {
		if nextOffset >= *meta.Total {
			return OffsetParams{}, false, nil
		}
	} else if count < meta.Limit {
		return OffsetParams{}, false, nil
	}
	return OffsetParams{Limit: meta.Limit, Offset: nextOffset}, true, nil
}
