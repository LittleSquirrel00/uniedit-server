package git

import (
	"time"

	"github.com/google/uuid"
)

// LFSObject represents an LFS object (content-addressable).
type LFSObject struct {
	oid         string // SHA-256
	size        int64
	storageKey  string
	contentType string
	createdAt   time.Time
}

// NewLFSObject creates a new LFS object.
func NewLFSObject(oid string, size int64, storageKey, contentType string) *LFSObject {
	return &LFSObject{
		oid:         oid,
		size:        size,
		storageKey:  storageKey,
		contentType: contentType,
		createdAt:   time.Now(),
	}
}

// ReconstructLFSObject reconstructs an LFS object from persistence.
func ReconstructLFSObject(oid string, size int64, storageKey, contentType string, createdAt time.Time) *LFSObject {
	return &LFSObject{
		oid:         oid,
		size:        size,
		storageKey:  storageKey,
		contentType: contentType,
		createdAt:   createdAt,
	}
}

// Getters
func (o *LFSObject) OID() string          { return o.oid }
func (o *LFSObject) Size() int64          { return o.size }
func (o *LFSObject) StorageKey() string   { return o.storageKey }
func (o *LFSObject) ContentType() string  { return o.contentType }
func (o *LFSObject) CreatedAt() time.Time { return o.createdAt }

// LFSRepoObject represents the association between a repo and LFS object.
type LFSRepoObject struct {
	repoID    uuid.UUID
	oid       string
	createdAt time.Time
}

// NewLFSRepoObject creates a new repo-object association.
func NewLFSRepoObject(repoID uuid.UUID, oid string) *LFSRepoObject {
	return &LFSRepoObject{
		repoID:    repoID,
		oid:       oid,
		createdAt: time.Now(),
	}
}

// ReconstructLFSRepoObject reconstructs a repo-object association.
func ReconstructLFSRepoObject(repoID uuid.UUID, oid string, createdAt time.Time) *LFSRepoObject {
	return &LFSRepoObject{
		repoID:    repoID,
		oid:       oid,
		createdAt: createdAt,
	}
}

// Getters
func (o *LFSRepoObject) RepoID() uuid.UUID   { return o.repoID }
func (o *LFSRepoObject) OID() string         { return o.oid }
func (o *LFSRepoObject) CreatedAt() time.Time { return o.createdAt }

// LFSLock represents a file lock.
type LFSLock struct {
	id       uuid.UUID
	repoID   uuid.UUID
	path     string
	ownerID  uuid.UUID
	lockedAt time.Time
}

// NewLFSLock creates a new LFS lock.
func NewLFSLock(repoID uuid.UUID, path string, ownerID uuid.UUID) *LFSLock {
	return &LFSLock{
		id:       uuid.New(),
		repoID:   repoID,
		path:     path,
		ownerID:  ownerID,
		lockedAt: time.Now(),
	}
}

// ReconstructLFSLock reconstructs an LFS lock from persistence.
func ReconstructLFSLock(id, repoID uuid.UUID, path string, ownerID uuid.UUID, lockedAt time.Time) *LFSLock {
	return &LFSLock{
		id:       id,
		repoID:   repoID,
		path:     path,
		ownerID:  ownerID,
		lockedAt: lockedAt,
	}
}

// Getters
func (l *LFSLock) ID() uuid.UUID        { return l.id }
func (l *LFSLock) RepoID() uuid.UUID    { return l.repoID }
func (l *LFSLock) Path() string         { return l.path }
func (l *LFSLock) OwnerID() uuid.UUID   { return l.ownerID }
func (l *LFSLock) LockedAt() time.Time  { return l.lockedAt }

// IsOwnedBy returns true if the lock is owned by the given user.
func (l *LFSLock) IsOwnedBy(userID uuid.UUID) bool {
	return l.ownerID == userID
}
