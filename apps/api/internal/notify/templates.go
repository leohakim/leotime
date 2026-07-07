package notify

import (
	"fmt"
	"strings"
	"time"

	"github.com/leotime/leotime/apps/api/internal/store"
)

func stillRunningSubject(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "Your time tracker is still running"
	}
	return "Tu cronómetro sigue activo"
}

func stillRunningBody(candidate store.StillRunningCandidate, publicBaseURL string, now time.Time) string {
	startedAt, err := time.Parse(time.RFC3339Nano, candidate.StartedAt)
	if err != nil {
		startedAt, _ = time.Parse(time.RFC3339, candidate.StartedAt)
	}

	elapsed := now.Sub(startedAt)
	if elapsed < 0 {
		elapsed = 0
	}

	dashboardURL := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if dashboardURL == "" {
		dashboardURL = "http://127.0.0.1:8080"
	}

	if strings.EqualFold(strings.TrimSpace(candidate.Locale), "en") {
		return fmt.Sprintf(
			"Hi %s,\n\nYour timer is still running on the same task.\n\nProject: %s\nTask: %s\nDescription: %s\nStarted: %s\nElapsed: %s\n\nOpen leotime: %s\n\nYou can change this reminder in Settings.",
			displayName(candidate.UserName, candidate.UserEmail),
			displayValue(candidate.ProjectName, "—"),
			displayValue(candidate.TaskName, "—"),
			displayValue(candidate.Description, "—"),
			formatTimestamp(startedAt),
			formatDuration(elapsed),
			dashboardURL,
		)
	}

	return fmt.Sprintf(
		"Hola %s,\n\nTu cronómetro sigue activo en la misma tarea.\n\nProyecto: %s\nTarea: %s\nDescripción: %s\nInicio: %s\nTranscurrido: %s\n\nAbrir leotime: %s\n\nPuedes cambiar este aviso en Ajustes.",
		displayName(candidate.UserName, candidate.UserEmail),
		displayValue(candidate.ProjectName, "—"),
		displayValue(candidate.TaskName, "—"),
		displayValue(candidate.Description, "—"),
		formatTimestamp(startedAt),
		formatDuration(elapsed),
		dashboardURL,
	)
}

func displayName(name string, email string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(email)
}

func displayValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.UTC().Format(time.RFC3339)
}

func formatDuration(value time.Duration) string {
	hours := int(value.Hours())
	minutes := int(value.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
