package codexswitch

import "strings"

var managedTopLevelConfigKeys = map[string]bool{
	"model_provider":                 true,
	"model":                          true,
	"review_model":                   true,
	"model_reasoning_effort":         true,
	"disable_response_storage":       true,
	"network_access":                 true,
	"windows_wsl_setup_acknowledged": true,
	"model_context_window":           true,
	"model_auto_compact_token_limit": true,
}

var managedSectionConfigKeys = map[string]map[string]bool{
	"model_providers.OpenAI": {
		"name":                 true,
		"base_url":             true,
		"wire_api":             true,
		"requires_openai_auth": true,
	},
}

type managedConfigBlock struct {
	topLevel      map[string]string
	topLevelOrder []string
	sections      map[string]map[string]string
	sectionOrder  []string
	sectionKeys   map[string][]string
}

func mergeManagedConfigRaw(currentRaw, targetRaw string) string {
	target := parseManagedConfigBlock(targetRaw)
	currentRaw = normalizeConfigRaw(currentRaw)

	var output []string
	currentSection := ""
	topLevelAppended := false
	sectionPresent := map[string]bool{}
	seenTopLevel := map[string]bool{}
	seenSectionKeys := map[string]map[string]bool{}

	appendLine := func(line string) {
		output = append(output, line)
	}
	appendBlankIfNeeded := func() {
		if len(output) > 0 && strings.TrimSpace(output[len(output)-1]) != "" {
			output = append(output, "")
		}
	}
	appendMissingTopLevel := func() {
		if topLevelAppended {
			return
		}
		for _, key := range target.topLevelOrder {
			if seenTopLevel[key] {
				continue
			}
			appendLine(target.topLevel[key])
			seenTopLevel[key] = true
		}
		topLevelAppended = true
	}
	appendMissingSectionKeys := func(section string) {
		keys := target.sectionKeys[section]
		if len(keys) == 0 {
			return
		}
		if seenSectionKeys[section] == nil {
			seenSectionKeys[section] = map[string]bool{}
		}
		for _, key := range keys {
			if seenSectionKeys[section][key] {
				continue
			}
			appendLine(target.sections[section][key])
			seenSectionKeys[section][key] = true
		}
	}

	for _, line := range strings.Split(currentRaw, "\n") {
		if strings.TrimSpace(line) == "" {
			appendLine(line)
			continue
		}

		if section, ok := parseTOMLSectionHeader(line); ok {
			if currentSection == "" {
				appendMissingTopLevel()
			} else {
				appendMissingSectionKeys(currentSection)
			}
			currentSection = section
			sectionPresent[section] = true
			appendLine(line)
			continue
		}

		key, _, ok := splitTOMLAssignment(strings.TrimSpace(stripTOMLComment(line)))
		if !ok {
			appendLine(line)
			continue
		}

		switch {
		case currentSection == "" && managedTopLevelConfigKeys[key]:
			if targetLine, ok := target.topLevel[key]; ok && !seenTopLevel[key] {
				appendLine(targetLine)
			}
			seenTopLevel[key] = true
		case isManagedSectionKey(currentSection, key):
			if seenSectionKeys[currentSection] == nil {
				seenSectionKeys[currentSection] = map[string]bool{}
			}
			if targetLine, ok := target.sections[currentSection][key]; ok && !seenSectionKeys[currentSection][key] {
				appendLine(targetLine)
			}
			seenSectionKeys[currentSection][key] = true
		default:
			appendLine(line)
		}
	}

	if currentSection == "" {
		appendMissingTopLevel()
	} else {
		appendMissingSectionKeys(currentSection)
	}

	for _, section := range target.sectionOrder {
		if sectionPresent[section] {
			continue
		}
		keys := target.sectionKeys[section]
		if len(keys) == 0 {
			continue
		}
		appendBlankIfNeeded()
		appendLine("[" + section + "]")
		for _, key := range keys {
			appendLine(target.sections[section][key])
		}
	}

	return normalizeConfigRaw(strings.Join(trimTrailingBlankLines(output), "\n"))
}

func parseManagedConfigBlock(raw string) managedConfigBlock {
	block := managedConfigBlock{
		topLevel:    map[string]string{},
		sections:    map[string]map[string]string{},
		sectionKeys: map[string][]string{},
	}

	currentSection := ""
	for _, line := range strings.Split(normalizeConfigRaw(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if section, ok := parseTOMLSectionHeader(line); ok {
			currentSection = section
			continue
		}

		key, _, ok := splitTOMLAssignment(strings.TrimSpace(stripTOMLComment(line)))
		if !ok {
			continue
		}
		switch {
		case currentSection == "" && managedTopLevelConfigKeys[key]:
			if _, exists := block.topLevel[key]; !exists {
				block.topLevelOrder = append(block.topLevelOrder, key)
			}
			block.topLevel[key] = strings.TrimSpace(line)
		case isManagedSectionKey(currentSection, key):
			if block.sections[currentSection] == nil {
				block.sections[currentSection] = map[string]string{}
				block.sectionOrder = append(block.sectionOrder, currentSection)
			}
			if _, exists := block.sections[currentSection][key]; !exists {
				block.sectionKeys[currentSection] = append(block.sectionKeys[currentSection], key)
			}
			block.sections[currentSection][key] = strings.TrimSpace(line)
		}
	}
	return block
}

func parseTOMLSectionHeader(line string) (string, bool) {
	line = strings.TrimSpace(stripTOMLComment(line))
	if len(line) < 3 || !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		return "", false
	}
	if strings.HasPrefix(line, "[[") || strings.HasSuffix(line, "]]") {
		return "", false
	}
	section := strings.TrimSpace(line[1 : len(line)-1])
	return section, section != ""
}

func isManagedSectionKey(section, key string) bool {
	keys := managedSectionConfigKeys[section]
	return keys != nil && keys[key]
}

func trimTrailingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
