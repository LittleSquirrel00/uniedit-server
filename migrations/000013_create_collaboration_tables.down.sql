-- Drop indexes
DROP INDEX IF EXISTS idx_team_invitations_pending;
DROP INDEX IF EXISTS idx_team_invitations_team_id;
DROP INDEX IF EXISTS idx_team_invitations_invitee_email;
DROP INDEX IF EXISTS idx_team_invitations_token;
DROP INDEX IF EXISTS idx_team_members_user_id;
DROP INDEX IF EXISTS idx_teams_owner_id;
DROP INDEX IF EXISTS idx_teams_owner_slug;

-- Drop tables
DROP TABLE IF EXISTS team_invitations;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
