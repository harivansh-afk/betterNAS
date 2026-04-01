package nodeagent

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/webdav"
)

type exportFileSystem struct {
	root *os.Root
}

var _ webdav.FileSystem = (*exportFileSystem)(nil)

func newExportFileSystem(rootPath string) (*exportFileSystem, error) {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, fmt.Errorf("open export root %s: %w", rootPath, err)
	}

	return &exportFileSystem{
		root: root,
	}, nil
}

func (f *exportFileSystem) Close() error {
	if f.root == nil {
		return nil
	}

	err := f.root.Close()
	f.root = nil
	return err
}

func (f *exportFileSystem) Mkdir(_ context.Context, name string, perm os.FileMode) error {
	resolvedName, err := resolveExportName(name)
	if err != nil {
		return pathError("mkdir", name, err)
	}

	if resolvedName == "." {
		return pathError("mkdir", name, os.ErrInvalid)
	}

	return f.root.Mkdir(resolvedName, perm)
}

func (f *exportFileSystem) OpenFile(_ context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	resolvedName, err := resolveExportName(name)
	if err != nil {
		return nil, pathError("open", name, err)
	}

	file, err := f.root.OpenFile(resolvedName, flag, perm)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (f *exportFileSystem) RemoveAll(_ context.Context, name string) error {
	resolvedName, err := resolveExportName(name)
	if err != nil {
		return pathError("removeall", name, err)
	}

	if resolvedName == "." {
		return pathError("removeall", name, os.ErrInvalid)
	}

	return f.root.RemoveAll(resolvedName)
}

func (f *exportFileSystem) Rename(_ context.Context, oldName, newName string) error {
	resolvedOldName, err := resolveExportName(oldName)
	if err != nil {
		return pathError("rename", oldName, err)
	}

	resolvedNewName, err := resolveExportName(newName)
	if err != nil {
		return pathError("rename", newName, err)
	}

	if resolvedOldName == "." || resolvedNewName == "." {
		return pathError("rename", oldName, os.ErrInvalid)
	}

	return f.root.Rename(resolvedOldName, resolvedNewName)
}

func (f *exportFileSystem) Stat(_ context.Context, name string) (os.FileInfo, error) {
	resolvedName, err := resolveExportName(name)
	if err != nil {
		return nil, pathError("stat", name, err)
	}

	return f.root.Stat(resolvedName)
}

func resolveExportName(name string) (string, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return "", os.ErrNotExist
	}

	if strings.Contains(name, "\x00") {
		return "", os.ErrNotExist
	}

	cleanedName := path.Clean("/" + name)
	cleanedName = strings.TrimPrefix(cleanedName, "/")
	if cleanedName == "" {
		return ".", nil
	}

	return filepath.FromSlash(cleanedName), nil
}

func pathError(op, path string, err error) error {
	return &os.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}
