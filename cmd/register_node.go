package cmd

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
)

type Node struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels,omitempty"`
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

func (s *Server) RegisterNode() string {
	url := s.config.ServerEndpoint + "/register_node"
	ip := getLocalIP()
	if ip == "" {
		return ""
	}
	node := Node{
		Targets: []string{ip + s.config.HttpPort},
	}
	data, err := json.Marshal(node)
	if err != nil {
		return ""
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	decoder := json.NewDecoder(resp.Body)
	var registeredNode Node
	if err := decoder.Decode(&registeredNode); err != nil {
		return ""
	}

	log.Println("Node added to cluster")
	_ = resp.Body.Close()
	return registeredNode.Labels["node"]
}
