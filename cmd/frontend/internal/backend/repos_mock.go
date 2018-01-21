package backend

import (
	"sync"
	"testing"

	"context"

	"sourcegraph.com/sourcegraph/sourcegraph/cmd/frontend/internal/db"
	"sourcegraph.com/sourcegraph/sourcegraph/cmd/frontend/internal/pkg/types"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/api/legacyerr"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/inventory"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/vcs"
)

type MockRepos struct {
	Get                  func(v0 context.Context, id int32) (*types.Repo, error)
	GetByURI             func(v0 context.Context, v1 string) (*types.Repo, error)
	List                 func(v0 context.Context, v1 db.ReposListOptions) ([]*types.Repo, error)
	GetCommit            func(v0 context.Context, v1 *types.RepoRevSpec) (*vcs.Commit, error)
	ResolveRev           func(v0 context.Context, repo int32, rev string) (vcs.CommitID, error)
	ListDeps             func(v0 context.Context, v1 []string) ([]string, error)
	GetInventory         func(v0 context.Context, v1 *types.RepoRevSpec) (*inventory.Inventory, error)
	GetInventoryUncached func(ctx context.Context, repoRev *types.RepoRevSpec) (*inventory.Inventory, error)
	RefreshIndex         func(ctx context.Context, repo string) (err error)
}

func (s *MockRepos) MockGet(t *testing.T, wantRepo int32) (called *bool) {
	called = new(bool)
	s.Get = func(ctx context.Context, id int32) (*types.Repo, error) {
		*called = true
		if id != wantRepo {
			t.Errorf("got repo %d, want %d", id, wantRepo)
			return nil, legacyerr.Errorf(legacyerr.NotFound, "repo %d not found", wantRepo)
		}
		return &types.Repo{ID: id}, nil
	}
	return
}

func (s *MockRepos) MockGetByURI(t *testing.T, wantURI string, repoID int32) (called *bool) {
	called = new(bool)
	s.GetByURI = func(ctx context.Context, uri string) (*types.Repo, error) {
		*called = true
		if uri != wantURI {
			t.Errorf("got repo URI %q, want %q", uri, wantURI)
			return nil, legacyerr.Errorf(legacyerr.NotFound, "repo %v not found", uri)
		}
		return &types.Repo{ID: repoID, URI: uri}, nil
	}
	return
}

func (s *MockRepos) MockGet_Return(t *testing.T, returns *types.Repo) (called *bool) {
	called = new(bool)
	s.Get = func(ctx context.Context, id int32) (*types.Repo, error) {
		*called = true
		if id != returns.ID {
			t.Errorf("got repo %d, want %d", id, returns.ID)
			return nil, legacyerr.Errorf(legacyerr.NotFound, "repo %d not found", returns.ID)
		}
		return returns, nil
	}
	return
}

func (s *MockRepos) MockList(t *testing.T, wantRepos ...string) (called *bool) {
	called = new(bool)
	s.List = func(ctx context.Context, opt db.ReposListOptions) ([]*types.Repo, error) {
		*called = true
		repos := make([]*types.Repo, len(wantRepos))
		for i, repo := range wantRepos {
			repos[i] = &types.Repo{URI: repo}
		}
		return repos, nil
	}
	return
}

func (s *MockRepos) MockResolveRev_NoCheck(t *testing.T, commitID vcs.CommitID) (called *bool) {
	var once sync.Once
	called = new(bool)
	s.ResolveRev = func(ctx context.Context, repo int32, rev string) (vcs.CommitID, error) {
		once.Do(func() {
			*called = true
		})
		return commitID, nil
	}
	return
}

func (s *MockRepos) MockResolveRev_NotFound(t *testing.T, wantRepo int32, wantRev string) (called *bool) {
	called = new(bool)
	s.ResolveRev = func(ctx context.Context, repo int32, rev string) (vcs.CommitID, error) {
		*called = true
		if repo != wantRepo {
			t.Errorf("got repo %v, want %v", repo, wantRepo)
		}
		if rev != wantRev {
			t.Errorf("got rev %v, want %v", rev, wantRev)
		}
		return "", legacyerr.Errorf(legacyerr.NotFound, "revision not found")
	}
	return
}

func (s *MockRepos) MockGetCommit_Return_NoCheck(t *testing.T, commit *vcs.Commit) (called *bool) {
	called = new(bool)
	s.GetCommit = func(ctx context.Context, repoRev *types.RepoRevSpec) (*vcs.Commit, error) {
		*called = true
		return commit, nil
	}
	return
}
