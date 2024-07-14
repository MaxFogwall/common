package main

import (
	"fmt"

	common "github.com/workflow-sync-poc/common/code"
)

func main() {
	common.WriteJobSummary(fmt.Sprintf("### Executed from `%s`", common.GetEnv("GH_REF")))
}
