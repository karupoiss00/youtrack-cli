package app

import (
	"context"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func (s *Service) CreateIssue(ctx context.Context, req CreateIssueRequest) (domain.Issue, error) {
	if strings.TrimSpace(string(req.Project)) == "" {
		return domain.Issue{}, Validationf("project is required")
	}
	if strings.TrimSpace(req.Summary) == "" {
		return domain.Issue{}, Validationf("summary is required")
	}

	project, err := s.resolveProject(ctx, req.Project)
	if err != nil {
		return domain.Issue{}, err
	}
	if err := s.access.Check(project, nil); err != nil {
		return domain.Issue{}, err
	}

	type fieldGroup struct {
		name   string
		values []string
	}
	groups := make([]fieldGroup, 0, len(req.Fields))
	groupIndex := make(map[string]int, len(req.Fields))
	for _, input := range req.Fields {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return domain.Issue{}, Validationf("field name is required")
		}
		key := strings.ToLower(name)
		if index, ok := groupIndex[key]; ok {
			groups[index].values = append(groups[index].values, input.Value)
			continue
		}
		groupIndex[key] = len(groups)
		groups = append(groups, fieldGroup{name: name, values: []string{input.Value}})
	}

	assignments := make([]FieldAssignment, 0, len(groups))
	for _, group := range groups {
		if isBoardRef(group.name) {
			return domain.Issue{}, Validationf("Board cannot be set when creating an issue")
		}
		assignment, err := s.resolver.ResolveForProject(ctx, project, domain.FieldRef(group.name), group.values)
		if err != nil {
			return domain.Issue{}, err
		}
		assignments = append(assignments, assignment)
	}

	tags := make([]domain.Tag, 0, len(req.Tags))
	seenTags := make(map[string]struct{}, len(req.Tags))
	for _, ref := range req.Tags {
		tag, err := s.resolveTag(ctx, ref)
		if err != nil {
			return domain.Issue{}, err
		}
		if _, seen := seenTags[tag.ID]; seen {
			continue
		}
		seenTags[tag.ID] = struct{}{}
		tags = append(tags, tag)
	}

	if err := s.access.Check(project, tags); err != nil {
		return domain.Issue{}, err
	}
	created, err := s.creator.CreateIssue(ctx, IssueCreate{
		Project: project, Summary: req.Summary, Description: req.Description, Fields: assignments, Tags: tags,
	})
	if err != nil {
		return domain.Issue{}, err
	}
	if err := s.access.Check(created.Project, created.Tags); err != nil {
		return domain.Issue{}, err
	}
	if err := s.checkIssueAccess(ctx, domain.IssueRef(created.ID)); err != nil {
		return domain.Issue{}, err
	}
	return created, nil
}
