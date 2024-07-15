package main

import (
	"fmt"

	common "github.com/workflow-sync-poc/common/code"
)

func main() {
	common.WriteJobSummary(fmt.Sprintf("### Executed Go File from `%s`", common.GetEnv("GO_FILE_REF")))
}
