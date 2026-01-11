package collaboration

import (
	"errors"

	"github.com/uniedit/server/internal/port/outbound"
)

// Domain errors for collaboration module.
var (
	// Team errors - reuse from outbound port for adapter compatibility
	ErrTeamNotFound      = outbound.ErrTeamNotFound
	ErrSlugAlreadyExists = errors.New("slug already exists")
	ErrTeamDeleted       = errors.New("team has been deleted")

	// Member errors - reuse from outbound port for adapter compatibility
	ErrMemberNotFound       = outbound.ErrMemberNotFound
	ErrAlreadyMember        = errors.New("user is already a member")
	ErrMemberLimitExceeded  = errors.New("team member limit exceeded")
	ErrCannotChangeOwner    = errors.New("cannot change owner role")
	ErrCannotRemoveOwner    = errors.New("cannot remove team owner")
	ErrOnlyOwnerCanDelete   = errors.New("only owner can delete team")
	ErrOnlyOwnerCanTransfer = errors.New("only owner can transfer ownership")

	// Permission errors
	ErrInsufficientPermission = errors.New("insufficient permission")
	ErrInvalidRole            = errors.New("invalid role")

	// Invitation errors - reuse from outbound port for adapter compatibility
	ErrInvitationNotFound         = outbound.ErrInvitationNotFound
	ErrInvitationExpired          = errors.New("invitation has expired")
	ErrInvitationAlreadyProcessed = errors.New("invitation has already been processed")
	ErrInvitationAlreadyPending   = errors.New("invitation already pending for this email")
	ErrInvitationNotForYou        = errors.New("invitation is not for you")
	ErrCannotRevokeProcessed      = errors.New("cannot revoke processed invitation")
)
