package git

import (
	"time"

	commonv1 "github.com/uniedit/server/api/pb/common"
	gitv1 "github.com/uniedit/server/api/pb/git"
	"github.com/uniedit/server/internal/model"
)

func toRepoPB(repo *model.GitRepo, baseURL string) *gitv1.Repo {
	if repo == nil {
		return nil
	}

	out := &gitv1.Repo{
		Id:            repo.ID.String(),
		OwnerId:       repo.OwnerID.String(),
		Name:          repo.Name,
		Slug:          repo.Slug,
		RepoType:      string(repo.RepoType),
		Visibility:    string(repo.Visibility),
		Description:   repo.Description,
		DefaultBranch: repo.DefaultBranch,
		SizeBytes:     repo.SizeBytes,
		LfsEnabled:    repo.LFSEnabled,
		LfsSizeBytes:  repo.LFSSizeBytes,
		TotalSize:     repo.TotalSize(),
		StarsCount:    int32(repo.StarsCount),
		ForksCount:    int32(repo.ForksCount),
	}

	if baseURL != "" {
		out.CloneUrl = baseURL + "/git/" + repo.OwnerID.String() + "/" + repo.Slug + ".git"
		if repo.LFSEnabled {
			out.LfsUrl = baseURL + "/lfs/" + repo.OwnerID.String() + "/" + repo.Slug
		}
	}

	return out
}

func toCollaboratorPB(in *model.GitRepoCollaborator) *gitv1.Collaborator {
	if in == nil {
		return nil
	}
	return &gitv1.Collaborator{
		RepoId:     in.RepoID.String(),
		UserId:     in.UserID.String(),
		Permission: string(in.Permission),
		CreatedAt:  formatTime(in.CreatedAt),
	}
}

func toPullRequestPB(in *model.GitPullRequest) *gitv1.PullRequest {
	if in == nil {
		return nil
	}

	out := &gitv1.PullRequest{
		Id:           in.ID.String(),
		RepoId:       in.RepoID.String(),
		Number:       int32(in.Number),
		Title:        in.Title,
		Description:  in.Description,
		SourceBranch: in.SourceBranch,
		TargetBranch: in.TargetBranch,
		Status:       string(in.Status),
		AuthorId:     in.AuthorID.String(),
		CreatedAt:    formatTime(in.CreatedAt),
		UpdatedAt:    formatTime(in.UpdatedAt),
	}

	if in.MergedBy != nil {
		out.MergedBy = in.MergedBy.String()
	}
	if in.MergedAt != nil {
		out.MergedAt = formatTime(*in.MergedAt)
	}
	if in.ClosedAt != nil {
		out.ClosedAt = formatTime(*in.ClosedAt)
	}

	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func empty() *commonv1.Empty { return &commonv1.Empty{} }

