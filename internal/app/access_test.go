package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type accessStoreFake struct {
	metadata map[domain.IssueRef]IssueAccess
	calls    []domain.IssueRef
	tags     []bool
	err      error
}

func (f *accessStoreFake) GetIssueAccess(_ context.Context, ref domain.IssueRef, tags bool) (IssueAccess, error) {
	f.calls = append(f.calls, ref)
	f.tags = append(f.tags, tags)
	return f.metadata[ref], f.err
}

func TestAccessPolicyMatchesProjectAndAnyTagReferences(t *testing.T) {
	project := domain.Project{ID: "0-1", Name: "Secret Project", ShortName: "SECRET"}
	tags := []domain.Tag{{ID: "tag-1", Name: "public"}, {ID: "tag-2", Name: "Sensitive Data"}}
	for _, ref := range []string{"0-1", "secret", "SECRET PROJECT", " secret "} {
		err := (AccessPolicy{DeniedProjects: []string{ref}}).Check(project, tags)
		require.Error(t, err)
		assert.Equal(t, ErrorAuth, KindOf(err))
	}
	for _, ref := range []string{"tag-2", "sensitive data"} {
		require.Error(t, (AccessPolicy{DeniedTags: []string{ref}}).Check(project, tags))
	}
	for _, policy := range []AccessPolicy{{}, {DeniedProjects: []string{"SEC", "*"}}, {DeniedTags: []string{"sensitive", "other"}}} {
		require.NoError(t, policy.Check(project, tags))
	}
}

func TestDeniedIssueOperationsNeverReachContentOrMutationStores(t *testing.T) {
	ctx := context.Background()
	for name, run := range map[string]func(*Service) error{
		"view":        func(s *Service) error { _, err := s.GetIssue(ctx, "2-1"); return err },
		"open":        func(s *Service) error { _, err := s.ResolveIssueIdentity(ctx, "2-1"); return err },
		"fields":      func(s *Service) error { _, err := s.ListFields(ctx, "2-1"); return err },
		"get field":   func(s *Service) error { _, err := s.GetField(ctx, "2-1", "Priority"); return err },
		"set field":   func(s *Service) error { _, err := s.SetField(ctx, "2-1", "Priority", []string{"High"}); return err },
		"clear field": func(s *Service) error { _, err := s.ClearField(ctx, "2-1", "Priority"); return err },
		"edit": func(s *Service) error {
			_, err := s.EditIssue(ctx, "2-1", EditRequest{Summary: new("updated")})
			return err
		},
		"move":           func(s *Service) error { _, err := s.MoveIssue(ctx, "2-1", "APP"); return err },
		"tags":           func(s *Service) error { _, err := s.ListTags(ctx, "2-1"); return err },
		"add tag":        func(s *Service) error { return s.AddTag(ctx, "2-1", "public") },
		"remove tag":     func(s *Service) error { return s.RemoveTag(ctx, "2-1", "private") },
		"comments":       func(s *Service) error { _, err := s.ListComments(ctx, "2-1", PageRequest{}); return err },
		"add comment":    func(s *Service) error { _, err := s.AddComment(ctx, "2-1", "text"); return err },
		"edit comment":   func(s *Service) error { _, err := s.EditComment(ctx, "2-1", "comment", "text"); return err },
		"remove comment": func(s *Service) error { return s.RemoveComment(ctx, "2-1", "comment") },
		"time types":     func(s *Service) error { _, err := s.ListWorkItemTypes(ctx, "2-1", PageRequest{}); return err },
		"time list":      func(s *Service) error { _, err := s.ListWorkItems(ctx, "2-1", PageRequest{}); return err },
		"time view":      func(s *Service) error { _, err := s.GetWorkItem(ctx, "2-1", "item"); return err },
		"time add": func(s *Service) error {
			_, err := s.AddWorkItem(ctx, "2-1", AddWorkItemRequest{Date: "2026-10-20", Duration: "1h"})
			return err
		},
		"time edit": func(s *Service) error {
			_, err := s.EditWorkItem(ctx, "2-1", "item", EditWorkItemRequest{Text: new("text")})
			return err
		},
		"time remove": func(s *Service) error { _, err := s.RemoveWorkItem(ctx, "2-1", "item"); return err },
		"links": func(s *Service) error {
			_, err := s.ListIssueLinks(ctx, "2-1", LinkRelationReference{Type: "relates"}, PageRequest{})
			return err
		},
		"link add": func(s *Service) error {
			_, err := s.AddIssueLink(ctx, "2-1", "2-2", LinkRelationReference{Type: "relates"})
			return err
		},
		"link remove": func(s *Service) error {
			_, err := s.RemoveIssueLink(ctx, "2-1", "2-2", LinkRelationReference{Type: "relates"})
			return err
		},
	} {
		for _, policy := range []AccessPolicy{{DeniedProjects: []string{"SECRET"}}, {DeniedTags: []string{"private"}}} {
			t.Run(name+"/"+policyTestName(policy), func(t *testing.T) {
				metadata := &accessStoreFake{metadata: map[domain.IssueRef]IssueAccess{"2-1": {Project: domain.Project{ShortName: "SECRET"}, Tags: []domain.Tag{{Name: "private"}}}}}
				// Nil downstream ports make any accidental read/write fail the test.
				service := new(Service).WithAccessPolicy(policy, metadata)
				err := run(service)
				require.Error(t, err)
				assert.Equal(t, ErrorAuth, KindOf(err))
				assert.Equal(t, []domain.IssueRef{"2-1"}, metadata.calls)
				assert.Equal(t, []bool{len(policy.DeniedTags) > 0}, metadata.tags)
			})
		}
	}
}

func policyTestName(policy AccessPolicy) string {
	if len(policy.DeniedTags) > 0 {
		return "tags"
	}
	return "projects"
}

func TestAccessPolicyFailsClosedOnMetadataFailure(t *testing.T) {
	want := errors.New("metadata unavailable")
	store := &accessStoreFake{err: want}
	service := new(Service).WithAccessPolicy(AccessPolicy{DeniedTags: []string{"private"}}, store)
	_, err := service.GetIssue(context.Background(), "2-1")
	require.ErrorIs(t, err, want)
	_, err = new(Service).WithAccessPolicy(AccessPolicy{DeniedTags: []string{"private"}}, nil).GetIssue(context.Background(), "2-1")
	require.ErrorContains(t, err, "metadata store")
}

func TestRestrictedSearchFiltersBeforePaginationAndKeepsQueryOrder(t *testing.T) {
	blocked := summary("SECRET-1")
	blocked.Project.ShortName = "SECRET"
	store := &searchStoreFake{scripts: []searchScript{
		{items: []domain.IssueSummary{blocked, summary("private")}},
		{items: []domain.IssueSummary{summary("skip"), summary("first")}},
		{items: []domain.IssueSummary{summary("second")}},
	}}
	metadata := &accessStoreFake{metadata: map[domain.IssueRef]IssueAccess{"private": {Tags: []domain.Tag{{Name: "private"}}}}}
	service := searchService(store).WithAccessPolicy(AccessPolicy{DeniedProjects: []string{"SECRET"}, DeniedTags: []string{"private"}}, metadata)
	query := "project: TT or project: SECRET sort by: updated desc"
	got, err := service.SearchIssues(context.Background(), query, PageRequest{Offset: 1, Limit: new(2)})
	require.NoError(t, err)
	assert.Equal(t, []domain.IssueSummary{summary("first"), summary("second")}, got)
	assert.Equal(t, []Page{{Offset: 0, Limit: 50}, {Offset: 2, Limit: 50}, {Offset: 4, Limit: 50}}, store.pages)
	assert.Equal(t, []string{query, query, query}, store.queries)
	assert.Equal(t, []domain.IssueRef{"private", "skip", "first", "second"}, metadata.calls)
}

func TestRestrictedSearchAllEmptyAndFailures(t *testing.T) {
	policy := AccessPolicy{DeniedProjects: []string{"TT"}}
	store := &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{summary("TT-1")}}, {}}}
	got, err := searchService(store).WithAccessPolicy(policy, nil).SearchIssues(context.Background(), "all", PageRequest{All: true})
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
	assert.Len(t, store.pages, 2)
	want := errors.New("offline")
	store = &searchStoreFake{scripts: []searchScript{{err: want}}}
	got, err = searchService(store).WithAccessPolicy(policy, nil).SearchIssues(context.Background(), "all", PageRequest{})
	require.ErrorIs(t, err, want)
	assert.Nil(t, got)
	store = &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{summary("TT-1")}}}}
	got, err = searchService(store).WithAccessPolicy(AccessPolicy{DeniedTags: []string{"private"}}, &accessStoreFake{err: want}).SearchIssues(context.Background(), "all", PageRequest{})
	require.ErrorIs(t, err, want)
	assert.Nil(t, got)
}

func TestAccessPolicyRejectsDestinationAndNewTagsBeforeWrites(t *testing.T) {
	for _, name := range []string{"move", "add tag", "edit tag"} {
		t.Run(name, func(t *testing.T) {
			fake := serviceFixture()
			fake.getProject = domain.Project{ID: "0-2", ShortName: "SECRET"}
			metadata := &accessStoreFake{metadata: map[domain.IssueRef]IssueAccess{"TT-1": {Project: fake.issue.Project}}}
			service := newService(fake).WithAccessPolicy(AccessPolicy{DeniedProjects: []string{"SECRET"}, DeniedTags: []string{"tag-backend"}}, metadata)
			var err error
			switch name {
			case "move":
				_, err = service.MoveIssue(context.Background(), "TT-1", "0-2")
			case "add tag":
				err = service.AddTag(context.Background(), "TT-1", "backend")
			case "edit tag":
				_, err = service.EditIssue(context.Background(), "TT-1", EditRequest{Summary: new("new"), AddTags: []string{"backend"}, Fields: []FieldInput{{Name: "Board", Value: "New"}}})
			}
			require.Error(t, err)
			assert.Equal(t, ErrorAuth, KindOf(err))
			assert.Zero(t, fake.updateCalls+fake.moveCalls+fake.addTagCalls+fake.boardValidates+fake.boardApplies)
		})
	}
}

func TestAccessPolicyRejectsCreateBeforeWrite(t *testing.T) {
	for _, policy := range []AccessPolicy{{DeniedProjects: []string{"APP"}}, {DeniedTags: []string{"backend-id"}}} {
		var calls []string
		creator := &createIssueCreatorFake{}
		service := createServiceFixture(creator, &calls).WithAccessPolicy(policy, nil)
		_, err := service.CreateIssue(context.Background(), CreateIssueRequest{Project: "APP", Summary: "summary", Tags: []string{"backend"}})
		require.Error(t, err)
		assert.Equal(t, ErrorAuth, KindOf(err))
		assert.Empty(t, creator.calls)
	}
}

func TestAccessPolicyChecksLinkTargetAndFiltersLinkedIssues(t *testing.T) {
	ctx := context.Background()
	for _, add := range []bool{false, true} {
		service, _, links := linkService(nil, nil)
		metadata := &accessStoreFake{metadata: map[domain.IssueRef]IssueAccess{"APP-2": {Tags: []domain.Tag{{Name: "private"}}}}}
		service.WithAccessPolicy(AccessPolicy{DeniedTags: []string{"private"}}, metadata)
		var err error
		if add {
			_, err = service.AddIssueLink(ctx, "APP-1", "APP-2", LinkRelationReference{Type: "relates"})
		} else {
			_, err = service.RemoveIssueLink(ctx, "APP-1", "APP-2", LinkRelationReference{Type: "relates"})
		}
		require.Error(t, err)
		assert.Equal(t, ErrorAuth, KindOf(err))
		assert.Zero(t, links.addCalls+links.removeCalls)
		assert.Empty(t, links.typePages)
	}
	blocked := linkSummary("private")
	service, _, _ := linkService([]domain.LinkType{{ID: "relates", Name: "Relates", Outward: "relates to"}}, []domain.IssueSummary{blocked, linkSummary("visible")})
	metadata := &accessStoreFake{metadata: map[domain.IssueRef]IssueAccess{"private": {Tags: []domain.Tag{{Name: "private"}}}}}
	service.WithAccessPolicy(AccessPolicy{DeniedTags: []string{"private"}}, metadata)
	got, err := service.ListIssueLinks(ctx, "APP-1", LinkRelationReference{Type: "relates"}, PageRequest{Limit: new(1)})
	require.NoError(t, err)
	assert.Equal(t, []domain.IssueSummary{linkSummary("visible")}, got.Issues)
}

func TestAllowedIssueRetainsNormalBehaviorWithRestrictions(t *testing.T) {
	fake := serviceFixture()
	metadata := &accessStoreFake{metadata: map[domain.IssueRef]IssueAccess{"TT-1": {Project: fake.issue.Project}}}
	policy := AccessPolicy{DeniedProjects: []string{"SECRET"}}
	service := newService(fake).WithAccessPolicy(policy, metadata)
	policy.DeniedProjects[0] = "TT" // The service owns a copy of its policy.
	got, err := service.EditIssue(context.Background(), "TT-1", EditRequest{Summary: new("updated")})
	require.NoError(t, err)
	assert.Equal(t, "updated", got.Summary)
	assert.Equal(t, 1, fake.updateCalls)
	assert.Equal(t, []domain.IssueRef{"TT-1", "TT-1"}, metadata.calls)
}

func TestRestrictedSavedSearchFiltersStoredQueryResults(t *testing.T) {
	saved := &savedSearchStoreFake{direct: domain.SavedSearch{ID: "saved", Name: "Mine", Query: "project: TT or project: SECRET"}}
	blocked := summary("SECRET-1")
	blocked.Project.ShortName = "SECRET"
	issues := &searchStoreFake{scripts: []searchScript{{items: []domain.IssueSummary{blocked, summary("TT-1")}}, {}}}
	service := savedSearchService(saved, issues).WithAccessPolicy(AccessPolicy{DeniedProjects: []string{"SECRET"}}, nil)
	got, err := service.ViewSavedSearch(context.Background(), "saved", PageRequest{All: true})
	require.NoError(t, err)
	assert.Equal(t, []domain.IssueSummary{summary("TT-1")}, got.Issues)
	assert.Equal(t, []string{saved.direct.Query, saved.direct.Query}, issues.queries)
}

func TestCreateSuppressesIssueRestrictedByServerWorkflow(t *testing.T) {
	for _, tagInResponse := range []bool{false, true} {
		var calls []string
		creator := &createIssueCreatorFake{result: domain.Issue{ID: "2-1", Project: domain.Project{ID: "0-1", ShortName: "APP"}}}
		if tagInResponse {
			creator.result.Tags = []domain.Tag{{Name: "private"}}
		}
		metadata := &accessStoreFake{metadata: map[domain.IssueRef]IssueAccess{"2-1": {Tags: []domain.Tag{{Name: "private"}}}}}
		service := createServiceFixture(creator, &calls).WithAccessPolicy(AccessPolicy{DeniedTags: []string{"private"}}, metadata)
		got, err := service.CreateIssue(context.Background(), CreateIssueRequest{Project: "APP", Summary: "summary"})
		require.Error(t, err)
		assert.Equal(t, ErrorAuth, KindOf(err))
		assert.Empty(t, got)
		assert.Len(t, creator.calls, 1, "server workflow denial cannot roll back a completed write")
	}
}
