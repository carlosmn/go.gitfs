## GitFS

This lets you present a Git repository as a `http.FileSystem` to let
the http package take care of the HTTP for you.

# Dependencies

This uses git2go which uses libgit2. This means you need a compatible
version of libgit2 installed and accessible on your system. At some
point libgit2 will have a static version, which will make this
simpler.

# Usage

[![GoDoc](https://godoc.org/github.com/carlosmn/go.gitfs?status.png)](https://godoc.org/github.com/carlosmn/go.gitfs)

The constructors let you do more or less work depending on how much
information you have. You can pass in a tree, a reference or simply
the name.

The tree will be used as the root of the filesystem.