package app

import (
	"context"
	"strings"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type FieldInput struct {
	Name  string
	Value string
}

type EditRequest struct {
	Summary     *string
	Description *string
	Fields      []FieldInput
	AddTags     []string
	RemoveTags  []string
}

type Service struct {
	access        AccessPolicy
	accessStore   IssueAccessStore
	issues        IssueStore
	creator       IssueCreator
	issueSearch   IssueSearchStore
	savedSearches SavedSearchStore
	fields        ProjectFieldStore
	projects      ProjectStore
	tags          TagStore
	users         UserStore
	groups        GroupStore
	boards        BoardStore
	comments      CommentStore
	workItems     WorkItemStore
	timeTracking  ProjectTimeTrackingStore
	userDirectory UserDirectoryStore
	issueIdentity IssueIdentityStore
	links         LinkStore
	resolver      *FieldResolver
}

func NewService(issues IssueStore, creator IssueCreator, issueSearch IssueSearchStore, savedSearches SavedSearchStore, fields ProjectFieldStore, projects ProjectStore, tags TagStore, users UserStore, groups GroupStore, boards BoardStore, comments CommentStore, workItems WorkItemStore, timeTracking ProjectTimeTrackingStore, userDirectory UserDirectoryStore, issueIdentity IssueIdentityStore, links LinkStore) *Service {
	return &Service{issues: issues, creator: creator, issueSearch: issueSearch, savedSearches: savedSearches, fields: fields, projects: projects, tags: tags, users: users, groups: groups, boards: boards, comments: comments, workItems: workItems, timeTracking: timeTracking, userDirectory: userDirectory, issueIdentity: issueIdentity, links: links, resolver: NewFieldResolver(fields, users, groups)}
}

func (s *Service) GetIssue(ctx context.Context, ref domain.IssueRef) (domain.Issue, error) {
	issue, err := s.accessibleIssue(ctx, ref)
	if err != nil {
		return domain.Issue{}, err
	}
	return s.enrichIssue(ctx, issue)
}

func (s *Service) ListFields(ctx context.Context, ref domain.IssueRef) ([]domain.IssueField, error) {
	issue, err := s.GetIssue(ctx, ref)
	if err != nil {
		return nil, err
	}
	return issue.Fields, nil
}

func (s *Service) GetField(ctx context.Context, ref domain.IssueRef, field domain.FieldRef) (domain.IssueField, error) {
	fields, err := s.ListFields(ctx, ref)
	if err != nil {
		return domain.IssueField{}, err
	}
	if isBoardRef(string(field)) {
		for _, f := range fields {
			if f.Kind == domain.FieldBoard {
				return f, nil
			}
		}
	}
	for _, f := range fields {
		if f.ID == string(field) || f.Name == string(field) {
			return f, nil
		}
	}
	var matches []domain.IssueField
	for _, f := range fields {
		if strings.EqualFold(f.Name, string(field)) {
			matches = append(matches, f)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.IssueField{}, Ambiguousf("field %q is ambiguous", field)
	}
	return domain.IssueField{}, NotFoundf("field %q was not found", field)
}

func (s *Service) SetField(ctx context.Context, ref domain.IssueRef, field domain.FieldRef, values []string) (domain.Issue, error) {
	issue, err := s.GetIssue(ctx, ref)
	if err != nil {
		return domain.Issue{}, err
	}
	if isBoardRef(string(field)) {
		change, err := s.resolveBoardChange(ctx, issue, values, false)
		if err != nil {
			return domain.Issue{}, err
		}
		if change.Empty() {
			return issue, nil
		}
		if err := s.boards.ValidateIssueBoardChange(ctx, issue.ID, change.Add, change.Remove); err != nil {
			return domain.Issue{}, err
		}
		if err := s.boards.ApplyIssueBoardChange(ctx, issue.ID, change.Add, change.Remove); err != nil {
			return domain.Issue{}, err
		}
		return s.GetIssue(ctx, ref)
	}
	assignment, err := s.resolver.Resolve(ctx, issue, field, values, false)
	if err != nil {
		return domain.Issue{}, err
	}
	if err := s.issues.UpdateIssue(ctx, ref, IssuePatch{Fields: []FieldAssignment{assignment}}); err != nil {
		return domain.Issue{}, err
	}
	return s.GetIssue(ctx, ref)
}

func (s *Service) ClearField(ctx context.Context, ref domain.IssueRef, field domain.FieldRef) (domain.Issue, error) {
	issue, err := s.GetIssue(ctx, ref)
	if err != nil {
		return domain.Issue{}, err
	}
	if isBoardRef(string(field)) {
		change, err := s.resolveBoardChange(ctx, issue, nil, true)
		if err != nil {
			return domain.Issue{}, err
		}
		if change.Empty() {
			return issue, nil
		}
		if err := s.boards.ValidateIssueBoardChange(ctx, issue.ID, change.Add, change.Remove); err != nil {
			return domain.Issue{}, err
		}
		if err := s.boards.ApplyIssueBoardChange(ctx, issue.ID, change.Add, change.Remove); err != nil {
			return domain.Issue{}, err
		}
		return s.GetIssue(ctx, ref)
	}
	assignment, err := s.resolver.Resolve(ctx, issue, field, nil, true)
	if err != nil {
		return domain.Issue{}, err
	}
	if err := s.issues.UpdateIssue(ctx, ref, IssuePatch{Fields: []FieldAssignment{assignment}}); err != nil {
		return domain.Issue{}, err
	}
	return s.GetIssue(ctx, ref)
}

func (s *Service) EditIssue(ctx context.Context, ref domain.IssueRef, req EditRequest) (domain.Issue, error) {
	if req.Summary == nil && req.Description == nil && len(req.Fields) == 0 && len(req.AddTags) == 0 && len(req.RemoveTags) == 0 {
		return domain.Issue{}, Validationf("no issue changes were provided")
	}
	issue, err := s.GetIssue(ctx, ref)
	if err != nil {
		return domain.Issue{}, err
	}

	grouped := map[string][]string{}
	originalName := map[string]string{}
	order := []string{}
	var boardValues []string
	for _, in := range req.Fields {
		name := strings.TrimSpace(in.Name)
		if name == "" {
			return domain.Issue{}, Validationf("custom field name cannot be empty")
		}
		if isBoardRef(name) {
			boardValues = append(boardValues, in.Value)
			continue
		}
		key := strings.ToLower(name)
		if _, ok := grouped[key]; !ok {
			order = append(order, key)
			originalName[key] = name
		}
		grouped[key] = append(grouped[key], in.Value)
	}
	assignments := make([]FieldAssignment, 0, len(grouped))
	for _, key := range order {
		assignment, err := s.resolver.Resolve(ctx, issue, domain.FieldRef(originalName[key]), grouped[key], false)
		if err != nil {
			return domain.Issue{}, err
		}
		assignments = append(assignments, assignment)
	}

	var desiredTags *[]domain.Tag
	if len(req.AddTags) > 0 || len(req.RemoveTags) > 0 {
		if err := validateTagConflicts(req.AddTags, req.RemoveTags); err != nil {
			return domain.Issue{}, err
		}
		tags, err := s.resolveDesiredTags(ctx, issue.Tags, req.AddTags, req.RemoveTags)
		if err != nil {
			return domain.Issue{}, err
		}
		desiredTags = &tags
		if err := s.access.Check(issue.Project, tags); err != nil {
			return domain.Issue{}, err
		}
	}

	var boardChange boardChange
	if len(boardValues) > 0 {
		boardChange, err = s.resolveBoardChange(ctx, issue, boardValues, false)
		if err != nil {
			return domain.Issue{}, err
		}
		if !boardChange.Empty() {
			if err := s.boards.ValidateIssueBoardChange(ctx, issue.ID, boardChange.Add, boardChange.Remove); err != nil {
				return domain.Issue{}, err
			}
		}
	}
	patch := IssuePatch{Summary: req.Summary, Description: req.Description, Fields: assignments, Tags: desiredTags}
	if !patch.Empty() {
		if err := s.issues.UpdateIssue(ctx, ref, patch); err != nil {
			return domain.Issue{}, err
		}
	}
	if !boardChange.Empty() {
		if err := s.boards.ApplyIssueBoardChange(ctx, issue.ID, boardChange.Add, boardChange.Remove); err != nil {
			return domain.Issue{}, err
		}
	}
	if patch.Empty() && boardChange.Empty() {
		return issue, nil
	}
	return s.GetIssue(ctx, ref)
}

func (s *Service) MoveIssue(ctx context.Context, ref domain.IssueRef, projectRef domain.ProjectRef) (domain.Issue, error) {
	if err := s.checkIssueAccess(ctx, ref); err != nil {
		return domain.Issue{}, err
	}
	project, err := s.resolveProject(ctx, projectRef)
	if err != nil {
		return domain.Issue{}, err
	}
	if err := s.access.Check(project, nil); err != nil {
		return domain.Issue{}, err
	}
	moved, err := s.issues.MoveIssue(ctx, ref, project.ID)
	if err != nil {
		return domain.Issue{}, err
	}
	if err := s.access.Check(moved.Project, moved.Tags); err != nil {
		return domain.Issue{}, err
	}
	if err := s.checkIssueAccess(ctx, domain.IssueRef(moved.ID)); err != nil {
		return domain.Issue{}, err
	}
	return s.enrichIssue(ctx, moved)
}

func (s *Service) ListTags(ctx context.Context, ref domain.IssueRef) ([]domain.Tag, error) {
	issue, err := s.GetIssue(ctx, ref)
	if err != nil {
		return nil, err
	}
	return issue.Tags, nil
}

func (s *Service) AddTag(ctx context.Context, ref domain.IssueRef, name string) error {
	issue, err := s.GetIssue(ctx, ref)
	if err != nil {
		return err
	}
	if _, ok := findTag(issue.Tags, name); ok {
		return nil
	}
	tag, err := s.resolveTag(ctx, name)
	if err != nil {
		return err
	}
	if err := s.access.Check(issue.Project, []domain.Tag{tag}); err != nil {
		return err
	}
	return s.issues.AddIssueTag(ctx, ref, tag)
}

func (s *Service) RemoveTag(ctx context.Context, ref domain.IssueRef, name string) error {
	issue, err := s.GetIssue(ctx, ref)
	if err != nil {
		return err
	}
	tag, ok := findTag(issue.Tags, name)
	if !ok {
		return nil
	}
	return s.issues.RemoveIssueTag(ctx, ref, tag)
}

func (s *Service) enrichIssue(ctx context.Context, issue domain.Issue) (domain.Issue, error) {
	defs, err := s.resolver.Definitions(ctx, issue.Project.ID)
	if err != nil {
		return domain.Issue{}, err
	}
	byName := make(map[string]domain.FieldDefinition, len(defs))
	byID := make(map[string]domain.FieldDefinition, len(defs))
	for _, d := range defs {
		byName[d.Name] = d
		byID[d.ID] = d
	}
	for i := range issue.Fields {
		if issue.Fields[i].Kind == domain.FieldBoard {
			continue
		}
		def, ok := byName[issue.Fields[i].Name]
		if !ok {
			def, ok = byID[issue.Fields[i].ID]
		}
		if !ok {
			continue
		}
		issue.Fields[i].Kind = def.Kind
		issue.Fields[i].Cardinality = def.Cardinality
		issue.Fields[i].Value = normalizeReadValue(def, issue.Fields[i].Value)
	}
	return issue, nil
}

func normalizeReadValue(def domain.FieldDefinition, v domain.FieldValue) domain.FieldValue {
	switch def.Kind {
	case domain.FieldDateTime:
		if n, ok := v.(domain.IntegerValue); ok {
			return domain.DateValue{Value: time.UnixMilli(n.Value)}
		}
	case domain.FieldFloat:
		if n, ok := v.(domain.IntegerValue); ok {
			return domain.FloatValue{Value: float64(n.Value)}
		}
	}
	return v
}

func (s *Service) resolveProject(ctx context.Context, ref domain.ProjectRef) (domain.Project, error) {
	p, err := s.projects.GetProject(ctx, ref)
	if err == nil {
		return p, nil
	}
	if KindOf(err) != ErrorNotFound {
		return domain.Project{}, err
	}
	items, err := s.projects.SearchProjects(ctx, string(ref))
	if err != nil {
		return domain.Project{}, err
	}
	for _, item := range items {
		if item.Name == string(ref) {
			return item, nil
		}
	}
	var matches []domain.Project
	for _, item := range items {
		if strings.EqualFold(item.Name, string(ref)) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.Project{}, Ambiguousf("project %q is ambiguous", ref)
	}
	return domain.Project{}, NotFoundf("project %q was not found", ref)
}

func (s *Service) resolveTag(ctx context.Context, ref string) (domain.Tag, error) {
	items, err := s.tags.SearchTags(ctx, ref)
	if err != nil {
		return domain.Tag{}, err
	}
	for _, item := range items {
		if item.ID == ref || item.Name == ref {
			return item, nil
		}
	}
	var matches []domain.Tag
	for _, item := range items {
		if strings.EqualFold(item.Name, ref) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return domain.Tag{}, Ambiguousf("tag %q is ambiguous", ref)
	}
	return domain.Tag{}, NotFoundf("tag %q was not found", ref)
}

func (s *Service) resolveDesiredTags(ctx context.Context, current []domain.Tag, add, remove []string) ([]domain.Tag, error) {
	result := append([]domain.Tag(nil), current...)
	for _, name := range remove {
		if tag, ok := findTag(result, name); ok {
			result = deleteTag(result, tag.ID)
		}
	}
	for _, name := range add {
		if _, ok := findTag(result, name); ok {
			continue
		}
		tag, err := s.resolveTag(ctx, name)
		if err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, nil
}

func findTag(tags []domain.Tag, ref string) (domain.Tag, bool) {
	for _, t := range tags {
		if t.ID == ref || t.Name == ref {
			return t, true
		}
	}
	for _, t := range tags {
		if strings.EqualFold(t.Name, ref) {
			return t, true
		}
	}
	return domain.Tag{}, false
}

func deleteTag(tags []domain.Tag, id string) []domain.Tag {
	out := tags[:0]
	for _, t := range tags {
		if t.ID != id {
			out = append(out, t)
		}
	}
	return out
}

func validateTagConflicts(add, remove []string) error {
	rm := map[string]struct{}{}
	for _, v := range remove {
		rm[strings.ToLower(v)] = struct{}{}
	}
	for _, v := range add {
		if _, ok := rm[strings.ToLower(v)]; ok {
			return Validationf("tag %q cannot be added and removed in the same edit", v)
		}
	}
	return nil
}
