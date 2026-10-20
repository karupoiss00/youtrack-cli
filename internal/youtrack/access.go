package youtrack

import (
	"context"
	"net/http"
	"net/url"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack/dto"
)

func (c *Client) GetIssueAccess(ctx context.Context, ref domain.IssueRef, includeTags bool) (app.IssueAccess, error) {
	var data dto.Issue
	path := "/api/issues/" + string(ref)
	params := url.Values{"fields": []string{issueAccessFields}}
	if err := c.doJSON(ctx, http.MethodGet, path, params, nil, &data); err != nil {
		return app.IssueAccess{}, err
	}
	if data.Project.ID == "" {
		return app.IssueAccess{}, app.Runtimef("cannot check access: issue metadata has no project ID")
	}
	metadata := app.IssueAccess{Project: domain.Project{ID: data.Project.ID, Name: data.Project.Name, ShortName: data.Project.ShortName}}
	if !includeTags {
		return metadata, nil
	}
	tags, err := paginate(ctx, func(ctx context.Context, skip, top int) ([]dto.Tag, error) {
		var page []dto.Tag
		params := url.Values{"fields": []string{tagFields}, "$skip": []string{itoa(skip)}, "$top": []string{itoa(top)}}
		err := c.doJSON(ctx, http.MethodGet, path+"/tags", params, nil, &page)
		return page, err
	})
	if err != nil {
		return app.IssueAccess{}, err
	}
	for _, tag := range tags {
		metadata.Tags = append(metadata.Tags, domain.Tag{ID: tag.ID, Name: tag.Name})
	}
	return metadata, nil
}
