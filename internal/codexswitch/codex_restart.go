package codexswitch

import "fmt"

func (s *Service) RestartCodex() error {
	if s == nil || s.restartCodex == nil {
		return fmt.Errorf("Codex 重启器未初始化")
	}
	return s.restartCodex()
}
