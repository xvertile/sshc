package ui

import (
	"sort"
	"strings"

	"github.com/xvertile/sshc/internal/config"
)

// sortHosts sorts hosts according to the current sort mode
func (m Model) sortHosts(hosts []config.SSHHost) []config.SSHHost {
	if m.historyManager == nil {
		return sortHostsByName(hosts)
	}

	switch m.sortMode {
	case SortByLastUsed:
		return m.historyManager.SortHostsByLastUsed(hosts)
	case SortByName:
		fallthrough
	default:
		return sortHostsByName(hosts)
	}
}

// sortEntries sorts unified entries according to the current sort mode
func (m Model) sortEntries(entries []HostEntry) []HostEntry {
	sorted := make([]HostEntry, len(entries))
	copy(sorted, entries)

	switch m.sortMode {
	case SortByLastUsed:
		if m.historyManager != nil {
			sort.SliceStable(sorted, func(i, j int) bool {
				timeI, existsI := m.historyManager.GetLastConnectionTime(sorted[i].Name)
				timeJ, existsJ := m.historyManager.GetLastConnectionTime(sorted[j].Name)

				// Hosts with history come first
				if existsI && !existsJ {
					return true
				}
				if !existsI && existsJ {
					return false
				}
				if existsI && existsJ {
					return timeI.After(timeJ)
				}
				// Both without history: sort alphabetically
				return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
			})
		}
	case SortByName:
		fallthrough
	default:
		sort.Slice(sorted, func(i, j int) bool {
			return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
		})
	}

	return sorted
}

// sortHostsByName sorts a slice of SSH hosts alphabetically by name
func sortHostsByName(hosts []config.SSHHost) []config.SSHHost {
	sorted := make([]config.SSHHost, len(hosts))
	copy(sorted, hosts)

	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}

// filterEntries filters unified entries (SSH + K8s) according to the search query
func (m Model) filterEntries(query string) []HostEntry {
	if query == "" {
		return m.allEntries
	}

	query = strings.ToLower(query)
	words := strings.Fields(query)

	var filtered []HostEntry
	for _, entry := range m.allEntries {
		matchesAll := true
		for _, word := range words {
			if !entryMatchesWord(entry, word) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			filtered = append(filtered, entry)
		}
	}

	return filtered
}

// entryMatchesWord checks if a HostEntry matches a single search word
func entryMatchesWord(entry HostEntry, word string) bool {
	// Check name
	if strings.Contains(strings.ToLower(entry.Name), word) {
		return true
	}
	// Check hostname
	if strings.Contains(strings.ToLower(entry.Hostname), word) {
		return true
	}
	// Check user (from underlying SSH host if available)
	if entry.SSHHost != nil && strings.Contains(strings.ToLower(entry.SSHHost.User), word) {
		return true
	}
	// Check tags
	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), word) {
			return true
		}
	}
	return false
}
