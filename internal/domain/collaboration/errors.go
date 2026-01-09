package collaboration

import "errors"

// Domain errors for collaboration module.
var (
	// Team errors
	ErrTeamNotFound      = errors.New("team not found")
	ErrSlugAlreadyExists = errors.New("slug already exists")
	ErrTeamDeleted       = errors.New("team has been deleted")

	// Member errors
	ErrMemberNotFound       = errors.New("member not found")
	ErrAlreadyMember        = errors.New("user is already a member")
	ErrMemberLimitExceeded  = errors.New("team member limit exceeded")
	ErrCannotChangeOwner    = errors.New("cannot change owner role")
	ErrCannotRemoveOwner    = errors.New("cannot remove team owner")
	ErrOnlyOwnerCanDelete   = errors.New("only owner can delete team")
	ErrOnlyOwnerCanTransfer = errors.New("only owner can transfer ownership")

	// Permission errors
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrInvalidRole            = errors.New("invalid role")

	// Invitation errors
	ErrInvitationNotFound         = errors.New("invitation not found")
	ErrInvitationExpired          = errors.New("invitation has expired")
	ErrInvitationAlreadyProcessed = errors.New("invitation has already been processed")
	ErrInvitationAlreadyPending   = errors.New("invitation already pending for this email")
	ErrInvitationNotForYou        = errors.New("invitation is not for you")
	ErrCannotRevokeProcessed      = errors.New("cannot revoke processed invitation")
)
