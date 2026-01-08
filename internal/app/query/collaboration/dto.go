package collaboration

// TeamDTO represents a team.
type TeamDTO struct {
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility"`
	MemberLimit int    `json:"member_limit"`
	MemberCount int    `json:"member_count,omitempty"`
	MyRole      string `json:"my_role,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

// MemberDTO represents a team member.
type MemberDTO struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	JoinedAt int64  `json:"joined_at"`
}

// InvitationDTO represents an invitation.
type InvitationDTO struct {
	ID           string `json:"id"`
	TeamID       string `json:"team_id"`
	TeamName     string `json:"team_name,omitempty"`
	InviterID    string `json:"inviter_id"`
	InviterName  string `json:"inviter_name,omitempty"`
	InviteeEmail string `json:"invitee_email"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	ExpiresAt    int64  `json:"expires_at"`
	CreatedAt    int64  `json:"created_at"`
}
