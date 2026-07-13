package enrich

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ProjectWorkspace struct {
	ProjectID           string `json:"projectId"`
	ProjectName         string `json:"projectName"`
	LocalRepoPath       string `json:"localRepoPath"`
	CursorWorkspaceSlug string `json:"cursorWorkspaceSlug"`
}

type CommitLine struct {
	ProjectName string `json:"projectName"`
	Hash        string `json:"hash"`
	Subject     string `json:"subject"`
}

type CursorActivity struct {
	WorkspaceSlug string   `json:"workspaceSlug"`
	UserQueries   []string `json:"userQueries"`
	FilesTouched  []string `json:"filesTouched"`
}

type TimeEntryFact struct {
	ClientName  string   `json:"clientName"`
	ProjectName string   `json:"projectName"`
	TaskName    string   `json:"taskName"`
	Topics      []string `json:"topics"`
	Description string   `json:"description"`
}

type ContextBundle struct {
	Date           string           `json:"date"`
	TemplateText   string           `json:"templateText"`
	ManualNote     string           `json:"manualNote"`
	Feedback       string           `json:"feedback"`
	CurrentDraft   string           `json:"currentDraft"`
	EntryFacts     []TimeEntryFact  `json:"entryFacts,omitempty"`
	Commits        []CommitLine     `json:"commits"`
	CursorActivity []CursorActivity `json:"cursorActivity"`
	Locale         string           `json:"locale"`
}

func CollectGitCommits(date string, authorEmail string, projects []ProjectWorkspace) ([]CommitLine, error) {
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	since := day.Format("2006-01-02") + " 00:00:00"
	until := day.Add(24*time.Hour).Format("2006-01-02") + " 00:00:00"

	lines := make([]CommitLine, 0)
	seen := map[string]struct{}{}
	for _, project := range projects {
		repoPath := strings.TrimSpace(project.LocalRepoPath)
		if repoPath == "" {
			continue
		}
		args := []string{
			"-C", repoPath, "log",
			fmt.Sprintf("--since=%s", since),
			fmt.Sprintf("--until=%s", until),
			"--pretty=format:%h\t%s",
		}
		if strings.TrimSpace(authorEmail) != "" {
			args = append(args, "--author="+strings.TrimSpace(authorEmail))
		}
		output, err := exec.Command("git", args...).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) != 2 {
				continue
			}
			key := project.ProjectName + ":" + parts[0]
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			lines = append(lines, CommitLine{
				ProjectName: project.ProjectName,
				Hash:        parts[0],
				Subject:     parts[1],
			})
		}
	}
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].ProjectName < lines[j].ProjectName
	})
	return lines, nil
}

func CollectCursorActivity(date string, projects []ProjectWorkspace) ([]CursorActivity, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	projectsRoot := filepath.Join(home, ".cursor", "projects")
	dayStart, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	slugSet := map[string]struct{}{}
	for _, project := range projects {
		slug := strings.TrimSpace(project.CursorWorkspaceSlug)
		if slug != "" {
			slugSet[slug] = struct{}{}
		}
	}

	activities := make([]CursorActivity, 0)
	for slug := range slugSet {
		transcriptDir := filepath.Join(projectsRoot, slug, "agent-transcripts")
		entries, err := os.ReadDir(transcriptDir)
		if err != nil {
			continue
		}
		activity := CursorActivity{WorkspaceSlug: slug}
		querySeen := map[string]struct{}{}
		fileSeen := map[string]struct{}{}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			mod := info.ModTime()
			if mod.Before(dayStart) || !mod.Before(dayEnd) {
				continue
			}
			parseTranscriptFile(filepath.Join(transcriptDir, entry.Name()), &activity, querySeen, fileSeen)
			if len(activity.UserQueries) >= 8 && len(activity.FilesTouched) >= 12 {
				break
			}
		}
		if len(activity.UserQueries) > 0 || len(activity.FilesTouched) > 0 {
			activities = append(activities, activity)
		}
	}
	return activities, nil
}

type transcriptEvent struct {
	Role    string `json:"role"`
	Message struct {
		Content []struct {
			Type  string         `json:"type"`
			Text  string         `json:"text"`
			Name  string         `json:"name"`
			Input map[string]any `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

func parseTranscriptFile(path string, activity *CursorActivity, querySeen, fileSeen map[string]struct{}) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var event transcriptEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.Role == "user" {
			for _, block := range event.Message.Content {
				if block.Type != "text" {
					continue
				}
				query := extractUserQuery(block.Text)
				if query == "" {
					continue
				}
				if _, ok := querySeen[query]; ok {
					continue
				}
				querySeen[query] = struct{}{}
				activity.UserQueries = append(activity.UserQueries, query)
			}
		}
		for _, block := range event.Message.Content {
			if block.Type != "tool_use" {
				continue
			}
			if block.Name != "Read" && block.Name != "Grep" && block.Name != "Write" && block.Name != "StrReplace" {
				continue
			}
			pathValue, ok := block.Input["path"].(string)
			if !ok {
				continue
			}
			short := filepath.Base(pathValue)
			if short == "" {
				continue
			}
			if _, ok := fileSeen[short]; ok {
				continue
			}
			fileSeen[short] = struct{}{}
			activity.FilesTouched = append(activity.FilesTouched, short)
		}
	}
}

func extractUserQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, "<user_query>"); idx >= 0 {
		rest := raw[idx+len("<user_query>"):]
		if end := strings.Index(rest, "</user_query>"); end >= 0 {
			return strings.TrimSpace(rest[:end])
		}
	}
	if len(raw) > 180 {
		return raw[:180] + "…"
	}
	return raw
}
