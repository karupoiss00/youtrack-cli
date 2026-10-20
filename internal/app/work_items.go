package app

import (
	"context"
	"strings"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type AddWorkItemRequest struct {
	Duration string
	Date     string
	Text     *string
	Type     *string
	Author   *string
}

type EditWorkItemRequest struct {
	Duration  *string
	Date      *string
	Text      *string
	Type      *string
	ClearType bool
	Author    *string
}

func (s *Service) ListWorkItemTypes(ctx context.Context, ref domain.IssueRef, request PageRequest) ([]domain.WorkItemType, error) {
	if strings.TrimSpace(string(ref)) == "" {
		return nil, Validationf("issue reference must not be blank")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	issue, err := s.accessibleIssue(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.listWorkItemTypes(ctx, issue.Project.ID, request)
}

func (s *Service) ListWorkItems(ctx context.Context, ref domain.IssueRef, request PageRequest) ([]domain.WorkItem, error) {
	if strings.TrimSpace(string(ref)) == "" {
		return nil, Validationf("issue reference must not be blank")
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	issue, err := s.accessibleIssue(ctx, ref)
	if err != nil {
		return nil, err
	}
	return s.listWorkItems(ctx, domain.IssueRef(issue.ID), request)
}

func (s *Service) GetWorkItem(ctx context.Context, ref domain.IssueRef, itemID string) (domain.WorkItem, error) {
	if strings.TrimSpace(string(ref)) == "" {
		return domain.WorkItem{}, Validationf("issue reference must not be blank")
	}
	if strings.TrimSpace(itemID) == "" {
		return domain.WorkItem{}, Validationf("work item ID must not be blank")
	}
	issue, err := s.accessibleIssue(ctx, ref)
	if err != nil {
		return domain.WorkItem{}, err
	}
	return s.workItems.GetWorkItem(ctx, domain.IssueRef(issue.ID), itemID)
}

func (s *Service) AddWorkItem(ctx context.Context, ref domain.IssueRef, request AddWorkItemRequest) (domain.WorkItem, error) {
	if strings.TrimSpace(string(ref)) == "" {
		return domain.WorkItem{}, Validationf("issue reference must not be blank")
	}
	date, err := parseWorkItemDate(request.Date)
	if err != nil {
		return domain.WorkItem{}, err
	}
	duration, err := parseWorkItemDuration(request.Duration)
	if err != nil {
		return domain.WorkItem{}, err
	}
	if err := validateWorkItemReference("type", request.Type); err != nil {
		return domain.WorkItem{}, err
	}
	if err := validateWorkItemReference("author", request.Author); err != nil {
		return domain.WorkItem{}, err
	}

	issue, err := s.accessibleIssue(ctx, ref)
	if err != nil {
		return domain.WorkItem{}, err
	}
	if err := s.requireTimeTracking(ctx, issue.Project.ID); err != nil {
		return domain.WorkItem{}, err
	}
	var workType *domain.WorkItemType
	if request.Type != nil {
		resolved, err := s.resolveWorkItemType(ctx, issue.Project.ID, *request.Type)
		if err != nil {
			return domain.WorkItem{}, err
		}
		workType = &resolved
	}
	var author *domain.User
	if request.Author != nil {
		resolved, err := s.resolveWorkItemAuthor(ctx, *request.Author)
		if err != nil {
			return domain.WorkItem{}, err
		}
		author = &resolved
	}
	return s.workItems.CreateWorkItem(ctx, domain.IssueRef(issue.ID), WorkItemCreate{
		Date: date, DurationMinutes: duration, Text: request.Text, Type: workType, Author: author,
	})
}

func (s *Service) EditWorkItem(ctx context.Context, ref domain.IssueRef, itemID string, request EditWorkItemRequest) (domain.WorkItem, error) {
	if strings.TrimSpace(string(ref)) == "" {
		return domain.WorkItem{}, Validationf("issue reference must not be blank")
	}
	if strings.TrimSpace(itemID) == "" {
		return domain.WorkItem{}, Validationf("work item ID must not be blank")
	}
	if request.Type != nil && request.ClearType {
		return domain.WorkItem{}, Validationf("--type and --clear-type are mutually exclusive")
	}
	if request.Duration == nil && request.Date == nil && request.Text == nil && request.Type == nil && !request.ClearType && request.Author == nil {
		return domain.WorkItem{}, Validationf("at least one work item change is required")
	}
	if err := validateWorkItemReference("type", request.Type); err != nil {
		return domain.WorkItem{}, err
	}
	if err := validateWorkItemReference("author", request.Author); err != nil {
		return domain.WorkItem{}, err
	}

	issue, err := s.accessibleIssue(ctx, ref)
	if err != nil {
		return domain.WorkItem{}, err
	}
	issueRef := domain.IssueRef(issue.ID)
	if _, err := s.workItems.GetWorkItem(ctx, issueRef, itemID); err != nil {
		return domain.WorkItem{}, err
	}
	if err := s.requireTimeTracking(ctx, issue.Project.ID); err != nil {
		return domain.WorkItem{}, err
	}

	patch := WorkItemPatch{Text: request.Text, Author: nil}
	if request.Duration != nil {
		duration, err := parseWorkItemDuration(*request.Duration)
		if err != nil {
			return domain.WorkItem{}, err
		}
		patch.DurationMinutes = &duration
	}
	if request.Date != nil {
		date, err := parseWorkItemDate(*request.Date)
		if err != nil {
			return domain.WorkItem{}, err
		}
		patch.Date = &date
	}
	if request.Type != nil {
		resolved, err := s.resolveWorkItemType(ctx, issue.Project.ID, *request.Type)
		if err != nil {
			return domain.WorkItem{}, err
		}
		patch.Type = WorkItemTypePatch{Set: true, Value: &resolved}
	} else if request.ClearType {
		patch.Type = WorkItemTypePatch{Set: true}
	}
	if request.Author != nil {
		resolved, err := s.resolveWorkItemAuthor(ctx, *request.Author)
		if err != nil {
			return domain.WorkItem{}, err
		}
		patch.Author = &resolved
	}
	return s.workItems.UpdateWorkItem(ctx, issueRef, itemID, patch)
}

func (s *Service) RemoveWorkItem(ctx context.Context, ref domain.IssueRef, itemID string) (string, error) {
	if strings.TrimSpace(string(ref)) == "" {
		return "", Validationf("issue reference must not be blank")
	}
	if strings.TrimSpace(itemID) == "" {
		return "", Validationf("work item ID must not be blank")
	}
	issue, err := s.accessibleIssue(ctx, ref)
	if err != nil {
		return "", err
	}
	item, err := s.workItems.GetWorkItem(ctx, domain.IssueRef(issue.ID), itemID)
	if err != nil {
		return "", err
	}
	if err := s.workItems.DeleteWorkItem(ctx, domain.IssueRef(issue.ID), itemID); err != nil {
		return "", err
	}
	return item.ID, nil
}

func (s *Service) listWorkItemTypes(ctx context.Context, projectID string, request PageRequest) ([]domain.WorkItemType, error) {
	items := make([]domain.WorkItemType, 0)
	offset := request.Offset
	remaining := request.pageLimit()
	for request.All || remaining > 0 {
		limit := DefaultPageLimit
		if !request.All && remaining < limit {
			limit = remaining
		}
		page, err := s.timeTracking.ListWorkItemTypes(ctx, projectID, Page{Offset: offset, Limit: limit})
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

func (s *Service) listWorkItems(ctx context.Context, issue domain.IssueRef, request PageRequest) ([]domain.WorkItem, error) {
	items := make([]domain.WorkItem, 0)
	offset := request.Offset
	remaining := request.pageLimit()
	for request.All || remaining > 0 {
		limit := DefaultPageLimit
		if !request.All && remaining < limit {
			limit = remaining
		}
		page, err := s.workItems.ListWorkItems(ctx, issue, Page{Offset: offset, Limit: limit})
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

func (s *Service) requireTimeTracking(ctx context.Context, projectID string) error {
	enabled, err := s.timeTracking.TimeTrackingEnabled(ctx, projectID)
	if err != nil {
		return err
	}
	if !enabled {
		return Validationf("time tracking is disabled for project %q", projectID)
	}
	return nil
}

func (s *Service) resolveWorkItemType(ctx context.Context, projectID, reference string) (domain.WorkItemType, error) {
	items, err := s.listWorkItemTypes(ctx, projectID, PageRequest{All: true})
	if err != nil {
		return domain.WorkItemType{}, err
	}
	for _, item := range items {
		if item.ID == reference {
			return item, nil
		}
	}
	if match, ok := uniqueWorkItemTypeMatch(items, reference, false); ok {
		return match, nil
	} else if !ok && workItemTypeMatchCount(items, reference, false) > 1 {
		return domain.WorkItemType{}, Ambiguousf("work item type %q is ambiguous", reference)
	}
	if match, ok := uniqueWorkItemTypeMatch(items, reference, true); ok {
		return match, nil
	} else if !ok && workItemTypeMatchCount(items, reference, true) > 1 {
		return domain.WorkItemType{}, Ambiguousf("work item type %q is ambiguous", reference)
	}
	return domain.WorkItemType{}, NotFoundf("work item type %q was not found", reference)
}

func (s *Service) resolveWorkItemAuthor(ctx context.Context, reference string) (domain.User, error) {
	if reference == "@me" {
		return s.users.Me(ctx)
	}
	items, err := s.listUsers(ctx)
	if err != nil {
		return domain.User{}, err
	}
	for _, item := range items {
		if item.ID == reference {
			return item, nil
		}
	}
	if match, ok := uniqueWorkItemUserMatch(items, reference, false); ok {
		return match, nil
	} else if !ok && workItemUserMatchCount(items, reference, false) > 1 {
		return domain.User{}, Ambiguousf("user %q is ambiguous", reference)
	}
	if match, ok := uniqueWorkItemUserMatch(items, reference, true); ok {
		return match, nil
	} else if !ok && workItemUserMatchCount(items, reference, true) > 1 {
		return domain.User{}, Ambiguousf("user %q is ambiguous", reference)
	}
	return domain.User{}, NotFoundf("user %q was not found", reference)
}

func (s *Service) listUsers(ctx context.Context) ([]domain.User, error) {
	items := make([]domain.User, 0)
	for offset := 0; ; {
		page, err := s.userDirectory.ListUsers(ctx, Page{Offset: offset, Limit: DefaultPageLimit})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return items, nil
		}
		items = append(items, page...)
		offset += len(page)
	}
}

func parseWorkItemDate(input string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", input)
	if err != nil {
		return time.Time{}, Validationf("invalid work item date %q: expected YYYY-MM-DD", input)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC), nil
}

func parseWorkItemDuration(input string) (int64, error) {
	minutes, err := parsePeriod(input)
	if err != nil || minutes <= 0 {
		return 0, Validationf("work item duration must be a positive h/m period")
	}
	return minutes, nil
}

func validateWorkItemReference(kind string, reference *string) error {
	if reference != nil && strings.TrimSpace(*reference) == "" {
		return Validationf("work item %s reference must not be blank", kind)
	}
	return nil
}

func uniqueWorkItemTypeMatch(items []domain.WorkItemType, reference string, folded bool) (domain.WorkItemType, bool) {
	var match domain.WorkItemType
	count := 0
	for _, item := range items {
		matches := item.Name == reference
		if folded {
			matches = strings.EqualFold(item.Name, reference)
		}
		if matches {
			match = item
			count++
		}
	}
	return match, count == 1
}

func workItemTypeMatchCount(items []domain.WorkItemType, reference string, folded bool) int {
	count := 0
	for _, item := range items {
		matches := item.Name == reference
		if folded {
			matches = strings.EqualFold(item.Name, reference)
		}
		if matches {
			count++
		}
	}
	return count
}

func uniqueWorkItemUserMatch(items []domain.User, reference string, folded bool) (domain.User, bool) {
	var match domain.User
	count := 0
	for _, item := range items {
		matches := item.Login == reference
		if folded {
			matches = strings.EqualFold(item.Login, reference)
		}
		if matches {
			match = item
			count++
		}
	}
	return match, count == 1
}

func workItemUserMatchCount(items []domain.User, reference string, folded bool) int {
	count := 0
	for _, item := range items {
		matches := item.Login == reference
		if folded {
			matches = strings.EqualFold(item.Login, reference)
		}
		if matches {
			count++
		}
	}
	return count
}
