package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type Node struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type registerResponse struct {
	Success bool `json:"success"`
	Data    Node `json:"data"`
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func listenPort(port string) string {
	if port == "" {
		return ":8080"
	}
	if port[0] != ':' {
		return ":" + port
	}
	return port
}

func (s *Server) RegisterNode() string {
	endpoint := strings.TrimSuffix(s.config.ServerEndpoint, "/")
	url := endpoint + "/register_node"
	ip := getLocalIP()
	if ip == "" {
		return ""
	}
	node := Node{
		Targets: []string{ip + listenPort(s.config.HttpPort)},
	}
	data, err := json.Marshal(node)
	if err != nil {
		return ""
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to register node: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read register_node response: %v", err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("register_node failed: %d %s", resp.StatusCode, string(body))
		return ""
	}

	var wrap registerResponse
	if err := json.Unmarshal(body, &wrap); err != nil {
		log.Printf("Failed to decode register_node response: %v", err)
		return ""
	}

	registered := wrap.Data
	if registered.Labels == nil {
		if err := json.Unmarshal(body, &registered); err != nil || registered.Labels == nil {
			return ""
		}
	}

	log.Println("Node added to cluster")
	return registered.Labels["node"]
}
