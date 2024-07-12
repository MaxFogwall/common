package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	common "github.com/workflow-sync-poc/common/code"
)

func nextMajorVersionForTag(tag string) int {
	majorVersionPattern := regexp.MustCompile(`v(?P<MajorVersion>\d+)`)
	majorVersionSubmatches := majorVersionPattern.FindStringSubmatch(tag)
	majorVersionSubmatchIndex := majorVersionPattern.SubexpIndex("MajorVersion")
	majorVersionString := majorVersionSubmatches[majorVersionSubmatchIndex]
	majorVersion, err := strconv.Atoi(majorVersionString)
	if err != nil {
		panic(fmt.Errorf("could not parse major version from tag '%s': %v", tag, err))
	}

	return majorVersion + 1
}

func getSyncedReposDefinitionChangedSince(sinceTag string) []string {
	syncedReposDefinitionChanged, err := common.GetFilesChangedSince(sinceTag, "repos.json")
	if err != nil {
		panic(err)
	}

	return syncedReposDefinitionChanged
}

func getSyncedWorkflowsChangedSince(sinceTag string) []string {
	syncedWorkflowsChanged, err := common.GetFilesChangedSince(sinceTag, ".github/workflows/synced_*")
	if err != nil {
		panic(err)
	}

	return syncedWorkflowsChanged
}

func getSyncedWorkflowsChangedInCurrentCommit() []string {
	return getSyncedWorkflowsChangedSince("HEAD^")
}

func shouldIncrementTag() bool {
	return len(getSyncedWorkflowsChangedInCurrentCommit()) > 0
}

func reasonToSyncWorkflows() string {
	lastSyncedTag := "last-synced"

	hasEverSynced, err := common.TagExists(lastSyncedTag)
	if err != nil {
		panic(err)
	}

	if !hasEverSynced {
		return fmt.Sprintf("no `%s` tag exists yet", lastSyncedTag)
	}

	changedFiles := append(getSyncedWorkflowsChangedSince(lastSyncedTag), getSyncedReposDefinitionChangedSince(lastSyncedTag)...)
	if len(changedFiles) == 0 {
		return ""
	}

	return fmt.Sprintf("`%s` were different since `%s`", strings.Join(changedFiles, "`, `"), lastSyncedTag)
}

func main() {
	common.SetupGitHubUser()

	tag, err := common.GetLatestVersionTag()
	if err != nil {
		panic(err)
	}

	var summaryLines []string

	if tag == "" {
		tag = "v1"
		if err := common.AddTag(tag); err != nil {
			panic(err)
		}
		summaryLines = append(summaryLines, fmt.Sprintf("### 🏷️ Tag `%s` Created", tag))
	} else if shouldIncrementTag() {
		nextMajorVersion := nextMajorVersionForTag(tag)
		nextTag := fmt.Sprintf("v%v", nextMajorVersion)
		if err := common.AddTag(nextTag); err != nil {
			panic(err)
		}
		summaryLines = append(summaryLines, fmt.Sprintf("### 🏷️ Tag `%s` Created", nextTag))
	} else {
		if err := common.MoveTag(tag); err != nil {
			panic(err)
		}
		summaryLines = append(summaryLines, fmt.Sprintf("### 🏷️ Tag `%s` Updated", tag))
	}

	reasonToSync := reasonToSyncWorkflows()
	if reasonToSync != "" {
		summaryLines = append(summaryLines, fmt.Sprintf("*Workflows need to be synchronized, because %s.*", reasonToSync))
	}

	common.WriteOutput(fmt.Sprintf("%v", reasonToSync != ""))
	common.WriteJobSummary(strings.Join(summaryLines, "\r\n"))
}
