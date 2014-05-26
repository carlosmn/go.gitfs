package gitfs

import (
	"github.com/libgit2/git2go"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"
)

func checkFatal(t *testing.T, err error) {
	if err == nil {
		return
	}

	// The failure happens at wherever we were called, not here
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("the runtime seems quite broken")
	}

	t.Fatalf("Fail at %v:%v; %v", file, line, err)
}

func createBareTestRepo(t *testing.T) *git.Repository {
	// figure out where we can create the test repo
	path, err := ioutil.TempDir("", "gitfs")
	checkFatal(t, err)
	repo, err := git.InitRepository(path, true)
	checkFatal(t, err)

	return repo
}

func seedRepo(t *testing.T, repo *git.Repository) (*git.Oid, *git.Oid) {
	loc, err := time.LoadLocation("Europe/Berlin")
	checkFatal(t, err)
	sig := &git.Signature{
		Name:  "Rand Om Hacker",
		Email: "random@hacker.com",
		When:  time.Date(2013, 03, 06, 14, 30, 0, 0, loc),
	}

	odb, err := repo.Odb()
	checkFatal(t, err)

	blobID, err := odb.Write([]byte("foo\n"), git.ObjectBlob)
	checkFatal(t, err)

	idx, err := git.NewIndex()
	checkFatal(t, err)

	entry := git.IndexEntry{
		Path: "README",
		Id:   blobID,
		Mode: git.FilemodeBlob,
	}

	err = idx.Add(&entry)
	checkFatal(t, err)

	treeId, err := idx.WriteTreeTo(repo)
	checkFatal(t, err)

	message := "This is a commit\n"
	tree, err := repo.LookupTree(treeId)
	checkFatal(t, err)
	commitId, err := repo.CreateCommit("HEAD", sig, sig, message, tree)
	checkFatal(t, err)

	return commitId, treeId
}

func TestGetFile(t *testing.T) {
	repo := createBareTestRepo(t)
	defer os.RemoveAll(repo.Path())

	//_commitID, _treeID := seedRepo(t, repo)
	_, treeID := seedRepo(t, repo)
	tree, err := repo.LookupTree(treeID)
	checkFatal(t, err)

	fs := http.FileServer(NewFromTree(tree))
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://localhost/README", nil)
	checkFatal(t, err)

	fs.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Request did not succeed %v\n", w.Code)
	}

	if w.Body.String() != "foo\n" {
		t.Fatal("bad content served")
	}
}
