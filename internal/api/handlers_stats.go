package api

import (
	"database/sql"
	"math"
	"net/http"
	"sort"
	"time"
)

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	// ── 1. Global uptime (24h, 7d, 30d) ──────────────────────
	type uptimeWindow struct {
		Label   string  `json:"label"`
		Percent float64 `json:"percent"`
		Window  string  `json:"window"`
	}

	windows := []struct {
		label string
		since time.Time
		win   string
	}{
		{"24h", now.Add(-24 * time.Hour), "24h"},
		{"7d", now.Add(-7 * 24 * time.Hour), "7d"},
		{"30d", now.Add(-30 * 24 * time.Hour), "30d"},
	}

	var globalUptime []uptimeWindow
	for _, w := range windows {
		var total, up int
		err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0)
			 FROM check_results WHERE checked_at >= ?`, w.since).Scan(&total, &up)
		pct := 100.0
		if err == nil && total > 0 {
			pct = float64(up) / float64(total) * 100.0
		}
		globalUptime = append(globalUptime, uptimeWindow{w.label, math.Round(pct*100) / 100, w.win})
	}

	// ── 2. Per-monitor stats ──────────────────────────────────
	type monitorStat struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Type        string   `json:"type"`
		Group       string   `json:"group"`
		Status      string   `json:"status"`
		Uptime24h   float64  `json:"uptime_24h"`
		AvgLatency  *float64 `json:"avg_latency"`
		MinLatency  *int     `json:"min_latency"`
		MaxLatency  *int     `json:"max_latency"`
		P95Latency  *int     `json:"p95_latency"`
		TotalChecks int      `json:"total_checks"`
	}

	monRows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.name, m.type, COALESCE(m.group_name, ''),
			m.enabled,
			(SELECT status FROM check_results WHERE monitor_id = m.id ORDER BY checked_at DESC LIMIT 1),
			(SELECT COUNT(*) FROM check_results WHERE monitor_id = m.id AND checked_at >= ? AND status = 'up') AS up24,
			(SELECT COUNT(*) FROM check_results WHERE monitor_id = m.id AND checked_at >= ?) AS total24,
			(SELECT AVG(latency_ms) FROM check_results WHERE monitor_id = m.id AND checked_at >= ? AND latency_ms IS NOT NULL),
			(SELECT MIN(latency_ms) FROM check_results WHERE monitor_id = m.id AND checked_at >= ? AND latency_ms IS NOT NULL),
			(SELECT MAX(latency_ms) FROM check_results WHERE monitor_id = m.id AND checked_at >= ? AND latency_ms IS NOT NULL),
			(SELECT COUNT(*) FROM check_results WHERE monitor_id = m.id)
		FROM monitors m
		ORDER BY m.group_name, m.name
	`, now.Add(-24*time.Hour), now.Add(-24*time.Hour), now.Add(-24*time.Hour), now.Add(-24*time.Hour), now.Add(-24*time.Hour))

	var monitorStats []monitorStat
	if err == nil {
		defer monRows.Close()
		for monRows.Next() {
			var ms monitorStat
			var enabled bool
			var status sql.NullString
			var up24, total24 int
			var avgLat sql.NullFloat64
			var minLat, maxLat sql.NullInt64
			if err := monRows.Scan(&ms.ID, &ms.Name, &ms.Type, &ms.Group,
				&enabled, &status, &up24, &total24,
				&avgLat, &minLat, &maxLat, &ms.TotalChecks); err != nil {
				continue
			}
			if !enabled {
				ms.Status = "paused"
			} else if status.Valid {
				ms.Status = status.String
			} else {
				ms.Status = "pending"
			}
			if total24 > 0 {
				ms.Uptime24h = math.Round(float64(up24)/float64(total24)*10000) / 100
			} else {
				ms.Uptime24h = 100
			}
			if avgLat.Valid {
				v := math.Round(avgLat.Float64*10) / 10
				ms.AvgLatency = &v
			}
			if minLat.Valid {
				v := int(minLat.Int64)
				ms.MinLatency = &v
			}
			if maxLat.Valid {
				v := int(maxLat.Int64)
				ms.MaxLatency = &v
			}
			monitorStats = append(monitorStats, ms)
		}
	}

	// Calculate P95 per monitor (separate query for accuracy)
	for i := range monitorStats {
		ms := &monitorStats[i]
		rows, err := s.db.QueryContext(ctx,
			`SELECT latency_ms FROM check_results
			 WHERE monitor_id = ? AND checked_at >= ? AND latency_ms IS NOT NULL
			 ORDER BY latency_ms ASC`, ms.ID, now.Add(-24*time.Hour))
		if err != nil {
			continue
		}
		var latencies []int
		for rows.Next() {
			var l int
			rows.Scan(&l)
			latencies = append(latencies, l)
		}
		rows.Close()
		if len(latencies) > 0 {
			idx := int(math.Ceil(float64(len(latencies))*0.95)) - 1
			if idx < 0 {
				idx = 0
			}
			v := latencies[idx]
			ms.P95Latency = &v
		}
	}

	// ── 3. Hourly timeline (last 24h) ────────────────────────
	type hourBucket struct {
		Hour string `json:"hour"`
		Up   int    `json:"up"`
		Down int    `json:"down"`
	}

	var hourly []hourBucket
	hourRows, err := s.db.QueryContext(ctx, `
		SELECT strftime('%Y-%m-%dT%H:00:00', checked_at) as bucket,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) as down_count
		FROM check_results
		WHERE checked_at >= ?
		GROUP BY bucket
		ORDER BY bucket ASC
	`, now.Add(-24*time.Hour))
	if err == nil {
		defer hourRows.Close()
		for hourRows.Next() {
			var hb hourBucket
			if hourRows.Scan(&hb.Hour, &hb.Up, &hb.Down) == nil {
				hourly = append(hourly, hb)
			}
		}
	}

	// ── 4. Latency distribution ──────────────────────────────
	type latBucket struct {
		Label string `json:"label"`
		Count int    `json:"count"`
	}

	var latDist []latBucket
	latRows, err := s.db.QueryContext(ctx, `
		SELECT
			CASE
				WHEN latency_ms < 50 THEN '<50ms'
				WHEN latency_ms < 200 THEN '50-200ms'
				WHEN latency_ms < 500 THEN '200-500ms'
				WHEN latency_ms < 1000 THEN '500ms-1s'
				ELSE '>1s'
			END as bucket,
			COUNT(*) as cnt
		FROM check_results
		WHERE checked_at >= ? AND latency_ms IS NOT NULL
		GROUP BY bucket
		ORDER BY MIN(latency_ms) ASC
	`, now.Add(-24*time.Hour))
	if err == nil {
		defer latRows.Close()
		for latRows.Next() {
			var lb latBucket
			if latRows.Scan(&lb.Label, &lb.Count) == nil {
				latDist = append(latDist, lb)
			}
		}
	}

	// ── 5. Type breakdown ────────────────────────────────────
	type typeCount struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}

	var typeCounts []typeCount
	typeRows, err := s.db.QueryContext(ctx, `SELECT type, COUNT(*) FROM monitors GROUP BY type ORDER BY COUNT(*) DESC`)
	if err == nil {
		defer typeRows.Close()
		for typeRows.Next() {
			var tc typeCount
			if typeRows.Scan(&tc.Type, &tc.Count) == nil {
				typeCounts = append(typeCounts, tc)
			}
		}
	}

	// ── 6. Response code breakdown (HTTP only, last 24h) ─────
	type codeCount struct {
		Code  string `json:"code"`
		Count int    `json:"count"`
	}

	var codeCounts []codeCount
	codeRows, err := s.db.QueryContext(ctx, `
		SELECT
			CASE
				WHEN status_code >= 200 AND status_code < 300 THEN '2xx'
				WHEN status_code >= 300 AND status_code < 400 THEN '3xx'
				WHEN status_code >= 400 AND status_code < 500 THEN '4xx'
				WHEN status_code >= 500 THEN '5xx'
				ELSE 'other'
			END as code_class,
			COUNT(*) as cnt
		FROM check_results
		WHERE checked_at >= ? AND status_code IS NOT NULL
		GROUP BY code_class
		ORDER BY code_class ASC
	`, now.Add(-24*time.Hour))
	if err == nil {
		defer codeRows.Close()
		for codeRows.Next() {
			var cc codeCount
			if codeRows.Scan(&cc.Code, &cc.Count) == nil {
				codeCounts = append(codeCounts, cc)
			}
		}
	}

	// ── 7. Summary stats ─────────────────────────────────────
	var totalChecks24h, totalChecksAll int
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM check_results WHERE checked_at >= ?`, now.Add(-24*time.Hour)).Scan(&totalChecks24h)
	s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM check_results`).Scan(&totalChecksAll)

	var avgLatGlobal sql.NullFloat64
	s.db.QueryRowContext(ctx,
		`SELECT AVG(latency_ms) FROM check_results WHERE checked_at >= ? AND latency_ms IS NOT NULL`, now.Add(-24*time.Hour)).Scan(&avgLatGlobal)

	var activeIncidents int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM incidents WHERE status != 'resolved'`).Scan(&activeIncidents)

	// Sort monitors: worst uptime first for leaderboard
	sort.Slice(monitorStats, func(i, j int) bool {
		return monitorStats[i].Uptime24h < monitorStats[j].Uptime24h
	})

	// ── Response ─────────────────────────────────────────────
	result := map[string]any{
		"global_uptime":        globalUptime,
		"monitors":             monitorStats,
		"hourly_timeline":      hourly,
		"latency_distribution": latDist,
		"type_breakdown":       typeCounts,
		"response_codes":       codeCounts,
		"summary": map[string]any{
			"total_checks_24h": totalChecks24h,
			"total_checks_all": totalChecksAll,
			"avg_latency_24h":  nil,
			"active_incidents": activeIncidents,
			"monitor_count":    len(monitorStats),
		},
	}
	if avgLatGlobal.Valid {
		result["summary"].(map[string]any)["avg_latency_24h"] = math.Round(avgLatGlobal.Float64*10) / 10
	}

	jsonOK(w, result)
}
