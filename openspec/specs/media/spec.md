# media Specification

## Purpose
TBD - created by archiving change refactor-ai-module. Update Purpose after archive.
## Requirements
### Requirement: Media Module Independence

The system SHALL provide media generation as an independent module separate from AI/LLM module.

#### Scenario: Module initialization

- **WHEN** server starts
- **THEN** media module initializes independently from ai module
- **AND** registers its own HTTP routes under `/api/v1/media/*`

#### Scenario: Independent lifecycle

- **WHEN** media module is modified or deployed
- **THEN** ai module is not affected
- **AND** shared/task infrastructure handles async operations

### Requirement: Media Service

The system SHALL provide unified media generation services for images, videos, and audio.

#### Scenario: Generate image

- **WHEN** user sends POST /api/v1/media/image with prompt
- **THEN** route to appropriate provider (DALL-E, etc.)
- **AND** return task_id for async tracking via shared/task

#### Scenario: Generate video

- **WHEN** user sends POST /api/v1/media/video with prompt
- **THEN** route to video provider (Runway, etc.)
- **AND** return task_id for async tracking via shared/task

#### Scenario: Image-to-video generation

- **WHEN** user provides image_url with video generation request
- **THEN** use image as first frame for video generation

#### Scenario: Get task status

- **WHEN** user sends GET /api/v1/media/tasks/{id}
- **THEN** query shared/task manager for status
- **AND** return current status, progress, and output if completed

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

The system SHALL provide adapter for OpenAI DALL-E image generation in media module.

#### Scenario: DALL-E text-to-image

- **WHEN** adapter receives image request
- **THEN** call OpenAI images/generations API
- **AND** return generated image URLs

#### Scenario: DALL-E parameters

- **WHEN** request includes size, quality, n parameters
- **THEN** pass to DALL-E API (1024x1024, hd, etc.)

### Requirement: Video Generation Adapter

The system SHALL provide adapter for video generation services in media module.

#### Scenario: Video generation request

- **WHEN** adapter receives video request with prompt
- **THEN** initiate async video generation
- **AND** return external task ID for status polling

#### Scenario: Video generation status

- **WHEN** polling video task status
- **THEN** return progress percentage
- **AND** return video URL when complete

### Requirement: Media Output Storage

The system SHALL handle media output storage and URLs.

#### Scenario: External URL response

- **WHEN** provider returns external URL
- **THEN** return URL directly to client

#### Scenario: URL expiration handling

- **WHEN** provider URL has expiration
- **THEN** include expiration info in response
- **AND** optionally download and store to R2

