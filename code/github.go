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
	return exec.Command(name, args...).Run()
}

func repoOwnerName(repo string) (string, string) {
	ownerNameSlice := strings.Split(repo, "/")
	if len(ownerNameSlice) < 2 {
		panic(errors.New("repository identifier was not in the correct format (i.e. \"owner/name\")"))
	}
	return ownerNameSlice[0], ownerNameSlice[1]
}

func CloneRepository(repo string, dir string) error {
	if FileExists(dir) {
		DeleteDirectory(dir)
	}

	repoUrl := fmt.Sprintf("https://github.com/%s.git", repo)
	if err := runCommand("git", "clone", repoUrl, dir); err != nil {
		return fmt.Errorf("could not clone git repository '%s' to '%s': %v", repo, dir, err)
	}

	return nil
}

func getClient() *gogithub.Client {
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("no GITHUB_AUTH_TOKEN provided")
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

func BranchExists(ctx context.Context, client *gogithub.Client, owner string, name string, branch string) (bool, error) {
	branchInfo, _, err := client.Repositories.GetBranch(ctx, owner, name, branch, 1)
	if err != nil {
		return false, fmt.Errorf("could not get branch info from '%s/%s@%s': %v", owner, name, branch, err)
	}
	return branchInfo != nil, nil
}

func DeleteBranch(owner string, name string, branch string) error {
	if err := runCommand("git", "push", "-d", "origin", branch); err != nil {
		return fmt.Errorf("could not delete branch remotely '%s/%s@%s': %v", owner, name, branch, err)
	}

	if err := runCommand("git", "branch", "-D", branch); err != nil {
		return fmt.Errorf("could not delete branch locally '%s/%s@%s': %v", owner, name, branch, err)
	}

	return nil
}

func PushToBranch(owner string, name string, branch string) error {
	if err := runCommand("git", "checkout", branch); err != nil {
		return fmt.Errorf("could not checkout '%s/%s@%s': %v", owner, name, branch, err)
	}

	if err := runCommand("git", "push", "origin", branch); err != nil {
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

	if err := DeleteSpecificFiles(targetRepoDir+"/.github/workflows", isSyncedFile); err != nil {
		return fmt.Errorf("could not delete synced workflow files from target repo '%s': %w", targetRepo, err)
	}
	if err := CopySpecificFiles(sourceRepoDir+"/.github/workflows", targetRepoDir+"/.github/workflows", isSyncedFile); err != nil {
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

	owner, name := repoOwnerName(targetRepo)
	ctx := context.Background()
	client := getClient()
	defaultBranch, err := getDefaultBranch(ctx, client, owner, name)
	if err != nil {
		return err
	}
	featureBranch := "sync-workflows"

	if exists, err := BranchExists(ctx, client, owner, name, featureBranch); err == nil && exists {
		if err := DeleteBranch(owner, name, featureBranch); err != nil {
			return fmt.Errorf("could not delete old '%s' branch: %w", featureBranch, err)
		}
	} else if err != nil {
		return fmt.Errorf("could not see if old '%s' branch exists: %w", featureBranch, err)
	}

	if err := PushToBranch(owner, name, featureBranch); err != nil {
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
