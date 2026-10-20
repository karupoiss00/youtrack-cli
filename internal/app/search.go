package app

import (
	"context"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (s *Service) SearchIssues(ctx context.Context, query string, request PageRequest) ([]domain.IssueSummary, error) {
	if strings.TrimSpace(query) == "" {
		return nil, Validationf("search query must not be blank")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if s.access.Restricted() {
		return s.restrictedIssueSummaries(ctx, request, func(ctx context.Context, page Page) ([]domain.IssueSummary, error) {
			return s.issueSearch.SearchIssues(ctx, query, page)
		})
	}

	items := make([]domain.IssueSummary, 0)
	offset := request.Offset
	remaining := request.pageLimit()
	for request.All || remaining > 0 {
		limit := DefaultPageLimit
		if !request.All && remaining < limit {
			limit = remaining
		}
		page, err := s.issueSearch.SearchIssues(ctx, query, Page{Offset: offset, Limit: limit})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		if !request.All && len(page) > remaining {
			page = page[:remaining]
		}
		items = append(items, page...)
		offset += len(page)
		if !request.All {
			remaining -= len(page)
		}
	}
	return items, nil
}
