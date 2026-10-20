package app

import (
	"context"
	"slices"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

// AccessPolicy denies a project or any issue carrying a denied tag.
// References match IDs or names, case insensitively, without wildcards.
type AccessPolicy struct {
	DeniedProjects []string
	DeniedTags     []string
}

func (p AccessPolicy) Restricted() bool {
	return len(p.DeniedProjects) > 0 || len(p.DeniedTags) > 0
}

func (p AccessPolicy) Check(project domain.Project, tags []domain.Tag) error {
	for _, ref := range p.DeniedProjects {
		if matchesAccessRef(ref, project.ID, project.ShortName, project.Name) {
			return Authf("access denied by local configuration: project is restricted")
		}
	}
	for _, tag := range tags {
		for _, ref := range p.DeniedTags {
			if matchesAccessRef(ref, tag.ID, tag.Name) {
				return Authf("access denied by local configuration: issue tag is restricted")
			}
		}
	}
	return nil
}

func matchesAccessRef(ref string, values ...string) bool {
	return slices.ContainsFunc(values, func(value string) bool {
		return value != "" && strings.EqualFold(strings.TrimSpace(ref), value)
	})
}

func (s *Service) WithAccessPolicy(policy AccessPolicy, store IssueAccessStore) *Service {
	s.access = AccessPolicy{DeniedProjects: slices.Clone(policy.DeniedProjects), DeniedTags: slices.Clone(policy.DeniedTags)}
	s.accessStore = store
	return s
}

func (s *Service) checkIssueAccess(ctx context.Context, ref domain.IssueRef) error {
	if !s.access.Restricted() {
		return nil
	}
	metadata, err := s.issueAccess(ctx, ref)
	if err != nil {
		return err
	}
	return s.access.Check(metadata.Project, metadata.Tags)
}

func (s *Service) issueAccess(ctx context.Context, ref domain.IssueRef) (IssueAccess, error) {
	if s.accessStore == nil {
		return IssueAccess{}, Runtimef("access policy requires an issue metadata store")
	}
	return s.accessStore.GetIssueAccess(ctx, ref, len(s.access.DeniedTags) > 0)
}

// restrictedIssueSummaries applies offset and limit to allowed results, retaining
// server ordering and continuing through pages that contain only denied issues.
func (s *Service) restrictedIssueSummaries(ctx context.Context, request PageRequest, fetch func(context.Context, Page) ([]domain.IssueSummary, error)) ([]domain.IssueSummary, error) {
	items := make([]domain.IssueSummary, 0)
	skipped := 0
	for offset := 0; ; {
		page, err := fetch(ctx, Page{Offset: offset, Limit: DefaultPageLimit})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return items, nil
		}
		offset += len(page)
		for _, item := range page {
			if s.access.Check(item.Project, nil) != nil {
				continue
			}
			if len(s.access.DeniedTags) > 0 {
				metadata, err := s.issueAccess(ctx, domain.IssueRef(item.ID))
				if err != nil {
					return nil, err
				}
				if s.access.Check(metadata.Project, metadata.Tags) != nil {
					continue
				}
			}
			if skipped < request.Offset {
				skipped++
				continue
			}
			items = append(items, item)
			if !request.All && len(items) >= request.pageLimit() {
				return items, nil
			}
		}
	}
}

// accessibleIssue checks policy before loading full issue details without field enrichment.
func (s *Service) accessibleIssue(ctx context.Context, ref domain.IssueRef) (domain.Issue, error) {
	if err := s.checkIssueAccess(ctx, ref); err != nil {
		return domain.Issue{}, err
	}
	issue, err := s.issues.GetIssue(ctx, ref)
	if err != nil {
		return domain.Issue{}, err
	}
	if err := s.access.Check(issue.Project, issue.Tags); err != nil {
		return domain.Issue{}, err
	}
	return issue, nil
}
