package vfs

import (
	"fmt"
	"os"
	"path"
	"sync"
)

var (
	DefaultFileSystem = New()
)

// FileSystemCache is a virtual passthrough filesystem that is able to write
// files to the actual filesystem and record exactly what it wrote so that
// it can be read back later without needing to read off of disk.
type FileSystemCache struct {
	folder string
	files  map[string][]byte
	mutex  sync.RWMutex
}

// New returns a FileSystemCache instance.
func New() *FileSystemCache {
	return &FileSystemCache{
		files: make(map[string][]byte),
	}
}

// SetBaseFolder sets the base folder used for persisting files in the underlying system.
func (f *FileSystemCache) SetBaseFolder(name string) *FileSystemCache {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.folder = name
	return f
}

// PathFor gives the underlying system path for the given file.
func (f *FileSystemCache) PathFor(name string) string {
	return path.Join(f.folder, name)
}

// WriteFile writes a file to the underlying filesystem and caches the write.
func (f *FileSystemCache) WriteFile(name string, data []byte, perm os.FileMode) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	directory := path.Dir(name)
	if directory != "." {
		if err := os.MkdirAll(path.Join(f.folder, directory), 0700); err != nil {
			return err
		}
	}

	if err := os.WriteFile(path.Join(f.folder, name), data, perm); err != nil {
		return err
	}
	f.files[name] = data
	return nil
}

// ReadFile reads a file from the cache.
func (f *FileSystemCache) ReadFile(name string) ([]byte, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	data, ok := f.files[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", os.ErrNotExist, name)
	}
	return data, nil
}

// All returns all files stored in the filesystem.
func (f *FileSystemCache) All() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	files := make([]string, 0, len(f.files))
	for file := range f.files {
		files = append(files, file)
	}
	return files
}

// WriteFile writes a file to the underlying filesystem using the default filesystem and caches the write.
func WriteFile(name string, data []byte, perm os.FileMode) error {
	return DefaultFileSystem.WriteFile(name, data, perm)
}

// ReadFile reads a file from the cache of the default filesystem.
func ReadFile(name string) ([]byte, error) {
	return DefaultFileSystem.ReadFile(name)
}

// All returns all files stored in the default filesystem.
func All(name string) []string {
	return DefaultFileSystem.All()
}

// SetBaseFolder sets the base folder used for persisting files in the default file system.
func SetBaseFolder(name string) {
	DefaultFileSystem.SetBaseFolder(name)
}

// PathFor gives the underlying system path for the given file.
func PathFor(name string) string {
	return DefaultFileSystem.PathFor(name)
}
