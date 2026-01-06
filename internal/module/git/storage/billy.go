package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
)

// R2Filesystem implements billy.Filesystem for R2 storage.
// It provides a file system interface over R2 objects.
type R2Filesystem struct {
	client *R2Client
	prefix string // Base prefix in R2 bucket (e.g., "repos/owner-id/repo-id/")

	// In-memory cache for directory structure
	mu    sync.RWMutex
	cache map[string]*cachedFile
}

// cachedFile represents a file being written.
type cachedFile struct {
	data    *bytes.Buffer
	modTime time.Time
}

// NewR2Filesystem creates a new R2-backed filesystem.
func NewR2Filesystem(client *R2Client, prefix string) *R2Filesystem {
	// Ensure prefix ends with /
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return &R2Filesystem{
		client: client,
		prefix: prefix,
		cache:  make(map[string]*cachedFile),
	}
}

// --- billy.Basic interface ---

// Create creates a new file.
func (fs *R2Filesystem) Create(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

// Open opens a file for reading.
func (fs *R2Filesystem) Open(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

// OpenFile opens a file with specified flags and mode.
func (fs *R2Filesystem) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	filename = cleanPath(filename)
	key := fs.prefix + filename

	f := &r2File{
		fs:       fs,
		name:     filename,
		key:      key,
		flag:     flag,
		perm:     perm,
		position: 0,
	}

	// If opening for read or read-write without truncate, load existing content
	if flag&os.O_RDONLY != 0 || (flag&os.O_RDWR != 0 && flag&os.O_TRUNC == 0) {
		// Check cache first
		fs.mu.RLock()
		cached, ok := fs.cache[filename]
		fs.mu.RUnlock()

		if ok {
			f.content = bytes.NewBuffer(cached.data.Bytes())
		} else {
			// Try to load from R2
			ctx := context.Background()
			reader, _, err := fs.client.GetObject(ctx, key)
			if err != nil {
				if errors.Is(err, ErrObjectNotFound) {
					// File doesn't exist
					if flag&os.O_CREATE == 0 {
						return nil, os.ErrNotExist
					}
					f.content = bytes.NewBuffer(nil)
				} else {
					return nil, fmt.Errorf("open file: %w", err)
				}
			} else {
				defer reader.Close()
				data, err := io.ReadAll(reader)
				if err != nil {
					return nil, fmt.Errorf("read file: %w", err)
				}
				f.content = bytes.NewBuffer(data)
			}
		}
	} else {
		f.content = bytes.NewBuffer(nil)
	}

	// Handle O_APPEND
	if flag&os.O_APPEND != 0 {
		f.position = int64(f.content.Len())
	}

	return f, nil
}

// Stat returns file info.
func (fs *R2Filesystem) Stat(filename string) (os.FileInfo, error) {
	filename = cleanPath(filename)
	key := fs.prefix + filename

	// Check cache first
	fs.mu.RLock()
	cached, ok := fs.cache[filename]
	fs.mu.RUnlock()

	if ok {
		return &r2FileInfo{
			name:    path.Base(filename),
			size:    int64(cached.data.Len()),
			mode:    0644,
			modTime: cached.modTime,
			isDir:   false,
		}, nil
	}

	// Check if it's a directory (has objects with this prefix)
	ctx := context.Background()
	dirPrefix := key
	if !strings.HasSuffix(dirPrefix, "/") {
		dirPrefix += "/"
	}

	objects, err := fs.client.ListObjects(ctx, dirPrefix, 1)
	if err == nil && len(objects) > 0 {
		return &r2FileInfo{
			name:    path.Base(filename),
			size:    0,
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	// Check if it's a file
	info, err := fs.client.HeadObject(ctx, key)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	modTime := time.Now()
	if info.LastModified != nil {
		modTime = *info.LastModified
	}

	return &r2FileInfo{
		name:    path.Base(filename),
		size:    info.Size,
		mode:    0644,
		modTime: modTime,
		isDir:   false,
	}, nil
}

// Rename renames a file.
func (fs *R2Filesystem) Rename(oldpath, newpath string) error {
	oldpath = cleanPath(oldpath)
	newpath = cleanPath(newpath)

	oldKey := fs.prefix + oldpath
	newKey := fs.prefix + newpath

	ctx := context.Background()

	// Copy to new location
	if err := fs.client.CopyObject(ctx, oldKey, newKey); err != nil {
		return fmt.Errorf("rename copy: %w", err)
	}

	// Delete old object
	if err := fs.client.DeleteObject(ctx, oldKey); err != nil {
		return fmt.Errorf("rename delete: %w", err)
	}

	// Update cache
	fs.mu.Lock()
	if cached, ok := fs.cache[oldpath]; ok {
		fs.cache[newpath] = cached
		delete(fs.cache, oldpath)
	}
	fs.mu.Unlock()

	return nil
}

// Remove removes a file.
func (fs *R2Filesystem) Remove(filename string) error {
	filename = cleanPath(filename)
	key := fs.prefix + filename

	ctx := context.Background()
	if err := fs.client.DeleteObject(ctx, key); err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	// Remove from cache
	fs.mu.Lock()
	delete(fs.cache, filename)
	fs.mu.Unlock()

	return nil
}

// Join joins path elements.
func (fs *R2Filesystem) Join(elem ...string) string {
	return path.Join(elem...)
}

// --- billy.Dir interface ---

// ReadDir reads a directory.
func (fs *R2Filesystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	dirname = cleanPath(dirname)
	prefix := fs.prefix + dirname
	if !strings.HasSuffix(prefix, "/") && prefix != "" {
		prefix += "/"
	}

	ctx := context.Background()
	objects, err := fs.client.ListObjects(ctx, prefix, 1000)
	if err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}

	// Track unique entries (files and directories)
	entries := make(map[string]os.FileInfo)

	for _, obj := range objects {
		// Remove prefix to get relative path
		relPath := strings.TrimPrefix(obj.Key, prefix)
		if relPath == "" {
			continue
		}

		// Split by / to get first component
		parts := strings.SplitN(relPath, "/", 2)
		name := parts[0]

		if _, exists := entries[name]; exists {
			continue
		}

		if len(parts) > 1 {
			// It's a directory
			entries[name] = &r2FileInfo{
				name:    name,
				size:    0,
				mode:    os.ModeDir | 0755,
				modTime: time.Now(),
				isDir:   true,
			}
		} else {
			// It's a file
			modTime := time.Now()
			if obj.LastModified != nil {
				modTime = *obj.LastModified
			}
			entries[name] = &r2FileInfo{
				name:    name,
				size:    obj.Size,
				mode:    0644,
				modTime: modTime,
				isDir:   false,
			}
		}
	}

	// Convert to sorted slice
	result := make([]os.FileInfo, 0, len(entries))
	for _, info := range entries {
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})

	return result, nil
}

// MkdirAll creates directories.
// In R2, directories don't really exist - they're implied by object keys.
// We create a placeholder object to represent the directory.
func (fs *R2Filesystem) MkdirAll(dirname string, perm os.FileMode) error {
	// R2 doesn't need explicit directory creation
	// Directories are implied by object keys
	return nil
}

// --- billy.TempFile interface ---

// TempFile creates a temporary file.
func (fs *R2Filesystem) TempFile(dir, prefix string) (billy.File, error) {
	name := fmt.Sprintf("%s/%s%d", dir, prefix, time.Now().UnixNano())
	return fs.Create(name)
}

// --- billy.Chroot interface ---

// Root returns the root path.
func (fs *R2Filesystem) Root() string {
	return "/"
}

// Chroot returns a new filesystem rooted at path.
func (fs *R2Filesystem) Chroot(path string) (billy.Filesystem, error) {
	path = cleanPath(path)
	newPrefix := fs.prefix + path
	if !strings.HasSuffix(newPrefix, "/") {
		newPrefix += "/"
	}
	return &R2Filesystem{
		client: fs.client,
		prefix: newPrefix,
		cache:  make(map[string]*cachedFile),
	}, nil
}

// --- billy.Symlink interface ---

// Lstat is the same as Stat for R2 (no symlinks).
func (fs *R2Filesystem) Lstat(filename string) (os.FileInfo, error) {
	return fs.Stat(filename)
}

// Symlink creates a symlink (not supported in R2).
func (fs *R2Filesystem) Symlink(target, link string) error {
	return errors.New("symlinks not supported in R2 filesystem")
}

// Readlink reads a symlink (not supported in R2).
func (fs *R2Filesystem) Readlink(link string) (string, error) {
	return "", errors.New("symlinks not supported in R2 filesystem")
}

// --- billy.Capable interface ---

// Capabilities returns the filesystem capabilities.
func (fs *R2Filesystem) Capabilities() billy.Capability {
	return billy.ReadCapability |
		billy.WriteCapability |
		billy.ReadAndWriteCapability |
		billy.SeekCapability |
		billy.TruncateCapability
}

// --- Helper functions ---

func cleanPath(p string) string {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")
	return p
}

// --- r2File implements billy.File ---

type r2File struct {
	fs       *R2Filesystem
	name     string
	key      string
	flag     int
	perm     os.FileMode
	content  *bytes.Buffer
	position int64
	closed   bool
}

func (f *r2File) Name() string {
	return f.name
}

func (f *r2File) Read(p []byte) (int, error) {
	if f.closed {
		return 0, os.ErrClosed
	}

	if f.position >= int64(f.content.Len()) {
		return 0, io.EOF
	}

	data := f.content.Bytes()
	n := copy(p, data[f.position:])
	f.position += int64(n)

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func (f *r2File) ReadAt(p []byte, off int64) (int, error) {
	if f.closed {
		return 0, os.ErrClosed
	}

	if off >= int64(f.content.Len()) {
		return 0, io.EOF
	}

	data := f.content.Bytes()
	n := copy(p, data[off:])

	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}

func (f *r2File) Write(p []byte) (int, error) {
	if f.closed {
		return 0, os.ErrClosed
	}

	if f.flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) == 0 {
		return 0, errors.New("file not opened for writing")
	}

	data := f.content.Bytes()

	// Extend if needed
	if f.position > int64(len(data)) {
		padding := make([]byte, f.position-int64(len(data)))
		data = append(data, padding...)
	}

	// Write at position
	if f.position < int64(len(data)) {
		// Overwrite existing content
		end := f.position + int64(len(p))
		if end > int64(len(data)) {
			data = append(data[:f.position], p...)
		} else {
			copy(data[f.position:], p)
		}
	} else {
		data = append(data, p...)
	}

	f.content = bytes.NewBuffer(data)
	f.position += int64(len(p))

	return len(p), nil
}

func (f *r2File) Seek(offset int64, whence int) (int64, error) {
	if f.closed {
		return 0, os.ErrClosed
	}

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = f.position + offset
	case io.SeekEnd:
		newPos = int64(f.content.Len()) + offset
	default:
		return 0, errors.New("invalid whence")
	}

	if newPos < 0 {
		return 0, errors.New("negative position")
	}

	f.position = newPos
	return f.position, nil
}

func (f *r2File) Close() error {
	if f.closed {
		return os.ErrClosed
	}
	f.closed = true

	// If file was opened for writing, save to R2
	if f.flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND) != 0 {
		ctx := context.Background()
		data := f.content.Bytes()

		// Save to cache
		f.fs.mu.Lock()
		f.fs.cache[f.name] = &cachedFile{
			data:    bytes.NewBuffer(data),
			modTime: time.Now(),
		}
		f.fs.mu.Unlock()

		// Upload to R2
		err := f.fs.client.PutObject(ctx, f.key, bytes.NewReader(data), int64(len(data)), "")
		if err != nil {
			return fmt.Errorf("close write: %w", err)
		}
	}

	return nil
}

func (f *r2File) Lock() error {
	// Not implemented for R2
	return nil
}

func (f *r2File) Unlock() error {
	// Not implemented for R2
	return nil
}

func (f *r2File) Truncate(size int64) error {
	if f.closed {
		return os.ErrClosed
	}

	data := f.content.Bytes()
	if size < int64(len(data)) {
		f.content = bytes.NewBuffer(data[:size])
	} else if size > int64(len(data)) {
		padding := make([]byte, size-int64(len(data)))
		f.content = bytes.NewBuffer(append(data, padding...))
	}

	return nil
}

// --- r2FileInfo implements os.FileInfo ---

type r2FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (fi *r2FileInfo) Name() string       { return fi.name }
func (fi *r2FileInfo) Size() int64        { return fi.size }
func (fi *r2FileInfo) Mode() os.FileMode  { return fi.mode }
func (fi *r2FileInfo) ModTime() time.Time { return fi.modTime }
func (fi *r2FileInfo) IsDir() bool        { return fi.isDir }
func (fi *r2FileInfo) Sys() interface{}   { return nil }

// Ensure R2Filesystem implements all required interfaces
var (
	_ billy.Filesystem = (*R2Filesystem)(nil)
	_ billy.Dir        = (*R2Filesystem)(nil)
	_ billy.Chroot     = (*R2Filesystem)(nil)
	_ billy.TempFile   = (*R2Filesystem)(nil)
)
