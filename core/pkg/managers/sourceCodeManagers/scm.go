package sourcecodemanager

import (
	"fmt"
	"os"
	"os/exec"

	vulkanErrors "github.com/mofe64/vulcan/pkg/errors"
	"github.com/mofe64/vulcan/pkg/utils"
)

// CloneRemoteRepository: clones the given repository URL into a temp directory.
func CloneRemoteRepository(deploymentId string, repoURL string) error {
	if !utils.CheckGitInstalled() {
		return fmt.Errorf("%w: git must be installed on the system", vulkanErrors.ErrGitNotInstalled)
	}
	scTempDir, err := utils.CreateSourceDir(deploymentId)
	if err != nil {
		return fmt.Errorf("failed to create source directory: %v", err)
	}

	cmd := exec.Command("git", "clone", repoURL, scTempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", vulkanErrors.ErrGitCloneFailed, err)
	}
	return nil

}

// CloneBranchFromRemoteRepositoryclones the specified branch from the repository URL into a temp directory.
func CloneBranchFromRemoteRepository(deploymentId string, repoURL, branch string) error {
	if !utils.CheckGitInstalled() {
		return fmt.Errorf("%w: git must be installed on the system", vulkanErrors.ErrGitNotInstalled)
	}
	scTempDir, err := utils.CreateSourceDir(deploymentId)
	if err != nil {
		return fmt.Errorf("failed to create source directory: %v", err)
	}
	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", repoURL, scTempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone branch %s from repository %s: %v", branch, repoURL, err)
	}

	return nil
}
