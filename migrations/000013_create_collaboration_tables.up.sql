-- Teams table
CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    member_limit INTEGER NOT NULL DEFAULT 5,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT teams_visibility_check CHECK (visibility IN ('public', 'private')),
    CONSTRAINT teams_status_check CHECK (status IN ('active', 'deleted'))
);

-- Unique slug per owner
CREATE UNIQUE INDEX idx_teams_owner_slug ON teams(owner_id, slug) WHERE status = 'active';

-- Index for listing teams by owner
CREATE INDEX idx_teams_owner_id ON teams(owner_id);

-- Team members table
CREATE TABLE IF NOT EXISTS team_members (
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (team_id, user_id),
    CONSTRAINT team_members_role_check CHECK (role IN ('owner', 'admin', 'member', 'guest'))
);

-- Index for finding all teams a user belongs to
CREATE INDEX idx_team_members_user_id ON team_members(user_id);

-- Team invitations table
CREATE TABLE IF NOT EXISTS team_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id),
    invitee_email VARCHAR(255) NOT NULL,
    invitee_id UUID REFERENCES users(id),
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    token VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,

    CONSTRAINT team_invitations_role_check CHECK (role IN ('admin', 'member', 'guest')),
    CONSTRAINT team_invitations_status_check CHECK (status IN ('pending', 'accepted', 'rejected', 'revoked', 'expired'))
);

-- Unique token index
CREATE UNIQUE INDEX idx_team_invitations_token ON team_invitations(token);

-- Index for finding invitations by email
CREATE INDEX idx_team_invitations_invitee_email ON team_invitations(invitee_email);

-- Index for finding invitations by team
CREATE INDEX idx_team_invitations_team_id ON team_invitations(team_id);

-- Index for finding pending invitations
CREATE INDEX idx_team_invitations_pending ON team_invitations(status) WHERE status = 'pending';
