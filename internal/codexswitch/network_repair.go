package codexswitch

type NetworkRepairResult struct {
	Success            bool     `json:"success"`
	Message            string   `json:"message"`
	PhysicalInterfaces []string `json:"physicalInterfaces,omitempty"`
	VPNInterfaces      []string `json:"vpnInterfaces,omitempty"`
	APIDNS             []string `json:"apiDns,omitempty"`
	ChatGPTDNS         []string `json:"chatgptDns,omitempty"`
	APIReachable       bool     `json:"apiReachable"`
	ChatGPTReachable   bool     `json:"chatgptReachable"`
	Error              string   `json:"error,omitempty"`
}

func (s *Service) RepairNetwork() (NetworkRepairResult, error) {
	return repairNetwork()
}
