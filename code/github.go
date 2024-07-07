package common

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	gogithub "github.com/google/go-github/v62/github"
)

func runCommand(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Printf("> %s %s", name, strings.Join(args, " "))
	return command.Run()
}

func runAndOutputCommand(name string, args ...string) ([]byte, error) {
	command := exec.Command(name, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Printf("> %s %s", name, strings.Join(args, " "))
	return command.Output()
}

func RepoOwnerName(repo string) (string, string) {
	ownerNameSlice := strings.Split(repo, "/")
	if len(ownerNameSlice) < 2 {
		panic(errors.New("repository identifier was not in the correct format (i.e. \"owner/name\")"))
	}
	return ownerNameSlice[0], ownerNameSlice[1]
}

func CloneRepository(repo string, dir string) error {
	if PathExists(dir) {
		DeleteDirectory(dir)
	}

	repoUrl := fmt.Sprintf("https://github.com/%s.git", repo)
	if err := runCommand("git", "clone", repoUrl, dir); err != nil {
		return fmt.Errorf("could not clone git repository '%s' to '%s': %v", repo, dir, err)
	}

	if err := runCommand("git", "remote", "set-url", "origin", repoUrl); err != nil {
		return fmt.Errorf("could not clone git repository '%s' to '%s': %v", repo, dir, err)
	}

	return nil
}

func getClient() *gogithub.Client {
	token := os.Getenv("GH_AUTH_TOKEN")
	if token == "" {
		log.Fatal("no GH_AUTH_TOKEN provided")
	}

	return gogithub.NewClient(nil).WithAuthToken(token)
}

func getDefaultBranch(ctx context.Context, client *gogithub.Client, owner string, name string) (string, error) {
	repoInfo, _, err := client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return "", fmt.Errorf("could not get repository info from '%s/%s': %v", owner, name, err)
	}
	return repoInfo.GetDefaultBranch(), nil
}

func ExecInDir(dir string, exec func() error) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get working directory: %v", err)
	}

	if err := runCommand("cd", workingDir+"/"+dir); err != nil {
		return fmt.Errorf("could not change directory to '%s': %v", dir, err)
	}

	if execErr := exec(); execErr != nil {
		if err := runCommand("cd", workingDir); err != nil {
			return fmt.Errorf("could not change directory back to '%s': %v", dir, err)
		}
		return execErr
	}

	return nil
}

func RemoteBranchExists(ctx context.Context, client *gogithub.Client, owner string, name string, branch string) (bool, error) {
	branchInfo, response, err := client.Repositories.GetBranch(ctx, owner, name, branch, 1)
	if response.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not get remote branch info from '%s/%s@%s': %v", owner, name, branch, err)
	}
	return branchInfo != nil, nil
}

func LocalBranchExists(branch string) (bool, error) {
	_, err := runAndOutputCommand("git", "rev-parse", "--verify", branch)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.Success(), nil
		}
		return false, fmt.Errorf("could not verify whether branch '%s' exists locally: %v", branch, err)
	}

	return true, nil
}

func DeleteLocalBranch(branch string) error {
	if err := runCommand("git", "branch", "-D", branch); err != nil {
		return fmt.Errorf("could not delete local branch '%s': %v", branch, err)
	}

	return nil
}

func DeleteRemoteBranch(branch string) error {
	if err := runCommand("git", "push", "origin", "--delete", branch); err != nil {
		return fmt.Errorf("could not delete remote branch '%s': %v", branch, err)
	}

	return nil
}

func DeleteBranch(ctx context.Context, client *gogithub.Client, owner string, name string, branch string) error {
	if exists, err := LocalBranchExists(branch); err != nil {
		return err
	} else if exists {
		if err := DeleteLocalBranch(branch); err != nil {
			return err
		}
	}

	if exists, err := RemoteBranchExists(ctx, client, owner, name, branch); err != nil {
		return err
	} else if exists {
		if err := DeleteRemoteBranch(branch); err != nil {
			return err
		}
	}

	return nil
}

func CreateAndPushToNewBranch(ctx context.Context, client *gogithub.Client, owner string, name string, branch string) error {
	if err := DeleteBranch(ctx, client, owner, name, branch); err != nil {
		return fmt.Errorf("could not delete old '%s' branch: %w", branch, err)
	}

	if err := runCommand("git", "checkout", "-b", branch); err != nil {
		return fmt.Errorf("could not checkout '%s/%s@%s': %v", owner, name, branch, err)
	}

	if err := runCommand("git", "push", "-u", "origin", branch); err != nil {
		return fmt.Errorf("could not push to remote '%s/%s@%s': %v", owner, name, branch, err)
	}

	return nil
}

func locallySync(targetRepo string, sourceRepoDir string) error {
	targetRepoDir := "TargetRepo"
	if err := CloneRepository(targetRepo, targetRepoDir); err != nil {
		return err
	}

	syncedFilePattern := regexp.MustCompile(`synced_.+\.y(a)?ml`)
	isSyncedFile := func(info os.FileInfo) bool {
		return syncedFilePattern.MatchString(info.Name())
	}

	targetWorkflowPath := targetRepoDir + "/.github/workflows"
	sourceWorkflowPath := sourceRepoDir + "/.github/workflows"

	if !PathExists(targetWorkflowPath) {
		if err := CreateDirectory(targetWorkflowPath); err != nil {
			return fmt.Errorf("could not create workflow path for target repo '%s': %w", targetRepo, err)
		}
	}

	if err := DeleteSpecificFiles(targetWorkflowPath, isSyncedFile); err != nil {
		return fmt.Errorf("could not delete synced workflow files from target repo '%s': %w", targetRepo, err)
	}
	if err := CopySpecificFiles(sourceWorkflowPath, targetWorkflowPath, isSyncedFile); err != nil {
		return fmt.Errorf("could not copy synced workflow files to target repo '%s': %w", targetRepo, err)
	}

	return nil
}

func isOk(response *gogithub.Response) bool {
	statusCodeString := fmt.Sprintf("%v", response.StatusCode)
	return statusCodeString[0] != '2'
}

func SyncRepository(targetRepo string, sourceRepoDir string) error {
	if err := locallySync(targetRepo, sourceRepoDir); err != nil {
		return fmt.Errorf("could not sync locally: %w", err)
	}

	owner, name := RepoOwnerName(targetRepo)
	ctx := context.Background()
	client := getClient()

	featureBranch := "sync-workflows"
	defaultBranch, err := getDefaultBranch(ctx, client, owner, name)

	if err != nil {
		return err
	}

	err = ExecInDir(targetRepo, func() error {
		if err := CreateAndPushToNewBranch(ctx, client, owner, name, featureBranch); err != nil {
			return fmt.Errorf("could not create and push to new branch '%s': %w", featureBranch, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	_, response, err := client.PullRequests.Create(ctx, owner, name, &gogithub.NewPullRequest{
		Title:               gogithub.String("(sync): update workflows"),
		Head:                gogithub.String(featureBranch),
		Base:                gogithub.String(defaultBranch),
		Body:                gogithub.String(""), // TODO: Link the workflow run in the description.
		MaintainerCanModify: gogithub.Bool(true),
	})

	if err != nil || !isOk(response) {
		format := "could not create pull request from '%s' to '%s': %v"
		if err != nil {
			return fmt.Errorf(format, featureBranch, defaultBranch, err)
		}
		return fmt.Errorf(format, featureBranch, defaultBranch, response)
	}

	// TODO: Approve pull request.
	// TODO: Merge pull request.
	// TODO: Delete feature branch.

	return nil
}
