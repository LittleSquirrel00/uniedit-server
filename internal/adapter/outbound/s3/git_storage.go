package s3

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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-git/go-billy/v5"

	"github.com/uniedit/server/internal/port/outbound"
)

// ErrObjectNotFound indicates the object was not found.
var ErrObjectNotFound = errors.New("object not found")

// GitStorageAdapter implements GitStoragePort using R2/S3.
type GitStorageAdapter struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

// NewGitStorageAdapter creates a new Git storage adapter.
func NewGitStorageAdapter(client *s3.Client, bucket string) *GitStorageAdapter {
	return &GitStorageAdapter{
		client:    client,
		presigner: s3.NewPresignClient(client),
		bucket:    bucket,
	}
}

// GetFilesystem returns a billy.Filesystem for a repository.
func (a *GitStorageAdapter) GetFilesystem(ctx context.Context, storagePath string) (billy.Filesystem, error) {
	return NewR2Filesystem(a.client, a.bucket, storagePath), nil
}

// DeleteRepository deletes all storage for a repository.
func (a *GitStorageAdapter) DeleteRepository(ctx context.Context, storagePath string) error {
	// List all objects with prefix
	paginator := s3.NewListObjectsV2Paginator(a.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(a.bucket),
		Prefix: aws.String(storagePath),
	})

	var objectsToDelete []types.ObjectIdentifier
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			objectsToDelete = append(objectsToDelete, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}
	}

	// Delete in batches of 1000
	for i := 0; i < len(objectsToDelete); i += 1000 {
		end := i + 1000
		if end > len(objectsToDelete) {
			end = len(objectsToDelete)
		}

		_, err := a.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(a.bucket),
			Delete: &types.Delete{
				Objects: objectsToDelete[i:end],
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("delete objects: %w", err)
		}
	}

	return nil
}

// GetRepositorySize calculates the size of a repository's storage.
func (a *GitStorageAdapter) GetRepositorySize(ctx context.Context, storagePath string) (int64, error) {
	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(a.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(a.bucket),
		Prefix: aws.String(storagePath),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Size != nil {
				totalSize += *obj.Size
			}
		}
	}

	return totalSize, nil
}

// Compile-time check
var _ outbound.GitStoragePort = (*GitStorageAdapter)(nil)

// ===== R2Filesystem implementation =====

// R2Filesystem implements billy.Filesystem for R2/S3 storage.
type R2Filesystem struct {
	client *s3.Client
	bucket string
	prefix string

	mu    sync.RWMutex
	cache map[string]*cachedFile
}

type cachedFile struct {
	data    *bytes.Buffer
	modTime time.Time
}

// NewR2Filesystem creates a new R2-backed filesystem.
func NewR2Filesystem(client *s3.Client, bucket, prefix string) *R2Filesystem {
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return &R2Filesystem{
		client: client,
		bucket: bucket,
		prefix: prefix,
		cache:  make(map[string]*cachedFile),
	}
}

// --- billy.Basic interface ---

func (fs *R2Filesystem) Create(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

func (fs *R2Filesystem) Open(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

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

	if flag&os.O_RDONLY != 0 || (flag&os.O_RDWR != 0 && flag&os.O_TRUNC == 0) {
		fs.mu.RLock()
		cached, ok := fs.cache[filename]
		fs.mu.RUnlock()

		if ok {
			f.content = bytes.NewBuffer(cached.data.Bytes())
		} else {
			ctx := context.Background()
			result, err := fs.client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(fs.bucket),
				Key:    aws.String(key),
			})
			if err != nil {
				var nsk *types.NoSuchKey
				if errors.As(err, &nsk) {
					if flag&os.O_CREATE == 0 {
						return nil, os.ErrNotExist
					}
					f.content = bytes.NewBuffer(nil)
				} else {
					return nil, fmt.Errorf("open file: %w", err)
				}
			} else {
				defer result.Body.Close()
				data, err := io.ReadAll(result.Body)
				if err != nil {
					return nil, fmt.Errorf("read file: %w", err)
				}
				f.content = bytes.NewBuffer(data)
			}
		}
	} else {
		f.content = bytes.NewBuffer(nil)
	}

	if flag&os.O_APPEND != 0 {
		f.position = int64(f.content.Len())
	}

	return f, nil
}

func (fs *R2Filesystem) Stat(filename string) (os.FileInfo, error) {
	filename = cleanPath(filename)
	key := fs.prefix + filename

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

	ctx := context.Background()

	// Check if it's a directory
	dirPrefix := key
	if !strings.HasSuffix(dirPrefix, "/") {
		dirPrefix += "/"
	}

	result, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(fs.bucket),
		Prefix:  aws.String(dirPrefix),
		MaxKeys: aws.Int32(1),
	})
	if err == nil && len(result.Contents) > 0 {
		return &r2FileInfo{
			name:    path.Base(filename),
			size:    0,
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	// Check if it's a file
	headResult, err := fs.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		var nf *types.NotFound
		if errors.As(err, &nsk) || errors.As(err, &nf) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	size := int64(0)
	if headResult.ContentLength != nil {
		size = *headResult.ContentLength
	}

	modTime := time.Now()
	if headResult.LastModified != nil {
		modTime = *headResult.LastModified
	}

	return &r2FileInfo{
		name:    path.Base(filename),
		size:    size,
		mode:    0644,
		modTime: modTime,
		isDir:   false,
	}, nil
}

func (fs *R2Filesystem) Rename(oldpath, newpath string) error {
	oldpath = cleanPath(oldpath)
	newpath = cleanPath(newpath)

	oldKey := fs.prefix + oldpath
	newKey := fs.prefix + newpath

	ctx := context.Background()

	// Copy to new location
	_, err := fs.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(fs.bucket),
		CopySource: aws.String(fs.bucket + "/" + oldKey),
		Key:        aws.String(newKey),
	})
	if err != nil {
		return fmt.Errorf("rename copy: %w", err)
	}

	// Delete old object
	_, err = fs.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(oldKey),
	})
	if err != nil {
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

func (fs *R2Filesystem) Remove(filename string) error {
	filename = cleanPath(filename)
	key := fs.prefix + filename

	ctx := context.Background()
	_, err := fs.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}

	fs.mu.Lock()
	delete(fs.cache, filename)
	fs.mu.Unlock()

	return nil
}

func (fs *R2Filesystem) Join(elem ...string) string {
	return path.Join(elem...)
}

// --- billy.Dir interface ---

func (fs *R2Filesystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	dirname = cleanPath(dirname)
	prefix := fs.prefix + dirname
	if !strings.HasSuffix(prefix, "/") && prefix != "" {
		prefix += "/"
	}

	ctx := context.Background()
	result, err := fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(fs.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(1000),
	})
	if err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}

	entries := make(map[string]os.FileInfo)

	for _, obj := range result.Contents {
		if obj.Key == nil {
			continue
		}
		relPath := strings.TrimPrefix(*obj.Key, prefix)
		if relPath == "" {
			continue
		}

		parts := strings.SplitN(relPath, "/", 2)
		name := parts[0]

		if _, exists := entries[name]; exists {
			continue
		}

		if len(parts) > 1 {
			entries[name] = &r2FileInfo{
				name:    name,
				size:    0,
				mode:    os.ModeDir | 0755,
				modTime: time.Now(),
				isDir:   true,
			}
		} else {
			size := int64(0)
			if obj.Size != nil {
				size = *obj.Size
			}
			modTime := time.Now()
			if obj.LastModified != nil {
				modTime = *obj.LastModified
			}
			entries[name] = &r2FileInfo{
				name:    name,
				size:    size,
				mode:    0644,
				modTime: modTime,
				isDir:   false,
			}
		}
	}

	infos := make([]os.FileInfo, 0, len(entries))
	for _, info := range entries {
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name() < infos[j].Name()
	})

	return infos, nil
}

func (fs *R2Filesystem) MkdirAll(dirname string, perm os.FileMode) error {
	// R2 doesn't need explicit directory creation
	return nil
}

// --- billy.TempFile interface ---

func (fs *R2Filesystem) TempFile(dir, prefix string) (billy.File, error) {
	name := fmt.Sprintf("%s/%s%d", dir, prefix, time.Now().UnixNano())
	return fs.Create(name)
}

// --- billy.Chroot interface ---

func (fs *R2Filesystem) Root() string {
	return "/"
}

func (fs *R2Filesystem) Chroot(p string) (billy.Filesystem, error) {
	p = cleanPath(p)
	newPrefix := fs.prefix + p
	if !strings.HasSuffix(newPrefix, "/") {
		newPrefix += "/"
	}
	return &R2Filesystem{
		client: fs.client,
		bucket: fs.bucket,
		prefix: newPrefix,
		cache:  make(map[string]*cachedFile),
	}, nil
}

// --- billy.Symlink interface ---

func (fs *R2Filesystem) Lstat(filename string) (os.FileInfo, error) {
	return fs.Stat(filename)
}

func (fs *R2Filesystem) Symlink(target, link string) error {
	return errors.New("symlinks not supported in R2 filesystem")
}

func (fs *R2Filesystem) Readlink(link string) (string, error) {
	return "", errors.New("symlinks not supported in R2 filesystem")
}

// --- billy.Capable interface ---

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

func (f *r2File) Name() string { return f.name }

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

	if f.position > int64(len(data)) {
		padding := make([]byte, f.position-int64(len(data)))
		data = append(data, padding...)
	}

	if f.position < int64(len(data)) {
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

	if f.flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND) != 0 {
		ctx := context.Background()
		data := f.content.Bytes()

		f.fs.mu.Lock()
		f.fs.cache[f.name] = &cachedFile{
			data:    bytes.NewBuffer(data),
			modTime: time.Now(),
		}
		f.fs.mu.Unlock()

		_, err := f.fs.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:        aws.String(f.fs.bucket),
			Key:           aws.String(f.key),
			Body:          bytes.NewReader(data),
			ContentLength: aws.Int64(int64(len(data))),
		})
		if err != nil {
			return fmt.Errorf("close write: %w", err)
		}
	}

	return nil
}

func (f *r2File) Lock() error   { return nil }
func (f *r2File) Unlock() error { return nil }

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

// Interface checks
var (
	_ billy.Filesystem = (*R2Filesystem)(nil)
	_ billy.Dir        = (*R2Filesystem)(nil)
	_ billy.Chroot     = (*R2Filesystem)(nil)
	_ billy.TempFile   = (*R2Filesystem)(nil)
)
