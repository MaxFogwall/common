package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	gogithub "github.com/google/go-github/v62/github"
)

func runCommand(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = io.MultiWriter(os.Stdout, &stdout)
	command.Stderr = io.MultiWriter(os.Stderr, &stderr)

	log.Printf("> %s %s", name, sanitize(strings.Join(args, " ")))

	err := command.Run()
	if err != nil && stderr.Len() > 0 {
		err = fmt.Errorf("%s", stderr.String())
	}

	return stdout.String(), err
}

func RepoOwnerName(repo string) (string, string) {
	ownerNameSlice := strings.Split(repo, "/")
	if len(ownerNameSlice) < 2 {
		panic(errors.New("repository identifier was not in the correct format (i.e. \"owner/name\")"))
	}
	return ownerNameSlice[0], ownerNameSlice[1]
}

func SetupGitHubUser() {
	runCommand("git", "config", "user.name", "workflow-sync-bot")
	runCommand("git", "config", "user.email", "workflow-sync.bot@example.com")
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

func SetOrigin(repo string) error {
	repoUrl := fmt.Sprintf("https://workflow-sync-bot:%s@github.com/%s.git", getClientToken(), repo)
	if _, err := runCommand("git", "remote", "set-url", "origin", repoUrl); err != nil {
		return fmt.Errorf("could not set url to git repository '%s': %v", repo, err)
	}

	return nil
}

func CloneRepository(repo string, dir string) error {
	if PathExists(dir) {
		DeleteDirectory(dir)
	}

	repoUrl := fmt.Sprintf("https://workflow-sync-bot:%s@github.com/%s.git", getClientToken(), repo)
	if _, err := runCommand("git", "clone", repoUrl, dir); err != nil {
		return fmt.Errorf("could not clone git repository '%s' to '%s': %v", repo, dir, err)
	}

	if err := SetOrigin(repo); err != nil {
		return err
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

func GetFilesChangedSince(tag string, dir string) ([]string, error) {
	out, err := runCommand("git", "diff", "--name-only", tag, "--", dir)
	if err != nil {
		return nil, fmt.Errorf("could not check if working tree was clean: %v", err)
	}

	if out == "" {
		return []string{}, nil
	}

	var filesChanged []string
	for _, fileChanged := range strings.Split(out, "\n") {
		if len(fileChanged) > 0 {
			filesChanged = append(filesChanged, fileChanged)
		}
	}

	return filesChanged, nil
}

func GetFilesChangedInLastCommit(dir string) ([]string, error) {
	return GetFilesChangedSince("HEAD^", dir)
}

func IsWorkingTreeClean() (bool, error) {
	out, err := runCommand("git", "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("could not check if working tree was clean: %v", err)
	}

	return string(out) == "", nil
}

func LocalBranchExists(branch string) (bool, error) {
	out, err := runCommand("git", "branch", "--list", branch)
	if err != nil {
		return false, fmt.Errorf("could not check if branch '%s' exists locally: %v", branch, err)
	}

	return string(out) != "", nil
}

func DeleteLocalBranch(branch string) error {
	if _, err := runCommand("git", "branch", "-D", branch); err != nil {
		return fmt.Errorf("could not delete local branch '%s': %v", branch, err)
	}

	return nil
}

func DeleteRemoteBranch(branch string) error {
	if _, err := runCommand("git", "push", "origin", "--delete", branch); err != nil {
		return fmt.Errorf("could not delete remote branch '%s': %v", branch, err)
	}

	return nil
}

func CheckoutNewBranch(branch string) error {
	if _, err := runCommand("git", "checkout", "-b", branch); err != nil {
		return fmt.Errorf("could not checkout new branch '%s': %v", branch, err)
	}

	return nil
}

func CheckoutExistingBranch(branch string) error {
	if _, err := runCommand("git", "checkout", branch); err != nil {
		return fmt.Errorf("could not checkout existing branch '%s': %v", branch, err)
	}

	return nil
}

func GetCurrentRepository() (string, error) {
	repoUrl, err := runCommand("git", "config", "--get", "remote.origin.url")
	if err != nil {
		return "", fmt.Errorf("could not get current repository: %v", err)
	}
	if repoUrl == "" {
		return "", fmt.Errorf("could not get current repository, it returned \"\"")
	}

	log.Printf("`repoUrl`: %s", repoUrl)

	// E.g. "https://github.com/workflow-sync-poc/common.git" -> "workflow-sync-poc/common"
	repoFromUrlPattern := regexp.MustCompile(`https:\/\/github\.com\/(?P<Repo>[^\.]+)`)
	repoFromUrlSubmatches := repoFromUrlPattern.FindStringSubmatch(repoUrl)
	repoFromUrlSubmatchIndex := repoFromUrlPattern.SubexpIndex("Repo")
	repo := string(repoFromUrlSubmatches[repoFromUrlSubmatchIndex])

	log.Printf("`repo`: %s", repo)

	return repo, nil
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

	if _, err := runCommand("git", "checkout", "-b", branch); err != nil {
		return false, fmt.Errorf("could not create branch '%s': %v", branch, err)
	}

	if _, err := runCommand("git", "add", ".github/workflows"); err != nil {
		return false, fmt.Errorf("could not add workflows: %v", err)
	}

	if clean, err := IsWorkingTreeClean(); err != nil {
		return false, err
	} else if clean {
		log.Println("No changes to commit, we are up to date!")
		return false, nil
	}

	if _, err := runCommand("git", "commit", "-m", "sync workflows"); err != nil {
		return false, fmt.Errorf("could not commit changes: %v", err)
	}

	if _, err := runCommand("git", "push", "-u", "origin", branch); err != nil {
		return false, fmt.Errorf("could not push to remote '%s': %v", branch, err)
	}

	return true, nil
}

func GetLatestVersionTag(repo string) (string, error) {
	err := SetOrigin(repo)
	if err != nil {
		return "", err
	}

	output, err := runCommand("bash", "-c", `git ls-remote --tags origin | grep -o 'refs/tags/v.*' | sed 's#refs/tags/##; s#\^{}##' | sort -V | tail -n1`)
	if err != nil {
		return "", fmt.Errorf("could not get latest tag: %v", err)
	}

	latestTag := strings.TrimSuffix(string(output), "\n")
	return latestTag, nil
}

func AddTag(tag string) error {
	if _, err := runCommand("git", "tag", tag); err != nil {
		return fmt.Errorf("could not update local tag '%s': %v", tag, err)
	}

	if _, err := runCommand("git", "push", "origin", tag); err != nil {
		return fmt.Errorf("could not push remote tag '%s': %v", tag, err)
	}

	return nil
}

func MoveTag(tag string) error {
	// See recommendation from https://github.com/actions/toolkit/blob/master/docs/action-versioning.md
	if _, err := runCommand("git", "tag", "-fa", tag, "-m", fmt.Sprintf("Update tag `%s` to latest commit", tag)); err != nil {
		return fmt.Errorf("could not update local tag '%s': %v", tag, err)
	}

	if _, err := runCommand("git", "push", "origin", tag, "--force"); err != nil {
		return fmt.Errorf("could not push remote tag '%s': %v", tag, err)
	}

	return nil
}

func TagExists(tag string) (bool, error) {
	if _, err := runCommand("bash", "-c", fmt.Sprintf("git ls-remote --tags origin | grep -q \"refs/tags/%s\"", tag)); err != nil {
		// `grep` returns an exit code of 1 if no match is found.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("could not check whether tag '%s' exists: %v", tag, err)
	}

	return true, nil
}

func AddOrMoveTag(tag string) error {
	tagExists, err := TagExists(tag)
	if err != nil {
		return fmt.Errorf("could not add or move tag '%s': %v", tag, err)
	}

	if !tagExists {
		err = AddTag(tag)
	} else {
		err = MoveTag(tag)
	}

	if err != nil {
		return fmt.Errorf("could not add or move tag '%s': %v", tag, err)
	}

	return nil
}

func locallySync(sourceRepo string, targetRepo string, targetRepoDir string) error {
	if err := CloneRepository(targetRepo, targetRepoDir); err != nil {
		return err
	}

	syncedFilePattern := regexp.MustCompile(`synced_.+\.y(a)?ml`)
	isSyncedFile := func(info os.FileInfo) bool {
		return syncedFilePattern.MatchString(info.Name())
	}

	versionTag, err := GetLatestVersionTag(sourceRepo)
	if err != nil {
		return err
	}
	if versionTag == "" {
		return fmt.Errorf("could not get latest version tag, it returned \"\"")
	}

	replaceRef := func(contents string) string {
		return strings.ReplaceAll(contents, "@main", "@"+versionTag)
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

func SyncRepository(sourceRepo string, targetRepo string) (*gogithub.PullRequest, error) {
	targetOwner, targetName := RepoOwnerName(targetRepo)
	targetRepoDir := targetName
	if err := locallySync(sourceRepo, targetRepo, targetRepoDir); err != nil {
		return nil, fmt.Errorf("could not sync locally: %w", err)
	}

	featureBranch := "sync-workflows"
	changesPushed := false
	err := ExecInDir(targetRepoDir, func() error {
		SetupGitHubUser()
		success, err := CreateAndPushToNewBranch(targetOwner, targetName, featureBranch)
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

	pullRequest, err := CreatePullRequest(targetOwner, targetName, featureBranch, "(sync): update workflows", workflowRun)
	if err != nil {
		return pullRequest, err
	}

	if err := ApprovePullRequest(targetOwner, targetName, pullRequest); err != nil {
		return pullRequest, err
	}

	if err := MergePullRequest(targetOwner, targetName, pullRequest); err != nil {
		return pullRequest, err
	}

	err = ExecInDir(targetRepoDir, func() error {
		SetupGitHubUser()
		if err := DeleteBranch(targetOwner, targetName, featureBranch); err != nil {
			return fmt.Errorf("could not delete merged '%s' branch: %w", featureBranch, err)
		}

		return nil
	})
	if err != nil {
		return pullRequest, err
	}

	return pullRequest, nil
}
