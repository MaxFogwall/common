package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	gogithub "github.com/google/go-github/v62/github"
)

func runCommand(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	log.Printf("> %s %s", name, sanitize(strings.Join(args, " ")))
	return command.Run()
}

func getCommand(name string, args ...string) *exec.Cmd {
	command := exec.Command(name, args...)
	log.Printf("> %s %s", name, strings.Join(args, " "))
	return command
}

func RepoOwnerName(repo string) (string, string) {
	ownerNameSlice := strings.Split(repo, "/")
	if len(ownerNameSlice) < 2 {
		panic(errors.New("repository identifier was not in the correct format (i.e. \"owner/name\")"))
	}
	return ownerNameSlice[0], ownerNameSlice[1]
}

func SetupGitHubUser(username string, email string) {
	runCommand("git", "config", "user.name", username)
	runCommand("git", "config", "user.email", email)
}

func WriteJobSummary(contents string) {
	WriteFile(getEnv("GITHUB_STEP_SUMMARY"), contents)
}

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("no '%s' provided in ENV", key)
	}
	return value
}

func getClientToken() string {
	return getEnv("GH_PAT_MF")
}

func getApproverClientToken() string {
	return getEnv("GH_PAT_AYYXD")
}

func sanitize(log string) string {
	sanitizied := log
	sensitiveStrings := []string{
		getClientToken(),
		getApproverClientToken(),
	}

	for _, sensitiveString := range sensitiveStrings {
		sanitizied = strings.ReplaceAll(sanitizied, sensitiveString, "<token>")
	}

	return sanitizied
}

func getClient() *gogithub.Client {
	return gogithub.NewClient(nil).WithAuthToken(getClientToken())
}

func getApproverClient() *gogithub.Client {
	return gogithub.NewClient(nil).WithAuthToken(getApproverClientToken())
}

func GetCurrentWorkflowRun() (*gogithub.WorkflowRun, error) {
	ctx := context.Background()
	client := getClient()

	owner, name := RepoOwnerName(getEnv("GO_FILE_REPO"))
	workflowRunString := getEnv("GH_WORKFLOW_RUN_ID")

	runId, err := strconv.ParseInt(workflowRunString, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not convert '%s' to int64: %v", workflowRunString, err)
	}

	workflowRun, response, err := client.Actions.GetWorkflowRunByID(ctx, owner, name, runId)
	if err != nil || !isOk(response) {
		format := "could not get workflow run #%v: %v"
		if err != nil {
			return nil, fmt.Errorf(format, runId, err)
		}
		return nil, fmt.Errorf(format, runId, response.Body)
	}

	return workflowRun, nil
}

func CloneRepository(repo string, dir string) error {
	if PathExists(dir) {
		DeleteDirectory(dir)
	}

	repoUrl := fmt.Sprintf("https://workflow-sync-prototype:%s@github.com/%s.git", getClientToken(), repo)
	if err := runCommand("git", "clone", repoUrl, dir); err != nil {
		return fmt.Errorf("could not clone git repository '%s' to '%s': %v", repo, dir, err)
	}

	if err := runCommand("git", "remote", "set-url", "origin", repoUrl); err != nil {
		return fmt.Errorf("could not clone git repository '%s' to '%s': %v", repo, dir, err)
	}

	return nil
}

func GetDefaultBranch(owner string, name string) (string, error) {
	ctx := context.Background()
	client := getClient()

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

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("could not change directory to '%s': %v", dir, err)
	}

	if execErr := exec(); execErr != nil {
		if err := os.Chdir(workingDir); err != nil {
			return fmt.Errorf("could not change directory back to '%s': %v", dir, err)
		}
		return execErr
	}

	if err := os.Chdir(workingDir); err != nil {
		return fmt.Errorf("could not change directory back to '%s': %v", dir, err)
	}

	return nil
}

func RemoteBranchExists(owner string, name string, branch string) (bool, error) {
	ctx := context.Background()
	client := getClient()

	branchInfo, response, err := client.Repositories.GetBranch(ctx, owner, name, branch, 1)
	if response.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not get remote branch info from '%s/%s@%s': %v", owner, name, branch, err)
	}

	return branchInfo != nil, nil
}

func IsLastCommitClean(dir string) (bool, error) {
	command := getCommand("git", "show", "--name-only", "--pretty=format:", "--", ".github/workflows/synced_*")
	out, err := command.Output()
	if err != nil {
		return false, fmt.Errorf("could not check if working tree was clean: %v", err)
	}

	return string(out) == "", nil
}

func IsWorkingTreeClean(dir string) (bool, error) {
	command := getCommand("git", "status", "--porcelain", "\""+dir+"\"")
	out, err := command.Output()
	if err != nil {
		return false, fmt.Errorf("could not check if working tree was clean: %v", err)
	}

	return string(out) == "", nil
}

func LocalBranchExists(branch string) (bool, error) {
	err := runCommand("git", "rev-parse", "--verify", branch)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.Success(), nil
		}
		return false, fmt.Errorf("could not check if branch '%s' exists locally: %v", branch, err)
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

func CheckoutNewBranch(branch string) error {
	if err := runCommand("git", "checkout", "-b", branch); err != nil {
		return fmt.Errorf("could not checkout new branch '%s': %v", branch, err)
	}

	return nil
}

func CheckoutExistingBranch(branch string) error {
	if err := runCommand("git", "checkout", branch); err != nil {
		return fmt.Errorf("could not checkout existing branch '%s': %v", branch, err)
	}

	return nil
}

func DeleteBranch(owner string, name string, branch string) error {
	defaultBranch, err := GetDefaultBranch(owner, name)
	if err != nil {
		return err
	}

	if err := CheckoutExistingBranch(defaultBranch); err != nil {
		return err
	}

	if exists, err := LocalBranchExists(branch); err != nil {
		return err
	} else if exists {
		if err := DeleteLocalBranch(branch); err != nil {
			return err
		}
	}

	if exists, err := RemoteBranchExists(owner, name, branch); err != nil {
		return err
	} else if exists {
		if err := DeleteRemoteBranch(branch); err != nil {
			return err
		}
	}

	return nil
}

func CreateAndPushToNewBranch(owner string, name string, branch string) (bool, error) {
	if err := DeleteBranch(owner, name, branch); err != nil {
		return false, fmt.Errorf("could not delete old '%s' branch: %w", branch, err)
	}

	if err := CheckoutNewBranch(branch); err != nil {
		return false, err
	}

	if err := runCommand("git", "add", ".github/workflows"); err != nil {
		return false, fmt.Errorf("could not add workflows: %v", err)
	}

	if clean, err := IsWorkingTreeClean("."); err != nil {
		return false, err
	} else if clean {
		log.Println("No changes to commit, we are up to date!")
		return false, nil
	}

	if err := runCommand("git", "commit", "-m", "sync workflows"); err != nil {
		return false, fmt.Errorf("could not commit changes: %v", err)
	}

	if err := runCommand("git", "push", "-u", "origin", branch); err != nil {
		return false, fmt.Errorf("could not push to remote '%s': %v", branch, err)
	}

	return true, nil
}

func GetLatestTag() (string, error) {
	command := getCommand("git", "describe", "--tags", "--abbrev=0")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		if stderr.String() == "fatal: No names found, cannot describe anything." {
			return "", nil
		}

		return "", fmt.Errorf("could not get latest tag (%s): %v", string(output), err)
	}

	return string(output), nil
}

func AddTag(tag string) error {
	if err := runCommand("git", "tag", tag); err != nil {
		return fmt.Errorf("could not update local tag '%s': %v", tag, err)
	}

	if err := runCommand("git", "push", "origin", tag); err != nil {
		return fmt.Errorf("could not push remote tag '%s': %v", tag, err)
	}

	return nil
}

func MoveTag(tag string) error {
	// See recommendation from https://github.com/actions/toolkit/blob/master/docs/action-versioning.md
	if err := runCommand("git", "tag", "-fa", tag, "-m", fmt.Sprintf("Update tag `%s` to latest commit", tag)); err != nil {
		return fmt.Errorf("could not update local tag '%s': %v", tag, err)
	}

	if err := runCommand("git", "push", "origin", tag, "--force"); err != nil {
		return fmt.Errorf("could not push remote tag '%s': %v", tag, err)
	}

	return nil
}

func locallySync(targetRepo string, targetRepoDir string, sourceRef string) error {
	if err := CloneRepository(targetRepo, targetRepoDir); err != nil {
		return err
	}

	syncedFilePattern := regexp.MustCompile(`synced_.+\.y(a)?ml`)
	isSyncedFile := func(info os.FileInfo) bool {
		return syncedFilePattern.MatchString(info.Name())
	}

	replaceRef := func(contents string) string {
		return strings.ReplaceAll(contents, "@main", "@"+sourceRef)
	}

	targetWorkflowPath := targetRepoDir + "/.github/workflows"
	sourceWorkflowPath := ".github/workflows"

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

	if err := ModifySpecificFiles(targetWorkflowPath, isSyncedFile, replaceRef); err != nil {
		return fmt.Errorf("could not replace ref of workflows in target repo '%s': %v", targetRepo, err)
	}

	return nil
}

func isOk(response *gogithub.Response) bool {
	statusCodeString := fmt.Sprintf("%v", response.StatusCode)
	return statusCodeString[0] != '4' && statusCodeString[0] != '5'
}

func CreatePullRequest(owner string, name string, branch string, title string, workflowRun *gogithub.WorkflowRun) (*gogithub.PullRequest, error) {
	ctx := context.Background()
	client := getClient()

	log.Println("- Creating pull request...")

	defaultBranch, err := GetDefaultBranch(owner, name)
	if err != nil {
		return nil, err
	}

	pullRequest, response, err := client.PullRequests.Create(ctx, owner, name, &gogithub.NewPullRequest{
		Title:               gogithub.String(title),
		Head:                gogithub.String(branch),
		Base:                gogithub.String(defaultBranch),
		Body:                gogithub.String(fmt.Sprintf("*Automatically generated from [workflow run **%s** #%v](%s) in [%s](%s).*", *workflowRun.Name, *workflowRun.RunNumber, *workflowRun.HTMLURL, *workflowRun.Repository.FullName, *workflowRun.Repository.HTMLURL)),
		MaintainerCanModify: gogithub.Bool(true),
	})
	if err != nil || !isOk(response) {
		format := "could not create pull request from '%s' to '%s': %v"
		if err != nil {
			return pullRequest, fmt.Errorf(format, branch, defaultBranch, err)
		}
		return pullRequest, fmt.Errorf(format, branch, defaultBranch, response.Body)
	}

	return pullRequest, nil
}

func ApprovePullRequest(owner string, name string, pullRequest *gogithub.PullRequest) error {
	ctx := context.Background()
	client := getApproverClient()

	log.Println("- Approving pull request...")

	_, response, err := client.PullRequests.CreateReview(ctx, owner, name, *pullRequest.Number, &gogithub.PullRequestReviewRequest{
		Event: gogithub.String("APPROVE"),
	})
	if err != nil || !isOk(response) {
		format := "could not approve pull request #%v: %v"
		if err != nil {
			return fmt.Errorf(format, *pullRequest.Number, err)
		}
		return fmt.Errorf(format, *pullRequest.Number, response.Body)
	}

	return nil
}

func MergePullRequest(owner string, name string, pullRequest *gogithub.PullRequest) error {
	ctx := context.Background()
	client := getClient()

	log.Println("- Merging pull request...")

	_, response, err := client.PullRequests.Merge(ctx, owner, name, *pullRequest.Number, "", &gogithub.PullRequestOptions{})
	if err != nil || !isOk(response) {
		format := "could not merge pull request #%v: %v"
		if err != nil {
			return fmt.Errorf(format, *pullRequest.Number, err)
		}
		return fmt.Errorf(format, *pullRequest.Number, response.Body)
	}

	return nil
}

func SyncRepository(repo string, sourceRef string) (*gogithub.PullRequest, error) {
	owner, name := RepoOwnerName(repo)
	repoDir := name
	if err := locallySync(repo, repoDir, sourceRef); err != nil {
		return nil, fmt.Errorf("could not sync locally: %w", err)
	}

	featureBranch := "sync-workflows"
	changesPushed := false
	err := ExecInDir(repoDir, func() error {
		SetupGitHubUser("workflow-sync-bot", "workflow-sync.bot@example.com")
		success, err := CreateAndPushToNewBranch(owner, name, featureBranch)
		changesPushed = success
		if err != nil {
			return fmt.Errorf("could not create and push to new branch '%s': %w", featureBranch, err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if !changesPushed {
		// There were no changes, so we have nothing to make a pull request of.
		return nil, nil
	}

	workflowRun, err := GetCurrentWorkflowRun()
	if err != nil {
		return nil, err
	}

	pullRequest, err := CreatePullRequest(owner, name, featureBranch, "(sync): update workflows", workflowRun)
	if err != nil {
		return pullRequest, err
	}

	if err := ApprovePullRequest(owner, name, pullRequest); err != nil {
		return pullRequest, err
	}

	if err := MergePullRequest(owner, name, pullRequest); err != nil {
		return pullRequest, err
	}

	err = ExecInDir(repoDir, func() error {
		SetupGitHubUser("workflow-sync-bot", "workflow-sync.bot@example.com")
		if err := DeleteBranch(owner, name, featureBranch); err != nil {
			return fmt.Errorf("could not delete merged '%s' branch: %w", featureBranch, err)
		}

		return nil
	})
	if err != nil {
		return pullRequest, err
	}

	return pullRequest, nil
}
