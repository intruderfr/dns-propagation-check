package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// renderTable prints a human-readable table and a final verdict.
// Returns the process exit code.
func renderTable(w io.Writer, opts Options, results []Result) int {
	fmt.Fprintf(w, "Query: %s (%s)\n\n", opts.Domain, opts.RecordType)
	fmt.Fprintf(w, "%-18s %-18s %-40s %-10s %s\n", "RESOLVER", "PROVIDER", "ANSWER", "LATENCY", "STATUS")
	fmt.Fprintln(w, strings.Repeat("-", 100))

	okCount := 0
	errCount := 0
	for _, r := range results {
		ans := strings.Join(r.Answers, ", ")
		if len(ans) > 38 {
			ans = ans[:35] + "..."
		}
		status := "OK"
		if r.Err != nil {
			status = "ERROR: " + shortErr(r.Err.Error())
			if ans == "" {
				ans = "-"
			}
			errCount++
		} else {
			okCount++
		}
		latStr := r.Latency.Round(time.Millisecond).String()
		fmt.Fprintf(w, "%-18s %-18s %-40s %-10s %s\n",
			r.Resolver.IP, r.Resolver.Provider, ans, latStr, status)
	}

	fmt.Fprintln(w)
	if okCount == 0 {
		fmt.Fprintln(w, "Result: ERROR — every resolver failed. Check network, domain, or record type.")
		return 1
	}

	if isConsistent(results) {
		fmt.Fprintf(w, "Result: CONSISTENT — all %d successful resolvers returned the same answer.\n", okCount)
		if errCount > 0 {
			fmt.Fprintf(w, "Note: %d resolvers errored (see table above).\n", errCount)
		}
		return 0
	}

	groups := groupByAnswer(results)
	fmt.Fprintf(w, "Result: INCONSISTENT — %d distinct answer sets observed.\n\n", len(groups))

	// Sort groups by size descending so the dominant answer appears first.
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(groups[keys[i]]) > len(groups[keys[j]])
	})
	for i, k := range keys {
		label := string(rune('A' + i))
		providers := make([]string, 0, len(groups[k]))
		for _, r := range groups[k] {
			providers = append(providers, r.Resolver.Provider)
		}
		ans := strings.ReplaceAll(k, "|", ", ")
		fmt.Fprintf(w, "  Group %s (%d resolvers — %s): %s\n",
			label, len(groups[k]), strings.Join(providers, ", "), ans)
	}
	fmt.Fprintln(w, "\nHint: incomplete propagation usually resolves itself. Re-run with --watch to wait for convergence.")
	return 2
}

// renderJSON emits a machine-readable JSON object; exit code matches table mode.
func renderJSON(w io.Writer, opts Options, results []Result) int {
	type answerRow struct {
		Resolver  string   `json:"resolver"`
		Provider  string   `json:"provider"`
		Answers   []string `json:"answers"`
		LatencyMS int64    `json:"latency_ms"`
		Error     string   `json:"error,omitempty"`
	}
	type output struct {
		Domain     string      `json:"domain"`
		Type       string      `json:"type"`
		Consistent bool        `json:"consistent"`
		Groups     int         `json:"answer_groups"`
		Results    []answerRow `json:"results"`
	}

	rows := make([]answerRow, 0, len(results))
	okCount := 0
	for _, r := range results {
		row := answerRow{
			Resolver:  r.Resolver.IP,
			Provider:  r.Resolver.Provider,
			Answers:   r.Answers,
			LatencyMS: r.Latency.Milliseconds(),
		}
		if r.Err != nil {
			row.Error = r.Err.Error()
		} else {
			okCount++
		}
		rows = append(rows, row)
	}
	consistent := isConsistent(results)
	out := output{
		Domain:     opts.Domain,
		Type:       opts.RecordType,
		Consistent: consistent,
		Groups:     len(groupByAnswer(results)),
		Results:    rows,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)

	if okCount == 0 {
		return 1
	}
	if consistent {
		return 0
	}
	return 2
}

// consistencyLabel produces a compact status line for watch-mode logging.
func consistencyLabel(consistent bool, results []Result) string {
	if consistent {
		return "CONSISTENT"
	}
	groups := groupByAnswer(results)
	return fmt.Sprintf("INCONSISTENT (%d distinct answer sets)", len(groups))
}

func shortErr(s string) string {
	if len(s) > 30 {
		return s[:27] + "..."
	}
	return s
}
