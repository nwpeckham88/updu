package p2p

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/updu/updu/internal/models"
)

var (
	processStartTime = time.Now()

	telemetryMu sync.Mutex
	cachedPubIP string
	lastPubIPCheck time.Time

	prevCPUTotal uint64
	prevCPUIdle  uint64
	prevCPUTime  time.Time
)

// CollectTelemetry gathers lightweight host metrics without external dependencies.
func CollectTelemetry() *models.PeerTelemetry {
	return &models.PeerTelemetry{
		TailnetIP: DetectTailnetIP(),
		PublicIP:  DetectPublicIP(),
		MemPct:    readMemoryPercent(),
		CPUPct:    readCPUPercent(),
		UptimeS:   readUptimeSeconds(),
	}
}

// DetectTailnetIP finds the Tailscale or WireGuard IPv4 address on the host.
func DetectTailnetIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		// Prefer interfaces named tailscale, wg, utun
		name := strings.ToLower(iface.Name)
		isTailnetNamed := strings.Contains(name, "tailscale") || strings.HasPrefix(name, "wg") || strings.HasPrefix(name, "utun")

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			ip4 := ip.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}

			// Tailscale CGNAT range is 100.64.0.0/10 (100.64.0.0 - 100.127.255.255)
			if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
				return ip4.String()
			}

			if isTailnetNamed {
				return ip4.String()
			}
		}
	}
	return ""
}

// DetectPublicIP returns the detected public WAN IP, cached for 5 minutes.
func DetectPublicIP() string {
	telemetryMu.Lock()
	if cachedPubIP != "" && time.Since(lastPubIPCheck) < 5*time.Minute {
		ip := cachedPubIP
		telemetryMu.Unlock()
		return ip
	}
	telemetryMu.Unlock()

	// Asynchronously resolve or quick query with 1s timeout
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get("https://api.ipify.org")
	if err == nil && resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		ipStr := strings.TrimSpace(string(body))
		if parsed := net.ParseIP(ipStr); parsed != nil {
			telemetryMu.Lock()
			cachedPubIP = ipStr
			lastPubIPCheck = time.Now()
			telemetryMu.Unlock()
			return ipStr
		}
	}

	telemetryMu.Lock()
	defer telemetryMu.Unlock()
	return cachedPubIP
}

func readMemoryPercent() float64 {
	if runtime.GOOS == "linux" {
		f, err := os.Open("/proc/meminfo")
		if err == nil {
			defer f.Close()
			var totalKB, availKB, freeKB, buffersKB, cachedKB uint64
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				fields := strings.Fields(line)
				if len(fields) < 2 {
					continue
				}
				key := strings.TrimSuffix(fields[0], ":")
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				switch key {
				case "MemTotal":
					totalKB = val
				case "MemAvailable":
					availKB = val
				case "MemFree":
					freeKB = val
				case "Buffers":
					buffersKB = val
				case "Cached":
					cachedKB = val
				}
			}
			if totalKB > 0 {
				if availKB > 0 {
					pct := (1.0 - float64(availKB)/float64(totalKB)) * 100.0
					if pct < 0 {
						pct = 0
					}
					if pct > 100 {
						pct = 100
					}
					return pct
				}
				used := totalKB - (freeKB + buffersKB + cachedKB)
				pct := (float64(used) / float64(totalKB)) * 100.0
				return pct
			}
		}
	}

	// Fallback using runtime.MemStats
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	if ms.Sys > 0 {
		pct := (float64(ms.Alloc) / float64(ms.Sys)) * 100.0
		if pct > 100 {
			pct = 100
		}
		return pct
	}
	return 0.0
}

func readCPUPercent() float64 {
	telemetryMu.Lock()
	defer telemetryMu.Unlock()

	if runtime.GOOS == "linux" {
		f, err := os.Open("/proc/stat")
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) >= 5 && fields[0] == "cpu" {
					var total uint64
					var idle uint64
					for i := 1; i < len(fields); i++ {
						v, _ := strconv.ParseUint(fields[i], 10, 64)
						total += v
						if i == 4 || i == 5 { // idle and iowait
							idle += v
						}
					}

					if prevCPUTotal > 0 && total > prevCPUTotal {
						diffTotal := total - prevCPUTotal
						diffIdle := idle - prevCPUIdle
						if diffIdle > diffTotal {
							diffIdle = diffTotal
						}
						diffBusy := diffTotal - diffIdle
						pct := (float64(diffBusy) / float64(diffTotal)) * 100.0

						prevCPUTotal = total
						prevCPUIdle = idle
						prevCPUTime = time.Now()
						return pct
					}

					prevCPUTotal = total
					prevCPUIdle = idle
					prevCPUTime = time.Now()
					return 0.0
				}
			}
		}
	}

	return 0.0
}

func readUptimeSeconds() uint64 {
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/uptime")
		if err == nil {
			fields := strings.Fields(string(data))
			if len(fields) > 0 {
				var secs float64
				if _, err := fmt.Sscanf(fields[0], "%f", &secs); err == nil && secs >= 0 {
					return uint64(secs)
				}
			}
		}
	}

	return uint64(time.Since(processStartTime).Seconds())
}
