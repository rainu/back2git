package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
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

func (s *store) Save(file *os.File) error {
	name := strings.TrimLeft(file.Name(), "/")
	targetPath := path.Join(s.repoRootDir, name)

	err := os.MkdirAll(path.Dir(targetPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create parent dir(s): %w", err)
	}

	if _, err := os.Stat(targetPath); err == nil {
		//this should not happen normally (because the hardlink is only temporarily available)
		err := os.Remove(targetPath)
		if err != nil {
			return fmt.Errorf("unable to renew hardlink: %w", err)
		}
	}

	err = os.Link(file.Name(), targetPath)
	if err != nil {
		return fmt.Errorf("cannot create temporary hardlink: %w", err)
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

	//We have to remove the hardlink eminently! Otherwise, the deletion of the "original" file
	//will not be detected!
	err = os.Remove(targetPath)
	if err != nil {
		return fmt.Errorf("unable to remove temporary hardlink: %w", err)
	}

	return nil
}

func (s *store) Delete(filePath string) error {
	name := strings.TrimLeft(filePath, "/")

	wt, err := s.repository.Worktree()
	if err != nil {
		return fmt.Errorf("cannot access the git working tree: %w", err)
	}
	_, err = wt.Remove(name)
	if err != nil {
		return fmt.Errorf("cannot add file to workingtree index: %w", err)
	}
	_, err = wt.Commit(fmt.Sprintf("Delete %s", name), &git.CommitOptions{})
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
