## GitFS

This lets you present a Git repository as a `http.FileSystem` to let
the http package take care of the HTTP for you.

# Usage

There is only one function for now. Call

```go
gitfs.NewFromReference()
```

and give it your repository and which reference you want it to
expose. It will load the tree and present that as a filesystem.
