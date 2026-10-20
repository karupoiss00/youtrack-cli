package app

import (
	"context"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

type LinkRelationReference struct {
	Type      string
	Direction *string
}

func (s *Service) ListLinkTypes(ctx context.Context, request PageRequest) ([]domain.LinkType, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	return s.listLinkTypes(ctx, request)
}

func (s *Service) ListIssueLinks(ctx context.Context, source domain.IssueRef, reference LinkRelationReference, request PageRequest) (domain.IssueLinkList, error) {
	if err := validateLinkIssueReference("issue", source); err != nil {
		return domain.IssueLinkList{}, err
	}
	if err := validateLinkRelationReference(reference); err != nil {
		return domain.IssueLinkList{}, err
	}
	if err := request.Validate(); err != nil {
		return domain.IssueLinkList{}, err
	}

	identity, err := s.ResolveIssueIdentity(ctx, source)
	if err != nil {
		return domain.IssueLinkList{}, err
	}
	relation, err := s.resolveLinkRelation(ctx, reference)
	if err != nil {
		return domain.IssueLinkList{}, err
	}
	issues, err := s.listLinkedIssues(ctx, domain.IssueRef(identity.ID), relation, request)
	if err != nil {
		return domain.IssueLinkList{}, err
	}
	return domain.IssueLinkList{IssueID: identity.IDReadable, Relation: relation, Issues: issues}, nil
}

func (s *Service) AddIssueLink(ctx context.Context, source, target domain.IssueRef, reference LinkRelationReference) (domain.IssueLinkChange, error) {
	return s.changeIssueLink(ctx, source, target, reference, true)
}

func (s *Service) RemoveIssueLink(ctx context.Context, source, target domain.IssueRef, reference LinkRelationReference) (domain.IssueLinkChange, error) {
	return s.changeIssueLink(ctx, source, target, reference, false)
}

func (s *Service) changeIssueLink(ctx context.Context, source, target domain.IssueRef, reference LinkRelationReference, add bool) (domain.IssueLinkChange, error) {
	if err := validateLinkIssueReference("issue", source); err != nil {
		return domain.IssueLinkChange{}, err
	}
	if err := validateLinkIssueReference("target issue", target); err != nil {
		return domain.IssueLinkChange{}, err
	}
	if err := validateLinkRelationReference(reference); err != nil {
		return domain.IssueLinkChange{}, err
	}

	sourceIdentity, err := s.ResolveIssueIdentity(ctx, source)
	if err != nil {
		return domain.IssueLinkChange{}, err
	}
	targetIdentity, err := s.ResolveIssueIdentity(ctx, target)
	if err != nil {
		return domain.IssueLinkChange{}, err
	}
	relation, err := s.resolveLinkRelation(ctx, reference)
	if err != nil {
		return domain.IssueLinkChange{}, err
	}
	if sourceIdentity.ID == targetIdentity.ID {
		return domain.IssueLinkChange{}, Validationf("issue and target issue must be different")
	}

	present, err := s.linkPresent(ctx, domain.IssueRef(sourceIdentity.ID), relation, targetIdentity.ID)
	if err != nil {
		return domain.IssueLinkChange{}, err
	}
	change := domain.IssueLinkChange{IssueID: sourceIdentity.IDReadable, TargetIssueID: targetIdentity.IDReadable, Relation: relation, Present: present}
	if add {
		if present {
			return change, nil
		}
		if err := s.links.AddIssueLink(ctx, domain.IssueRef(sourceIdentity.ID), relation, domain.IssueRef(targetIdentity.ID)); err != nil {
			return domain.IssueLinkChange{}, err
		}
		change.Present = true
		change.Changed = true
		return change, nil
	}
	if !present {
		return change, nil
	}
	if err := s.links.RemoveIssueLink(ctx, domain.IssueRef(sourceIdentity.ID), relation, domain.IssueRef(targetIdentity.ID)); err != nil {
		return domain.IssueLinkChange{}, err
	}
	change.Present = false
	change.Changed = true
	return change, nil
}

func (s *Service) listLinkTypes(ctx context.Context, request PageRequest) ([]domain.LinkType, error) {
	items := make([]domain.LinkType, 0)
	offset := request.Offset
	remaining := request.pageLimit()
	for request.All || remaining > 0 {
		limit := DefaultPageLimit
		if !request.All && remaining < limit {
			limit = remaining
		}
		page, err := s.links.ListLinkTypes(ctx, Page{Offset: offset, Limit: limit})
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

func (s *Service) listLinkedIssues(ctx context.Context, source domain.IssueRef, relation domain.LinkRelation, request PageRequest) ([]domain.IssueSummary, error) {
	if s.access.Restricted() {
		return s.restrictedIssueSummaries(ctx, request, func(ctx context.Context, page Page) ([]domain.IssueSummary, error) {
			return s.links.ListLinkedIssues(ctx, source, relation, page)
		})
	}
	issues := make([]domain.IssueSummary, 0)
	offset := request.Offset
	remaining := request.pageLimit()
	for request.All || remaining > 0 {
		limit := DefaultPageLimit
		if !request.All && remaining < limit {
			limit = remaining
		}
		page, err := s.links.ListLinkedIssues(ctx, source, relation, Page{Offset: offset, Limit: limit})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		if !request.All && len(page) > remaining {
			page = page[:remaining]
		}
		issues = append(issues, page...)
		offset += len(page)
		if !request.All {
			remaining -= len(page)
		}
	}
	return issues, nil
}

func (s *Service) linkPresent(ctx context.Context, source domain.IssueRef, relation domain.LinkRelation, targetID string) (bool, error) {
	issues, err := s.listLinkedIssues(ctx, source, relation, PageRequest{All: true})
	if err != nil {
		return false, err
	}
	for _, issue := range issues {
		if issue.ID == targetID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) resolveLinkRelation(ctx context.Context, reference LinkRelationReference) (domain.LinkRelation, error) {
	types, err := s.listLinkTypes(ctx, PageRequest{All: true})
	if err != nil {
		return domain.LinkRelation{}, err
	}
	for _, linkType := range types {
		if linkType.ID == reference.Type {
			return relationForLinkType(linkType, reference.Direction)
		}
	}
	if linkType, count := linkTypeNameMatch(types, reference.Type, false); count == 1 {
		return relationForLinkType(linkType, reference.Direction)
	} else if count > 1 {
		return domain.LinkRelation{}, Ambiguousf("link type %q is ambiguous", reference.Type)
	}
	if linkType, count := linkTypeNameMatch(types, reference.Type, true); count == 1 {
		return relationForLinkType(linkType, reference.Direction)
	} else if count > 1 {
		return domain.LinkRelation{}, Ambiguousf("link type %q is ambiguous", reference.Type)
	}
	if relation, count := linkRelationLabelMatch(types, reference.Type); count == 1 {
		if reference.Direction != nil && *reference.Direction != string(relation.Direction) {
			return domain.LinkRelation{}, Validationf("link label %q has direction %q", reference.Type, relation.Direction)
		}
		return relation, nil
	} else if count > 1 {
		return domain.LinkRelation{}, Ambiguousf("link label %q is ambiguous", reference.Type)
	}
	return domain.LinkRelation{}, NotFoundf("link type %q was not found", reference.Type)
}

func relationForLinkType(linkType domain.LinkType, direction *string) (domain.LinkRelation, error) {
	if !linkType.Directed {
		if direction != nil {
			return domain.LinkRelation{}, Validationf("undirected link type %q does not accept a direction", linkType.Name)
		}
		return domain.LinkRelation{Type: linkType, Direction: domain.LinkDirectionUndirected, Label: linkType.Outward}, nil
	}
	if direction == nil {
		return domain.LinkRelation{}, Validationf("directed link type %q requires a direction", linkType.Name)
	}
	switch *direction {
	case string(domain.LinkDirectionOutward):
		return domain.LinkRelation{Type: linkType, Direction: domain.LinkDirectionOutward, Label: linkType.Outward}, nil
	case string(domain.LinkDirectionInward):
		return domain.LinkRelation{Type: linkType, Direction: domain.LinkDirectionInward, Label: linkType.Inward}, nil
	default:
		return domain.LinkRelation{}, Validationf("link direction must be outward or inward")
	}
}

func validateLinkIssueReference(kind string, reference domain.IssueRef) error {
	if strings.TrimSpace(string(reference)) == "" {
		return Validationf("%s reference must not be blank", kind)
	}
	return nil
}

func validateLinkRelationReference(reference LinkRelationReference) error {
	if strings.TrimSpace(reference.Type) == "" {
		return Validationf("link type must not be blank")
	}
	return nil
}

func linkTypeNameMatch(types []domain.LinkType, reference string, folded bool) (domain.LinkType, int) {
	var match domain.LinkType
	count := 0
	for _, linkType := range types {
		matches := linkType.Name == reference
		if folded {
			matches = strings.EqualFold(linkType.Name, reference)
		}
		if matches {
			match = linkType
			count++
		}
	}
	return match, count
}

func linkRelationLabelMatch(types []domain.LinkType, reference string) (domain.LinkRelation, int) {
	var match domain.LinkRelation
	count := 0
	for _, linkType := range types {
		if linkType.Outward == reference {
			direction := domain.LinkDirectionUndirected
			if linkType.Directed {
				direction = domain.LinkDirectionOutward
			}
			match = domain.LinkRelation{Type: linkType, Direction: direction, Label: linkType.Outward}
			count++
		}
		if linkType.Directed && linkType.Inward == reference {
			match = domain.LinkRelation{Type: linkType, Direction: domain.LinkDirectionInward, Label: linkType.Inward}
			count++
		}
	}
	return match, count
}
