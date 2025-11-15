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
	CPUUsage     *prometheus.GaugeVec
	MemoryUsed   *prometheus.GaugeVec
	DiskUsed     *prometheus.GaugeVec
	DiskRead     *prometheus.CounterVec
	DiskWrite    *prometheus.CounterVec
	NetworkRecv  *prometheus.CounterVec
	NetworkTrans *prometheus.CounterVec
	nodeID       string
}

func newSystemMetrics(nodeID string) *SystemMetrics {
	return &SystemMetrics{
		nodeID: nodeID,
		CPUUsage: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "Total CPU usage percentage across all cores",
		}, []string{"node_id"}),
		MemoryUsed: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "system_memory_used_bytes",
			Help: "Total used memory in bytes",
		}, []string{"node_id"}),
		DiskUsed: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "system_disk_used_bytes",
			Help: "Total disk usage in bytes for root filesystem",
		}, []string{"node_id"}),
		DiskRead: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "system_disk_read_bytes_total",
			Help: "Total disk read bytes since boot",
		}, []string{"node_id"}),
		DiskWrite: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "system_disk_write_bytes_total",
			Help: "Total disk write bytes since boot",
		}, []string{"node_id"}),
		NetworkRecv: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "system_network_receive_bytes_total",
			Help: "Total received bytes across all interfaces",
		}, []string{"node_id"}),
		NetworkTrans: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "system_network_transmit_bytes_total",
			Help: "Total transmitted bytes across all interfaces",
		}, []string{"node_id"}),
	}
}

func (m *SystemMetrics) Collect() {
	go func() {
		var prevDiskRead, prevDiskWrite uint64
		var prevNetRecv, prevNetTrans uint64
		firstRun := true

		for {
			// CPU %
			cpuPercent, _ := cpu.Percent(0, false)
			if len(cpuPercent) > 0 {
				m.CPUUsage.WithLabelValues(m.nodeID).Set(cpuPercent[0])
			}

			// Memory
			vmStat, _ := mem.VirtualMemory()
			m.MemoryUsed.WithLabelValues(m.nodeID).Set(float64(vmStat.Used))

			// Disk usage
			diskStat, _ := disk.Usage("/")
			m.DiskUsed.WithLabelValues(m.nodeID).Set(float64(diskStat.Used))

			// Disk I/O
			ioStats, _ := disk.IOCounters()
			var totalRead, totalWrite uint64
			for _, io := range ioStats {
				totalRead += io.ReadBytes
				totalWrite += io.WriteBytes
			}
			if !firstRun {
				m.DiskRead.WithLabelValues(m.nodeID).Add(float64(totalRead - prevDiskRead))
				m.DiskWrite.WithLabelValues(m.nodeID).Add(float64(totalWrite - prevDiskWrite))
			}
			prevDiskRead, prevDiskWrite = totalRead, totalWrite

			// Network I/O
			netStats, _ := gnet.IOCounters(false)
			if len(netStats) > 0 {
				currRecv := netStats[0].BytesRecv
				currTrans := netStats[0].BytesSent
				if !firstRun {
					m.NetworkRecv.WithLabelValues(m.nodeID).Add(float64(currRecv - prevNetRecv))
					m.NetworkTrans.WithLabelValues(m.nodeID).Add(float64(currTrans - prevNetTrans))
				}
				prevNetRecv, prevNetTrans = currRecv, currTrans
			}

			firstRun = false
			time.Sleep(5 * time.Second)
		}
	}()
}

func (s *Server) RegisterMetrics() {
	node := s.RegisterNode()
	log.Println(node)
	sysMetrics := newSystemMetrics(node)
	sysMetrics.Collect()
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(w, r)
}
