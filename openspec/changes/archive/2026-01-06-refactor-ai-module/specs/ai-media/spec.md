## MODIFIED Requirements

### Requirement: Media Service

The system SHALL provide unified media generation services for images, videos, and audio.

**Deprecation Notice**: This requirement is deprecated. New implementations SHOULD use the independent `media` module instead.

**Migration Path**:
- Import from `internal/module/media` instead of `internal/module/ai/media`
- API routes will change from `/api/v1/ai/media/*` to `/api/v1/media/*` in a future version
- See new `media` capability spec for updated requirements

#### Scenario: Generate image

- **WHEN** user requests image generation with prompt
- **THEN** route to appropriate provider (DALL-E, etc.)
- **AND** return task_id for async tracking

#### Scenario: Generate video

- **WHEN** user requests video generation with prompt
- **THEN** route to video provider (Runway, etc.)
- **AND** return task_id for async tracking
