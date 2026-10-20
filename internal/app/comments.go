package app

import (
	"context"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (s *Service) ListComments(ctx context.Context, issue domain.IssueRef, request PageRequest) ([]domain.Comment, error) {
	if strings.TrimSpace(string(issue)) == "" {
		return nil, Validationf("issue reference must not be blank")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if err := s.checkIssueAccess(ctx, issue); err != nil {
		return nil, err
	}

	items := make([]domain.Comment, 0)
	offset := request.Offset
	remaining := request.pageLimit()
	for request.All || remaining > 0 {
		limit := DefaultPageLimit
		if !request.All && remaining < limit {
			limit = remaining
		}
		page, err := s.comments.ListComments(ctx, issue, Page{Offset: offset, Limit: limit})
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

func (s *Service) AddComment(ctx context.Context, issue domain.IssueRef, text string) (domain.Comment, error) {
	if strings.TrimSpace(string(issue)) == "" {
		return domain.Comment{}, Validationf("issue reference must not be blank")
	}
	if strings.TrimSpace(text) == "" {
		return domain.Comment{}, Validationf("comment text must not be blank")
	}
	if err := s.checkIssueAccess(ctx, issue); err != nil {
		return domain.Comment{}, err
	}
	return s.comments.CreateComment(ctx, issue, text)
}

func (s *Service) EditComment(ctx context.Context, issue domain.IssueRef, commentID, text string) (domain.Comment, error) {
	if strings.TrimSpace(string(issue)) == "" {
		return domain.Comment{}, Validationf("issue reference must not be blank")
	}
	if strings.TrimSpace(commentID) == "" {
		return domain.Comment{}, Validationf("comment ID must not be blank")
	}
	if strings.TrimSpace(text) == "" {
		return domain.Comment{}, Validationf("comment text must not be blank")
	}
	if err := s.checkIssueAccess(ctx, issue); err != nil {
		return domain.Comment{}, err
	}
	return s.comments.EditComment(ctx, issue, commentID, text)
}

func (s *Service) RemoveComment(ctx context.Context, issue domain.IssueRef, commentID string) error {
	if strings.TrimSpace(string(issue)) == "" {
		return Validationf("issue reference must not be blank")
	}
	if strings.TrimSpace(commentID) == "" {
		return Validationf("comment ID must not be blank")
	}
	if err := s.checkIssueAccess(ctx, issue); err != nil {
		return err
	}
	return s.comments.SoftRemoveComment(ctx, issue, commentID)
}
