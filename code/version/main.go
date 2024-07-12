package main

import (
	"fmt"
	"regexp"
	"strconv"

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

	return majorVersion
}

func isAnySyncedWorkflowChanged() bool {
	isNoSyncedWorkflowChanged, err := common.IsLastCommitClean(".github/workflows/synced_*")
	if err != nil {
		panic(err)
	}

	return !isNoSyncedWorkflowChanged
}

func isSyncedReposListChanged() bool {
	isSyncedReposListSame, err := common.IsLastCommitClean("repos.json")
	if err != nil {
		panic(err)
	}

	return isSyncedReposListSame
}

func shouldIncrementTag() bool {
	return isAnySyncedWorkflowChanged()
}

func shouldSyncWorkflows() bool {
	return isAnySyncedWorkflowChanged() || isSyncedReposListChanged()
}

func main() {
	tag, err := common.GetLatestTag()
	if err != nil {
		panic(err)
	}

	if tag == "" {
		tag = "v1"
		common.AddTag(tag)
		common.WriteJobSummary(fmt.Sprintf("### 🏷️ Tag `%s` Created", tag))
	} else if shouldIncrementTag() {
		nextMajorVersion := nextMajorVersionForTag(tag)
		nextTag := fmt.Sprintf("v%v", nextMajorVersion)
		common.AddTag(nextTag)
		common.WriteJobSummary(fmt.Sprintf("### 🏷️ Tag `%s` Created", nextTag))
	} else {
		common.MoveTag(tag)
		common.WriteJobSummary(fmt.Sprintf("### 🏷️ Tag `%s` Updated", tag))
	}

	common.WriteOutput("should-sync-workflows", fmt.Sprintf("%v", shouldSyncWorkflows()))
}
