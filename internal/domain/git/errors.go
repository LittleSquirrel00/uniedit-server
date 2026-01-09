package git

import "errors"

// Repository errors.
var (
	ErrRepoNotFound         = errors.New("repository not found")
	ErrRepoAlreadyExists    = errors.New("repository already exists")
	ErrInvalidRepoName      = errors.New("invalid repository name")
	ErrInvalidRepoType      = errors.New("invalid repository type")
	ErrStorageQuotaExceeded = errors.New("storage quota exceeded")
)

// Access control errors.
var (
	ErrAccessDenied      = errors.New("access denied")
	ErrNotOwner          = errors.New("not repository owner")
	ErrNotCollaborator   = errors.New("not a collaborator")
	ErrInvalidPermission = errors.New("invalid permission level")
)

// LFS errors.
var (
	ErrLFSNotEnabled     = errors.New("LFS is not enabled for this repository")
	ErrLFSObjectNotFound = errors.New("LFS object not found")
	ErrLFSFileTooLarge   = errors.New("file size exceeds maximum allowed")
	ErrLFSQuotaExceeded  = errors.New("LFS storage quota exceeded")
)

// Lock errors.
var (
	ErrLockNotFound      = errors.New("lock not found")
	ErrLockAlreadyExists = errors.New("file is already locked")
	ErrLockNotOwned      = errors.New("lock is owned by another user")
)

// Pull request errors.
var (
	ErrPRNotFound      = errors.New("pull request not found")
	ErrPRAlreadyMerged = errors.New("pull request is already merged")
	ErrPRAlreadyClosed = errors.New("pull request is already closed")
	ErrInvalidBranch   = errors.New("invalid branch")
	ErrSameBranch      = errors.New("source and target branches are the same")
)

// Git protocol errors.
var (
	ErrInvalidService = errors.New("invalid git service")
	ErrPushRejected   = errors.New("push rejected")
)
