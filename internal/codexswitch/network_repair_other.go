//go:build !windows

package codexswitch

import "fmt"

func repairNetwork() (NetworkRepairResult, error) {
	return NetworkRepairResult{}, fmt.Errorf("网络修复目前仅支持 Windows")
}
