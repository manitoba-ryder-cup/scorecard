package golf

import (
	"context"
	"fmt"
)

// FormatService exposes the code-defined match formats. They are seeded reference
// data, not user content, so this is read-only.
type FormatService struct {
	FormatDB formatDB
}

func (s *FormatService) ListFormats(ctx context.Context) ([]MatchFormat, error) {
	formats, err := s.FormatDB.ListMatchFormats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list match formats: %w", err)
	}
	return formats, nil
}
