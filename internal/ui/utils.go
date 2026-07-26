package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/xvertile/sshc/internal/config"
	"github.com/xvertile/sshc/internal/connectivity"
)

// saveSortMode persists the current sort mode to config
func (m *Model) saveSortMode() {
	if m.appConfig == nil {
		return
	}

	sortModeStr := "name"
	if m.sortMode == SortByLastUsed {
		sortModeStr = "recent"
	}

	m.appConfig.SortMode = sortModeStr
	config.SaveAppConfig(m.appConfig)
}

// formatTimeAgo formats a time into a readable "X time ago" string
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	switch {
	case duration < time.Minute:
		seconds := int(duration.Seconds())
		if seconds <= 1 {
			return "1 second ago"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case duration < 365*24*time.Hour:
		months := int(duration.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(duration.Hours() / (24 * 365))
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// formatTimeCompact formats a time as a short age such as "5m", "2h" or "3mo",
// for use in table columns where "3 months ago" would cost 12 columns.
// Returns an em dash when the time is zero.
func formatTimeCompact(t time.Time) string {
	if t.IsZero() {
		return "—"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "now"
	case duration < time.Hour:
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh", int(duration.Hours()))
	case duration < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	case duration < 30*24*time.Hour:
		return fmt.Sprintf("%dw", int(duration.Hours()/(24*7)))
	case duration < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(duration.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(duration.Hours()/(24*365)))
	}
}

// formatConfigFile formats a config file path for display
func formatConfigFile(filePath string) string {
	if filePath == "" {
		return "Unknown"
	}
	// Show just the filename and parent directory for readability
	parts := strings.Split(filePath, "/")
	if len(parts) >= 2 {
		return fmt.Sprintf(".../%s/%s", parts[len(parts)-2], parts[len(parts)-1])
	}
	return filePath
}

// getPingStatusIndicator returns a status indicator based on ping status.
//
// The glyphs are shape-distinct as well as colour-distinct, so status still
// reads on the selected row, where the selection bar owns the colours.
func (m *Model) getPingStatusIndicator(hostName string) string {
	if m.pingManager == nil {
		return "○" // Empty circle for unknown
	}

	switch m.pingManager.GetStatus(hostName) {
	case connectivity.StatusOnline:
		return "●" // Filled circle for online
	case connectivity.StatusOffline:
		return "×" // X for offline
	case connectivity.StatusConnecting:
		return "◌" // Dotted circle for connecting
	default:
		return "○" // Empty circle for unknown
	}
}

// statusColor returns the colour for a host's connectivity glyph.
func (m *Model) statusColor(hostName string) string {
	theme := GetCurrentTheme()

	if m.pingManager == nil {
		return theme.Muted
	}

	switch m.pingManager.GetStatus(hostName) {
	case connectivity.StatusOnline:
		return theme.Success
	case connectivity.StatusOffline:
		return theme.Error
	case connectivity.StatusConnecting:
		return theme.Accent
	default:
		return theme.Muted
	}
}

// rebuildTagColors assigns every distinct tag its own position on the colour
// wheel.
//
// Colours are handed out by position in the sorted tag list rather than by
// hashing the name. A hash into a fixed palette gives two tags the same colour
// as soon as there are more tags than slots, which defeats the point of
// colouring them at all; positions are unique by construction.
//
// Sorting keeps the assignment stable for a given set of tags, so a colour
// only moves when tags are added or removed.
func (m *Model) rebuildTagColors() {
	var tags []string

	seen := make(map[string]bool)
	for i := range m.allEntries {
		for _, tag := range m.allEntries[i].Tags {
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}

	slices.Sort(tags)

	m.tagPositions = make(map[string]int, len(tags))
	for i, tag := range tags {
		m.tagPositions[tag] = i
	}
}

// tagColor returns the colour assigned to a tag, spacing the tags evenly
// through the theme's own anchor colours.
func (m *Model) tagColor(tag string) string {
	total := len(m.tagPositions)
	if total == 0 {
		return GetCurrentTheme().Primary
	}

	return GetCurrentTheme().blend(float64(m.tagPositions[tag]) / float64(total))
}

// recencyColor fades a last-login timestamp with age, so how recently a host
// was used is legible without reading the number.
func recencyColor(t time.Time, exists bool) string {
	theme := GetCurrentTheme()

	if !exists {
		return theme.Secondary
	}

	switch age := time.Since(t); {
	case age < 24*time.Hour:
		return theme.Foreground
	case age < 7*24*time.Hour:
		return theme.Muted
	default:
		return theme.Secondary
	}
}

// selectedEntry returns the entry under the table cursor, or nil when the list
// is empty.
//
// Selection is resolved by row index against filteredEntries rather than by
// parsing the rendered row text. The previous approach split the first column
// on whitespace, so a host named "k web" was read as a Kubernetes host named
// "web" and was dispatched to kubectl instead of ssh.
func (m *Model) selectedEntry() *HostEntry {
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.filteredEntries) {
		return nil
	}
	return &m.filteredEntries[cursor]
}

// getHostEntryByName finds a host entry by name from the filtered entries
func (m *Model) getHostEntryByName(name string) *HostEntry {
	for i := range m.filteredEntries {
		if m.filteredEntries[i].Name == name {
			return &m.filteredEntries[i]
		}
	}
	return nil
}

// rebuildEntries rebuilds the unified host entries and re-applies the active
// search.
//
// Entries are built from the complete host sets, not the filtered ones: when
// they were built from filteredHosts, deleting a host while a search was
// active shrank allEntries to the matching subset, so clearing the search
// afterwards showed only those matches until the app was restarted.
func (m *Model) rebuildEntries() {
	var allEntries []HostEntry

	// Add SSH hosts
	for i := range m.hosts {
		host := &m.hosts[i]
		allEntries = append(allEntries, HostEntry{
			Name:     host.Name,
			IsK8s:    false,
			SSHHost:  host,
			Tags:     host.Tags,
			Hostname: host.Hostname,
		})
	}

	// Add K8s hosts
	for i := range m.k8sHosts {
		host := &m.k8sHosts[i]
		allEntries = append(allEntries, HostEntry{
			Name:     host.Name,
			IsK8s:    true,
			K8sHost:  host,
			Tags:     host.Tags,
			Hostname: fmt.Sprintf("%s/%s", host.Namespace, host.Pod),
		})
	}

	m.allEntries = m.sortEntries(allEntries)
	m.rebuildTagColors()
	m.applyFilter()
}

// applyFilter re-applies the active search to the entry list under the current
// sort mode.
func (m *Model) applyFilter() {
	if query := m.searchInput.Value(); query != "" {
		m.filteredEntries = m.sortEntries(m.filterEntries(query))
		return
	}
	m.filteredEntries = m.sortEntries(m.allEntries)
}
