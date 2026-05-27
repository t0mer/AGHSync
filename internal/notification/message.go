package notification

import (
	"fmt"
	"strings"
	"time"

	"github.com/t0mer/aghsync/internal/history"
	"github.com/t0mer/aghsync/internal/instance"
)

// BuildMessage composes the notification text for a completed sync run.
// instanceNames maps instance ID → display name.
func BuildMessage(run *history.Run, results []*history.Result, instanceNames map[string]string) string {
	var sb strings.Builder

	statusEmoji := "✅"
	if run.Status == "partial_failure" {
		statusEmoji = "⚠️"
	} else if run.Status == "error" {
		statusEmoji = "❌"
	}

	sb.WriteString(fmt.Sprintf("%s AGHSync — Sync %s\n", statusEmoji, strings.ToUpper(run.Status)))
	sb.WriteString(fmt.Sprintf("Run: %s\n", run.ID[:8]))
	sb.WriteString(fmt.Sprintf("Triggered by: %s\n", run.TriggeredBy))
	sb.WriteString(fmt.Sprintf("Started: %s\n", run.StartedAt.Format(time.RFC1123)))
	if run.FinishedAt != nil {
		dur := run.FinishedAt.Sub(run.StartedAt).Round(time.Millisecond)
		sb.WriteString(fmt.Sprintf("Duration: %s\n", dur))
	}

	if len(results) == 0 {
		return sb.String()
	}

	// Group results by instance.
	type instResult struct {
		name    string
		success []string
		failed  []string
	}
	byInst := make(map[string]*instResult)
	order := make([]string, 0)
	for _, res := range results {
		ir, ok := byInst[res.InstanceID]
		if !ok {
			name := instanceNames[res.InstanceID]
			if name == "" {
				name = res.InstanceID[:8]
			}
			ir = &instResult{name: name}
			byInst[res.InstanceID] = ir
			order = append(order, res.InstanceID)
		}
		if res.Status == "success" {
			ir.success = append(ir.success, res.ConfigType)
		} else {
			ir.failed = append(ir.failed, res.ConfigType)
		}
	}

	sb.WriteString("\nResults:\n")
	for _, id := range order {
		ir := byInst[id]
		line := fmt.Sprintf("• %s: ", ir.name)
		parts := make([]string, 0)
		if len(ir.success) > 0 {
			parts = append(parts, fmt.Sprintf("✅ %s", strings.Join(ir.success, ", ")))
		}
		if len(ir.failed) > 0 {
			parts = append(parts, fmt.Sprintf("❌ %s", strings.Join(ir.failed, ", ")))
		}
		line += strings.Join(parts, "  ")
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// instanceNamesFromList builds the id→name map from a list of instances.
func instanceNamesFromList(instances []*instance.Instance) map[string]string {
	m := make(map[string]string, len(instances))
	for _, inst := range instances {
		m[inst.ID] = inst.Name
	}
	return m
}
