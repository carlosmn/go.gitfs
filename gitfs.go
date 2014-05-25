// Package gitfs implements a http.FileSystem which is backed by a Git
// repository.
//
// More specifically, the object returned is tied to a particular
// tree, which it will present to the http library as an approximation
// to a real filesystem.
package gitfs

import (
	"errors"
	"github.com/libgit2/git2go"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type gitFile struct {
	entry  *git.TreeEntry
	blob   *git.Blob
	offset int64
}

type gitFileInfo struct {
	entry *git.TreeEntry
	obj   git.Object
}

func (v *gitFile) Close() error {
	v.blob.Free()
	v.blob = nil

	return nil
}

func (v *gitFile) Read(out []byte) (int, error) {
	data := v.blob.Contents()

	n := copy(out, data[v.offset:])
	if n == 0 {
		return 0, io.EOF
	}

	v.offset += int64(n)

	return n, nil
}

func (v *gitFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("sorry, I'm not a directory")
}

func (v *gitFile) Seek(offset int64, whence int) (int64, error) {
	// whence values as raw ints, programming like it's 1990
	switch whence {
	case 0:
		v.offset = offset
	case 1:
		v.offset += offset
	case 2:
		v.offset = v.blob.Size() + offset
	default:
		return 0, errors.New("invalid whence")
	}

	return v.offset, nil
}

func (v *gitFile) Stat() (os.FileInfo, error) {
	return &gitFileInfo{
		entry: v.entry,
		obj:   v.blob,
	}, nil
}

func (v *gitFileInfo) Name() string {
	return v.entry.Name
}

func (v *gitFileInfo) Size() int64 {
	// the real size for a "file", otherwise whatever
	if blob, ok := v.obj.(*git.Blob); ok {
		return blob.Size()
	}

	return 0
}

func (v *gitFileInfo) Mode() os.FileMode {
	var mode os.FileMode

	switch v.entry.Filemode {
	case git.FilemodeBlob:
		mode = 0
	case git.FilemodeTree:
		mode = os.ModeDir
	}

	return mode
}

func (v *gitFileInfo) ModTime() time.Time {
	return time.Now()
}

func (v *gitFileInfo) IsDir() bool {
	return v.Mode().IsDir()
}

func (v *gitFileInfo) Sys() interface{} {
	return nil
}

type gitTree struct {
	entry *git.TreeEntry
	tree  *git.Tree
	idx   uint64
}

func (v *gitTree) Close() error {
	v.tree.Free()
	v.tree = nil

	return nil
}

func (v *gitTree) Read(out []byte) (int, error) {
	return 0, io.EOF
}

func (v *gitTree) Readdir(count int) ([]os.FileInfo, error) {
	max := v.tree.EntryCount()
	list := make([]os.FileInfo, 0, max)
	for ; v.idx < max; v.idx++ {
		entry := v.tree.EntryByIndex(v.idx)
		obj, err := v.tree.Owner().Lookup(entry.Id)
		if err != nil {
			return nil, err
		}

		list = append(list, &gitFileInfo{
			entry: entry,
			obj:   obj,
		})
	}

	return list, nil
}

func (v *gitTree) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("what you wanna seek")
}

func (v *gitTree) Stat() (os.FileInfo, error) {
	return &gitFileInfo{
		entry: v.entry,
		obj:   v.tree,
	}, nil
}

type gitFileSystem struct {
	tree *git.Tree
}

func (v *gitFileSystem) Open(name string) (http.File, error) {
	var err error
	var entry *git.TreeEntry

	if name == "/" {
		entry = &git.TreeEntry{
			Name:     "",
			Type:     git.ObjectTree,
			Filemode: git.FilemodeTree,
			Id:       v.tree.Id(),
		}
	} else {
		// for some reason we're asked for //index.html
		for strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		if entry, err = v.tree.EntryByPath(name); err != nil {
			return nil, err
		}
	}

	var obj git.Object
	if obj, err = v.tree.Owner().Lookup(entry.Id); err != nil {
		return nil, err
	}

	if entry.Type == git.ObjectTree {
		return &gitTree{
			entry: entry,
			tree:  obj.(*git.Tree),
		}, nil
	}

	return &gitFile{
		entry: entry,
		blob:  obj.(*git.Blob),
	}, nil
}

// NewFromTree creates a new http.FileSystem from a given tree. The
// tree will be exposed as the root of the filesystem.
func NewFromTree(tree *git.Tree) http.FileSystem {
	return &gitFileSystem{
		tree: tree,
	}
}

// NewFromReference creates a new http.FileSystem from a given
// reference. The reference must peel to a tree, which will then be
// exposed as the root of the filesystem
func NewFromReference(ref *git.Reference) (http.FileSystem, error) {
	var obj git.Object
	var err error
	if obj, err = ref.Peel(git.ObjectTree); err != nil {
		return nil, err
	}

	return NewFromTree(obj.(*git.Tree)), nil
}

// NewFromReferenceName creates a new FileSystem from a reference
// (specified by name) in the given repository. The reference must
// peel to a tree, which will then be exposed as the root of the
// filesystem.
func NewFromReferenceName(repo *git.Repository, branch string) (http.FileSystem, error) {
	var ref *git.Reference
	var err error

	if ref, err = repo.LookupReference(branch); err != nil {
		return nil, err
	}

	return NewFromReference(ref)
}
