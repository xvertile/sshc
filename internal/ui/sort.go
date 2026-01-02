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

// sortHostsByName sorts a slice of SSH hosts alphabetically by name
func sortHostsByName(hosts []config.SSHHost) []config.SSHHost {
	sorted := make([]config.SSHHost, len(hosts))
	copy(sorted, hosts)

	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}

// filterHosts filters hosts according to the search query (name or tags)
func (m Model) filterHosts(query string) []config.SSHHost {
	subqueries := strings.Split(query, " ")
	subqueriesLength := len(subqueries)
	subfilteredHosts := make([][]config.SSHHost, subqueriesLength)
	for i, subquery := range subqueries {
		subfilteredHosts[i] = m.filterHostsByWord(subquery)
	}

	// return the intersection of search results
	result := make([]config.SSHHost, 0)
	tempMap := map[string]int{}
	for _, hosts := range subfilteredHosts {
		for _, host := range hosts {
			if _, ok := tempMap[host.Name]; !ok {
				tempMap[host.Name] = 1
			} else {
				tempMap[host.Name] = tempMap[host.Name] + 1
			}

			if tempMap[host.Name] == subqueriesLength {
				result = append(result, host)
			}
		}
	}

	return result
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

// filterHostsByWord filters hosts according to a single word
func (m Model) filterHostsByWord(word string) []config.SSHHost {
	var filtered []config.SSHHost

	if word == "" {
		filtered = m.hosts
	} else {
		word = strings.ToLower(word)

		for _, host := range m.hosts {
			// Check the hostname
			if strings.Contains(strings.ToLower(host.Name), word) {
				filtered = append(filtered, host)
				continue
			}

			// Check the hostname
			if strings.Contains(strings.ToLower(host.Hostname), word) {
				filtered = append(filtered, host)
				continue
			}

			// Check the user
			if strings.Contains(strings.ToLower(host.User), word) {
				filtered = append(filtered, host)
				continue
			}

			// Check the tags
			for _, tag := range host.Tags {
				if strings.Contains(strings.ToLower(tag), word) {
					filtered = append(filtered, host)
					break
				}
			}
		}
	}

	return m.sortHosts(filtered)
}
