# ai-media Specification

## Purpose
TBD - created by archiving change add-ai-module. Update Purpose after archive.
## Requirements
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

### Requirement: Media Adapter Interface

The system SHALL provide MediaAdapter interface for media generation providers.

#### Scenario: Supported types query

- **WHEN** system queries adapter capabilities
- **THEN** return list of supported MediaGenerationTypes

#### Scenario: Generate image via adapter

- **WHEN** MediaService calls adapter.GenerateImage
- **THEN** adapter transforms request to provider format
- **AND** returns MediaResult with outputs or task_id

#### Scenario: Generate video via adapter

- **WHEN** MediaService calls adapter.GenerateVideo
- **THEN** adapter initiates async generation
- **AND** returns task_id for polling

#### Scenario: Poll external task status

- **WHEN** system polls adapter.GetTaskStatus with external_task_id
- **THEN** return current status, progress, and outputs if complete

### Requirement: OpenAI DALL-E Adapter

The system SHALL provide adapter for OpenAI DALL-E image generation.

#### Scenario: DALL-E text-to-image

- **WHEN** adapter receives image request
- **THEN** call OpenAI images/generations API
- **AND** return generated image URLs

#### Scenario: DALL-E parameters

- **WHEN** request includes size, quality, n parameters
- **THEN** pass to DALL-E API (1024x1024, hd, etc.)

### Requirement: Video Generation Adapter

The system SHALL provide adapter for video generation services.

#### Scenario: Video generation request

- **WHEN** adapter receives video request with prompt
- **THEN** initiate async video generation
- **AND** return external task ID for status polling

#### Scenario: Video generation status

- **WHEN** polling video task status
- **THEN** return progress percentage
- **AND** return video URL when complete

### Requirement: Media Routing

The system SHALL route media requests using simplified strategy chain.

#### Scenario: Route image request

- **WHEN** MediaService routes image generation
- **THEN** apply HealthFilter and CapabilityFilter
- **AND** select provider supporting image_generation

#### Scenario: Route video request

- **WHEN** MediaService routes video generation
- **THEN** apply HealthFilter and CapabilityFilter
- **AND** select provider supporting video_generation

### Requirement: Media Output Storage

The system SHALL handle media output storage and URLs.

#### Scenario: External URL response

- **WHEN** provider returns external URL
- **THEN** return URL directly to client

#### Scenario: URL expiration handling

- **WHEN** provider URL has expiration
- **THEN** include expiration info in response
- **AND** optionally download and store to R2

