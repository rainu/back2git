package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
)

type store struct {
	repoRootDir    string
	repository     *git.Repository
	authentication transport.AuthMethod
}

func NewGitStore(repoUrl, branch, path string, auth transport.AuthMethod) (*store, error) {
	repo, err := git.PlainOpen(path)
	if err == git.ErrRepositoryNotExists {
		log.Info("Start cloning repository.")

		var referenceName plumbing.ReferenceName
		if branch != "" {
			referenceName = plumbing.NewBranchReferenceName(branch)
		}

		repo, err = git.PlainClone(path, false, &git.CloneOptions{
			URL:           repoUrl,
			ReferenceName: referenceName,
			Auth:          auth,
			Progress:      os.Stderr,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to clone repository: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("unable to open repository: %w", err)
	}

	return &store{
		repoRootDir:    path,
		repository:     repo,
		authentication: auth,
	}, nil
}

func (s *store) Save(name string, content io.Reader) error {
	targetPath := path.Join(s.repoRootDir, name)

	err := os.MkdirAll(path.Dir(targetPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create parent dir(s): %w", err)
	}
	targetFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}

	_, err = io.Copy(targetFile, content)
	if err != nil {
		return fmt.Errorf("cannot write file content: %w", err)
	}

	wt, err := s.repository.Worktree()
	if err != nil {
		return fmt.Errorf("cannot access the git working tree: %w", err)
	}
	_, err = wt.Add(name)
	if err != nil {
		return fmt.Errorf("cannot add file to workingtree index: %w", err)
	}
	_, err = wt.Commit(fmt.Sprintf("New version of %s", name), &git.CommitOptions{})
	if err != nil {
		return fmt.Errorf("cannot commit change: %w", err)
	}

	return nil
}

func (s *store) Sync() error {
	log.Info("Start pushing to remote repository.")
	err := s.repository.Push(&git.PushOptions{
		Auth:     s.authentication,
		Progress: os.Stderr,
	})

	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return err
}
