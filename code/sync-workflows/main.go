package main

import (
	"encoding/json"
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

func getTargetRepos() []string {
	reposJsonPath := "repos.json"
	reposJson, err := common.ReadFile(reposJsonPath)
	if err != nil {
		panic(fmt.Errorf("could not read '%s': %v", reposJsonPath, err))
	}

	var repos []string
	err = json.Unmarshal([]byte(reposJson), &repos)
	if err != nil {
		panic(fmt.Errorf("could not parse '%s', expected a JSON formatted list of strings: %v", reposJsonPath, err))
	}

	return repos
}

func formatRepo(syncedRepo SyncedRepository) string {
	_, name := common.RepoOwnerName(syncedRepo.Identifier)
	return fmt.Sprintf("**[`%s`](https://github.com/%s)**", name, syncedRepo.Identifier)
}

func formatSuccess(syncedRepo SyncedRepository) string {
	if syncedRepo.Error != nil {
		return "❌"
	}

	return "✔️"
}

func formatPullRequestStatus(syncedRepo SyncedRepository) string {
	mergedImageUrl := "https://github.com/workflow-sync-poc/component-1/assets/48988185/43f86b74-a8eb-4df3-a2f3-ac8e714784b5"
	openImageUrl := "https://github.com/workflow-sync-poc/component-1/assets/48988185/bc28bc57-f91c-4389-a103-d4524d4f6e39"

	format := "<img align=\"center\" src=\"%s\"/>"

	if syncedRepo.Error != nil {
		return fmt.Sprintf(format, openImageUrl)
	}

	return fmt.Sprintf(format, mergedImageUrl)
}

func formatPullRequest(syncedRepo SyncedRepository) string {
	pullRequestString := "No changes needed."
	if syncedRepo.PullRequest != nil {
		pullRequestString = fmt.Sprintf("%s [**%s**](%s) #%v", formatPullRequestStatus(syncedRepo), *syncedRepo.PullRequest.Title, *syncedRepo.PullRequest.HTMLURL, *syncedRepo.PullRequest.Number)
	} else if syncedRepo.Error != nil {
		pullRequestString = "Could not create."
	}

	return fmt.Sprintf("<ul><li>%s</li></ul>", pullRequestString)
}

func formatTime(syncedRepo SyncedRepository) string {
	return syncedRepo.ElapsedTime.Round(time.Second).String()
}

func GetSyncedReposTableAndErrors(syncedRepos []SyncedRepository) string {
	var syncedReposTable []string
	var syncedReposErrors []string

	syncedReposTable = append(syncedReposTable, "| Repository | Success | Pull Request | T-Start |")
	syncedReposTable = append(syncedReposTable, "|:-|:-:|:-|-:|")

	for _, syncedRepo := range syncedRepos {
		if syncedRepo.Error != nil {
			syncedReposErrors = append(syncedReposErrors, fmt.Sprintf("- ❌ %s (%s)", formatRepo(syncedRepo), syncedRepo.Error))
		}

		syncedReposTable = append(syncedReposTable, fmt.Sprintf("| %s | %s | %s | %s |", formatRepo(syncedRepo), formatSuccess(syncedRepo), formatPullRequest(syncedRepo), formatTime(syncedRepo)))
	}

	var tableAndErrorsLines []string
	tableAndErrorsLines = append(tableAndErrorsLines, strings.Join(syncedReposTable, "\r\n"))
	if len(syncedReposErrors) > 0 {
		tableAndErrorsLines = append(tableAndErrorsLines, strings.Join(syncedReposErrors, "\r\n"))
	}

	return strings.Join(tableAndErrorsLines, "\r\n")
}

func AnySyncedRepoHasError(syncedRepos []SyncedRepository) bool {
	for _, syncedRepo := range syncedRepos {
		if syncedRepo.Error != nil {
			return true
		}
	}

	return false
}

func AnySyncedRepoSucceeded(syncedRepos []SyncedRepository) bool {
	for _, syncedRepo := range syncedRepos {
		if syncedRepo.Error == nil {
			return true
		}
	}

	return false
}

func updateLastSynced(dir string) {
	err := common.ExecInDir(dir, func() error {
		common.SetupGitHubUser()
		if err := common.SetOrigin("workflow-sync-poc/common"); err != nil {
			return err
		}

		if err := common.AddOrMoveTag("last-synced"); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func main() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	sourceRepo, err := common.GetCurrentRepository()
	if err != nil {
		panic(err)
	}

	versionTag, err := common.GetLatestVersionTag(sourceRepo)
	if err != nil {
		panic(err)
	}
	if versionTag == "" {
		panic(fmt.Errorf("could not get latest version tag, it returned \"\""))
	}

	startTime := time.Now()
	syncedRepos := []SyncedRepository{}
	targetRepos := getTargetRepos()

	for _, targetRepo := range targetRepos {
		pullRequest, err := common.SyncRepository(targetRepo, versionTag)
		if err != nil {
			log.Printf("Failed to sync to '%s': %v\n", targetRepo, err)
		}

		syncedRepository := SyncedRepository{
			Identifier:  targetRepo,
			Error:       err,
			ElapsedTime: time.Since(startTime),
			PullRequest: pullRequest,
		}

		syncedRepos = append(syncedRepos, syncedRepository)
	}

	tableAndErrors := GetSyncedReposTableAndErrors(syncedRepos)

	var summaryLines []string
	success := !AnySyncedRepoHasError(syncedRepos)

	if success {
		updateLastSynced(workingDirectory)
		summaryLines = append(summaryLines, fmt.Sprintf("### 🟢 All Repos Now Use `%s` For Workflows", versionTag))
	} else {
		partialSuccess := AnySyncedRepoSucceeded(syncedRepos)
		if partialSuccess {
			summaryLines = append(summaryLines, fmt.Sprintf("### 🟡 Some Repos Now Use `%s` For Workflows", versionTag))
		} else {
			summaryLines = append(summaryLines, "### 🔴 No Workflows Changed")
		}
	}

	summaryLines = append(summaryLines, tableAndErrors)
	common.WriteJobSummary(strings.Join(summaryLines, "\r\n"))

	if !success {
		panic(errors.New("one or more repositories were not synced successfully"))
	}
}
