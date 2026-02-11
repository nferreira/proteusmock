package services

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/PaesslerAG/jsonpath"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

// Paginate applies pagination to rendered body bytes.
// It extracts the array at the configured data path, slices it according to
// query parameters, and wraps the result in a pagination envelope.
func Paginate(body []byte, cfg *match.CompiledPagination, queryParams map[string]string) ([]byte, error) {
	var fullData any
	if err := json.Unmarshal(body, &fullData); err != nil {
		return nil, fmt.Errorf("failed to parse response body as JSON: %w", err)
	}

	items, err := extractArray(fullData, cfg.DataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract array at %q: %w", cfg.DataPath, err)
	}

	totalItems := len(items)
	offset, limit := resolveSliceBounds(cfg, queryParams)

	// Clamp offset and end.
	offset = min(offset, totalItems)
	end := min(offset+limit, totalItems)

	sliced := items[offset:end]

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}
	currentPage := (offset / limit) + 1
	hasNext := end < totalItems
	hasPrevious := offset > 0

	env := cfg.Envelope
	envelope := map[string]any{
		env.DataField:        sliced,
		env.PageField:        currentPage,
		env.SizeField:        limit,
		env.TotalItemsField:  totalItems,
		env.TotalPagesField:  totalPages,
		env.HasNextField:     hasNext,
		env.HasPreviousField: hasPrevious,
	}

	result, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pagination envelope: %w", err)
	}

	return result, nil
}

// resolveSliceBounds extracts offset and limit from query parameters
// according to the configured pagination style.
func resolveSliceBounds(cfg *match.CompiledPagination, qp map[string]string) (offset, limit int) {
	limit = cfg.DefaultSize

	switch cfg.Style {
	case "offset_limit":
		if v, ok := qp[cfg.OffsetParam]; ok {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}
		if v, ok := qp[cfg.LimitParam]; ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
	default: // "page_size"
		page := 1
		if v, ok := qp[cfg.PageParam]; ok {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 {
				page = n
			}
		}
		if v, ok := qp[cfg.SizeParam]; ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		offset = (page - 1) * limit
	}

	if limit > cfg.MaxSize {
		limit = cfg.MaxSize
	}
	if limit <= 0 {
		limit = 10
	}

	return offset, limit
}

func extractArray(data any, dataPath string) ([]any, error) {
	if dataPath == "$" {
		arr, ok := data.([]any)
		if !ok {
			return nil, fmt.Errorf("expected root to be an array")
		}
		return arr, nil
	}

	result, err := jsonpath.Get(dataPath, data)
	if err != nil {
		return nil, fmt.Errorf("jsonpath extraction failed: %w", err)
	}

	arr, ok := result.([]any)
	if !ok {
		return nil, fmt.Errorf("value at %q is not an array", dataPath)
	}
	return arr, nil
}
