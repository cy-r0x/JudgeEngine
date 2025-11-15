package cmd

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
)

type SystemMetrics struct {
	CPUUsage     prometheus.Gauge
	MemoryUsed   prometheus.Gauge
	DiskUsed     prometheus.Gauge
	DiskRead     prometheus.Counter
	DiskWrite    prometheus.Counter
	NetworkRecv  prometheus.Counter
	NetworkTrans prometheus.Counter
}

func newSystemMetrics() *SystemMetrics {
	return &SystemMetrics{
		CPUUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "Total CPU usage percentage across all cores",
		}),
		MemoryUsed: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_memory_used_bytes",
			Help: "Total used memory in bytes",
		}),
		DiskUsed: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_disk_used_bytes",
			Help: "Total disk usage in bytes for root filesystem",
		}),
		DiskRead: promauto.NewCounter(prometheus.CounterOpts{
			Name: "system_disk_read_bytes_total",
			Help: "Total disk read bytes since boot",
		}),
		DiskWrite: promauto.NewCounter(prometheus.CounterOpts{
			Name: "system_disk_write_bytes_total",
			Help: "Total disk write bytes since boot",
		}),
		NetworkRecv: promauto.NewCounter(prometheus.CounterOpts{
			Name: "system_network_receive_bytes_total",
			Help: "Total received bytes across all interfaces",
		}),
		NetworkTrans: promauto.NewCounter(prometheus.CounterOpts{
			Name: "system_network_transmit_bytes_total",
			Help: "Total transmitted bytes across all interfaces",
		}),
	}
}

func (m *SystemMetrics) Collect() {
	go func() {
		var prevDiskRead, prevDiskWrite uint64
		var prevNetRecv, prevNetTrans uint64

		for {
			// CPU %
			cpuPercent, _ := cpu.Percent(0, false)
			if len(cpuPercent) > 0 {
				m.CPUUsage.Set(cpuPercent[0])
			}

			// Memory
			vmStat, _ := mem.VirtualMemory()
			m.MemoryUsed.Set(float64(vmStat.Used))

			// Disk usage
			diskStat, _ := disk.Usage("/")
			m.DiskUsed.Set(float64(diskStat.Used))

			// Disk I/O
			ioStats, _ := disk.IOCounters()
			var totalRead, totalWrite uint64
			for _, io := range ioStats {
				totalRead += io.ReadBytes
				totalWrite += io.WriteBytes
			}
			m.DiskRead.Add(float64(totalRead - prevDiskRead))
			m.DiskWrite.Add(float64(totalWrite - prevDiskWrite))
			prevDiskRead, prevDiskWrite = totalRead, totalWrite
			// Network I/O
			netStats, _ := gnet.IOCounters(false)
			if len(netStats) > 0 {
				currRecv := netStats[0].BytesRecv
				currTrans := netStats[0].BytesSent
				m.NetworkRecv.Add(float64(currRecv - prevNetRecv))
				m.NetworkTrans.Add(float64(currTrans - prevNetTrans))
				prevNetRecv, prevNetTrans = currRecv, currTrans
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

func (s *Server) RegisterMetrics() {
	node := s.RegisterNode()
	log.Printf("Registered node with ID: %s", node)
	sysMetrics := newSystemMetrics()
	sysMetrics.Collect()
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}
