package enrich

import (
	"fmt"
	"strings"
)

type dailySummaryGroup struct {
	bullets []string
	heading string
}

type dailySummarySkeleton struct {
	closing    string
	dateLine   string
	groups     []dailySummaryGroup
	header     string
	standalone []string
}

func dailySummaryBaseText(bundle ContextBundle) string {
	if text := strings.TrimSpace(bundle.TemplateText); text != "" {
		return text
	}
	return strings.TrimSpace(bundle.CurrentDraft)
}

func enforceDailySummaryStructure(template, enriched string) string {
	template = strings.TrimSpace(template)
	enriched = strings.TrimSpace(enriched)
	if template == "" {
		return enriched
	}
	if enriched == "" {
		return template
	}

	base, err := parseDailySummarySkeleton(template)
	if err != nil {
		return enriched
	}

	candidate, candidateErr := parseDailySummarySkeleton(enriched)
	if candidateErr == nil && dailySummarySkeletonCompatible(base, candidate) {
		return renderDailySummarySkeleton(candidate)
	}

	childBullets := extractDailySummaryChildBullets(enriched)
	standalone := extractDailySummaryStandaloneBullets(enriched)
	expected := base.totalChildBullets()

	if len(childBullets) >= expected && expected > 0 {
		applyDailySummaryChildBullets(&base, childBullets[:expected])
	} else if len(childBullets) > 0 && len(childBullets) < expected {
		applyDailySummaryChildBulletsPartial(&base, childBullets)
	}

	if len(standalone) > 0 {
		base.standalone = standalone
	}

	return renderDailySummarySkeleton(base)
}

func parseDailySummarySkeleton(text string) (dailySummarySkeleton, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) < 2 {
		return dailySummarySkeleton{}, fmt.Errorf("summary too short")
	}

	skeleton := dailySummarySkeleton{
		dateLine: strings.TrimSpace(lines[0]),
		header:   strings.TrimSpace(lines[1]),
	}

	var current *dailySummaryGroup
	for i := 2; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		if isDailySummaryClosingLine(trimmed) {
			skeleton.closing = trimmed
			current = nil
			continue
		}
		if heading, ok := dailySummaryGroupHeading(trimmed); ok {
			skeleton.groups = append(skeleton.groups, dailySummaryGroup{heading: heading})
			current = &skeleton.groups[len(skeleton.groups)-1]
			continue
		}
		if bullet, ok := dailySummaryChildBullet(line); ok {
			if current == nil {
				return dailySummarySkeleton{}, fmt.Errorf("child bullet without group")
			}
			current.bullets = append(current.bullets, bullet)
			continue
		}
		if bullet, ok := dailySummaryStandaloneBullet(trimmed); ok {
			current = nil
			skeleton.standalone = append(skeleton.standalone, bullet)
			continue
		}
	}

	if skeleton.dateLine == "" || skeleton.header == "" {
		return dailySummarySkeleton{}, fmt.Errorf("missing header")
	}
	return skeleton, nil
}

func dailySummaryGroupHeading(line string) (string, bool) {
	if !strings.HasPrefix(line, "- ") {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "- "))
	if !strings.HasSuffix(rest, ":") {
		return "", false
	}
	heading := strings.TrimSpace(strings.TrimSuffix(rest, ":"))
	if heading == "" {
		return "", false
	}
	return heading, true
}

func dailySummaryChildBullet(line string) (string, bool) {
	if !strings.HasPrefix(line, "    - ") {
		return "", false
	}
	text := strings.TrimSpace(strings.TrimPrefix(line, "    - "))
	if text == "" {
		return "", false
	}
	return text, true
}

func dailySummaryStandaloneBullet(line string) (string, bool) {
	if strings.HasPrefix(line, "    ") {
		return "", false
	}
	if !strings.HasPrefix(line, "- ") {
		return "", false
	}
	if strings.HasSuffix(line, ":") {
		return "", false
	}
	text := strings.TrimSpace(strings.TrimPrefix(line, "- "))
	if text == "" {
		return "", false
	}
	return text, true
}

func isDailySummaryClosingLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	return strings.HasPrefix(lower, "hasta ") || strings.HasPrefix(lower, "see you")
}

func (s dailySummarySkeleton) totalChildBullets() int {
	total := 0
	for _, group := range s.groups {
		total += len(group.bullets)
	}
	return total
}

func dailySummarySkeletonCompatible(base, candidate dailySummarySkeleton) bool {
	if base.dateLine != candidate.dateLine || base.header != candidate.header {
		return false
	}
	if len(base.groups) != len(candidate.groups) {
		return false
	}
	for i := range base.groups {
		if base.groups[i].heading != candidate.groups[i].heading {
			return false
		}
		if len(base.groups[i].bullets) != len(candidate.groups[i].bullets) {
			return false
		}
	}
	if len(base.standalone) != len(candidate.standalone) {
		return false
	}
	for i := range base.standalone {
		if base.standalone[i] != candidate.standalone[i] {
			return false
		}
	}
	if base.closing != "" && candidate.closing == "" {
		return false
	}
	if base.closing != "" && candidate.closing != "" && base.closing != candidate.closing {
		return false
	}
	return candidate.totalChildBullets() > 0 || len(candidate.standalone) > 0 || candidate.closing != ""
}

func extractDailySummaryChildBullets(text string) []string {
	bullets := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		if bullet, ok := dailySummaryChildBullet(line); ok {
			bullets = append(bullets, bullet)
		}
	}
	return bullets
}

func extractDailySummaryStandaloneBullets(text string) []string {
	bullets := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isDailySummaryClosingLine(trimmed) {
			continue
		}
		if _, ok := dailySummaryChildBullet(line); ok {
			continue
		}
		if _, ok := dailySummaryGroupHeading(trimmed); ok {
			continue
		}
		if bullet, ok := dailySummaryStandaloneBullet(trimmed); ok {
			bullets = append(bullets, bullet)
		}
	}
	return bullets
}

func applyDailySummaryChildBullets(skeleton *dailySummarySkeleton, bullets []string) {
	index := 0
	for groupIdx := range skeleton.groups {
		for bulletIdx := range skeleton.groups[groupIdx].bullets {
			if index >= len(bullets) {
				return
			}
			skeleton.groups[groupIdx].bullets[bulletIdx] = bullets[index]
			index++
		}
	}
}

func applyDailySummaryChildBulletsPartial(skeleton *dailySummarySkeleton, bullets []string) {
	index := 0
	for groupIdx := range skeleton.groups {
		for bulletIdx := range skeleton.groups[groupIdx].bullets {
			if index >= len(bullets) {
				return
			}
			skeleton.groups[groupIdx].bullets[bulletIdx] = bullets[index]
			index++
		}
	}
}

func renderDailySummarySkeleton(skeleton dailySummarySkeleton) string {
	lines := []string{skeleton.dateLine, skeleton.header}
	for _, group := range skeleton.groups {
		lines = append(lines, "- "+group.heading+":")
		for _, bullet := range group.bullets {
			lines = append(lines, "    - "+bullet)
		}
	}
	for _, bullet := range skeleton.standalone {
		lines = append(lines, "- "+bullet)
	}
	if skeleton.closing != "" {
		lines = append(lines, skeleton.closing)
	}
	return strings.Join(lines, "\n")
}
