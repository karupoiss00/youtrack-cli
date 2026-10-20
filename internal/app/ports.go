package app

import (
	"context"
	"net/http"
	"time"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type FieldAssignment struct {
	Field domain.FieldDefinition
	Value domain.FieldValue
}

type CreateIssueRequest struct {
	Project     domain.ProjectRef
	Summary     string
	Description *string
	Fields      []FieldInput
	Tags        []string
}

type IssueCreate struct {
	Project     domain.Project
	Summary     string
	Description *string
	Fields      []FieldAssignment
	Tags        []domain.Tag
}

type IssueCreator interface {
	CreateIssue(context.Context, IssueCreate) (domain.Issue, error)
}

type IssuePatch struct {
	Summary     *string
	Description *string
	Fields      []FieldAssignment
	Tags        *[]domain.Tag
}

func (p IssuePatch) Empty() bool {
	return p.Summary == nil && p.Description == nil && len(p.Fields) == 0 && p.Tags == nil
}

type IssueStore interface {
	GetIssue(context.Context, domain.IssueRef) (domain.Issue, error)
	UpdateIssue(context.Context, domain.IssueRef, IssuePatch) error
	MoveIssue(context.Context, domain.IssueRef, string) (domain.Issue, error)
	AddIssueTag(context.Context, domain.IssueRef, domain.Tag) error
	RemoveIssueTag(context.Context, domain.IssueRef, domain.Tag) error
}

type PageRequest struct {
	Offset int
	Limit  *int
	All    bool
}

type Page struct {
	Offset int
	Limit  int
}

type IssueSearchStore interface {
	SearchIssues(context.Context, string, Page) ([]domain.IssueSummary, error)
}

type SavedSearchStore interface {
	GetSavedSearch(context.Context, string) (domain.SavedSearch, error)
	ListSavedSearches(context.Context, Page) ([]domain.SavedSearch, error)
}

type ProjectFieldStore interface {
	ListProjectFields(context.Context, string) ([]domain.FieldDefinition, error)
	ListFieldOptions(context.Context, string, domain.FieldDefinition) ([]domain.FieldOption, error)
	ListFieldUsers(context.Context, string, domain.FieldDefinition) ([]domain.User, error)
}

type ProjectStore interface {
	GetProject(context.Context, domain.ProjectRef) (domain.Project, error)
	SearchProjects(context.Context, string) ([]domain.Project, error)
}

type TagStore interface {
	SearchTags(context.Context, string) ([]domain.Tag, error)
}

type UserStore interface {
	Me(context.Context) (domain.User, error)
}

type GroupStore interface {
	SearchGroups(context.Context, string) ([]domain.Group, error)
}

type BoardStore interface {
	ListBoards(context.Context) ([]domain.Board, error)
	ValidateIssueBoardChange(context.Context, string, []domain.Board, []domain.Board) error
	ApplyIssueBoardChange(context.Context, string, []domain.Board, []domain.Board) error
}

type CommentStore interface {
	ListComments(context.Context, domain.IssueRef, Page) ([]domain.Comment, error)
	CreateComment(context.Context, domain.IssueRef, string) (domain.Comment, error)
	EditComment(context.Context, domain.IssueRef, string, string) (domain.Comment, error)
	SoftRemoveComment(context.Context, domain.IssueRef, string) error
}

type WorkItemCreate struct {
	Date            time.Time
	DurationMinutes int64
	Text            *string
	Type            *domain.WorkItemType
	Author          *domain.User
}

type WorkItemTypePatch struct {
	Set   bool
	Value *domain.WorkItemType
}

type WorkItemPatch struct {
	DurationMinutes *int64
	Date            *time.Time
	Text            *string
	Type            WorkItemTypePatch
	Author          *domain.User
}

type WorkItemStore interface {
	ListWorkItems(context.Context, domain.IssueRef, Page) ([]domain.WorkItem, error)
	GetWorkItem(context.Context, domain.IssueRef, string) (domain.WorkItem, error)
	CreateWorkItem(context.Context, domain.IssueRef, WorkItemCreate) (domain.WorkItem, error)
	UpdateWorkItem(context.Context, domain.IssueRef, string, WorkItemPatch) (domain.WorkItem, error)
	DeleteWorkItem(context.Context, domain.IssueRef, string) error
}

type ProjectTimeTrackingStore interface {
	TimeTrackingEnabled(context.Context, string) (bool, error)
	ListWorkItemTypes(context.Context, string, Page) ([]domain.WorkItemType, error)
}

type UserDirectoryStore interface {
	ListUsers(context.Context, Page) ([]domain.User, error)
}

type IssueIdentityStore interface {
	GetIssueIdentity(context.Context, domain.IssueRef) (domain.IssueIdentity, error)
}

type IssueAccess struct {
	Project domain.Project
	Tags    []domain.Tag
}

// IssueAccessStore loads only policy metadata, including every tag when requested.
type IssueAccessStore interface {
	GetIssueAccess(context.Context, domain.IssueRef, bool) (IssueAccess, error)
}

type LinkStore interface {
	ListLinkTypes(context.Context, Page) ([]domain.LinkType, error)
	ListLinkedIssues(context.Context, domain.IssueRef, domain.LinkRelation, Page) ([]domain.IssueSummary, error)
	AddIssueLink(context.Context, domain.IssueRef, domain.LinkRelation, domain.IssueRef) error
	RemoveIssueLink(context.Context, domain.IssueRef, domain.LinkRelation, domain.IssueRef) error
}

type RawResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type RawAPI interface {
	DoRaw(context.Context, string, string, []byte, http.Header) (RawResponse, error)
}
