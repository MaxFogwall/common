package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gogithub "github.com/google/go-github/v62/github"
	common "github.com/workflow-sync-poc/common/code"
)

type SyncedRepository struct {
	Identifier  string
	Error       error
	ElapsedTime time.Duration
	PullRequest *gogithub.PullRequest
}

func formatRepo(syncedRepo SyncedRepository) string {
	_, name := common.RepoOwnerName(syncedRepo.Identifier)
	return fmt.Sprintf("**[`%s`](https://github.com/%s)**", name, syncedRepo.Identifier)
}

func formatSuccess(syncedRepo SyncedRepository) string {
	if syncedRepo.Error != nil {
		return "❌"
	}

	return "✅"
}

func formatPullRequest(syncedRepo SyncedRepository) string {
	pullRequestString := "No changes needed."
	if syncedRepo.PullRequest != nil {
		pullRequestString = *syncedRepo.PullRequest.HTMLURL
	} else if syncedRepo.Error != nil {
		pullRequestString = "Could not create."
	}

	return fmt.Sprintf("<ul><li>%s</li></ul>", pullRequestString)
}

func formatTime(syncedRepo SyncedRepository) string {
	return syncedRepo.ElapsedTime.Round(time.Second).String()
}

func WriteSyncedRepoSummary(syncedRepo SyncedRepository) {
	var syncedReposTable []string
	var syncedReposErrors []string

	syncedReposTable = append(syncedReposTable, "| Repository | Success | Pull Request | T-Start |")
	syncedReposTable = append(syncedReposTable, "|:-|:-:|:-|-:|")

	if syncedRepo.Error != nil {
		syncedReposErrors = append(syncedReposErrors, fmt.Sprintf("- ❌ %s (%s)", formatRepo(syncedRepo), syncedRepo.Error))
	}

	syncedReposTable = append(syncedReposTable, fmt.Sprintf("| %s | %s | %s | %s |", formatRepo(syncedRepo), formatSuccess(syncedRepo), formatPullRequest(syncedRepo), formatTime(syncedRepo)))

	var summaryLines []string
	summaryLines = append(summaryLines, "### Overview")
	summaryLines = append(summaryLines, strings.Join(syncedReposTable, "\r\n"))
	if len(syncedReposErrors) > 0 {
		summaryLines = append(summaryLines, "### Errors")
		summaryLines = append(summaryLines, strings.Join(syncedReposErrors, "\r\n"))
	}

	common.UpdateJobSummary(strings.Join(summaryLines, "\r\n"))
}

func syncWorkflows(repo string) {
	startTime := time.Now()

	pullRequest, err := common.SyncRepository(repo)
	if err != nil {
		log.Printf("Failed to sync to '%s': %v\n", repo, err)
	}

	WriteSyncedRepoSummary(SyncedRepository{
		Identifier:  repo,
		Error:       err,
		ElapsedTime: time.Since(startTime),
		PullRequest: pullRequest,
	})

	if err != nil {
		panic(errors.New("the repository was not synced successfully"))
	}
}

func main() {
	targetRepo := os.Args[1]
	syncWorkflows(targetRepo)
}
