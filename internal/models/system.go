package models

// SystemMetrics provides high-level aggregate counts for the dashboard.
type SystemMetrics struct {
	TotalMonitors    int `json:"total_monitors"`
	MonitorsUp       int `json:"monitors_up"`
	MonitorsDown     int `json:"monitors_down"`
	MonitorsDegraded int `json:"monitors_degraded"`
	MonitorsPaused   int `json:"monitors_paused"`
	ActiveIncidents  int `json:"active_incidents"`
}
