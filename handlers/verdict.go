package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/judgenot0/judge-deamon/structs"
)

type EngineData struct {
	SubmissionId    int64    `json:"submission_id"`
	ProblemId       int64    `json:"problem_id"`
	Verdict         string   `json:"verdict"`
	ExecutionTime   *float32 `json:"execution_time"`
	ExecutionMemory *float32 `json:"execution_memory"`
	Timestamp       int64    `json:"timestamp"`
}

type EnginePayload struct {
	Data        *EngineData `json:"payload"`
	AccessToken string      `json:"access_token"`
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func GenerateToken(submissionId int64, problemId int64, verdict string, execTime, execMem *float32, secret string) (*EnginePayload, error) {
	data := &EngineData{
		SubmissionId:    submissionId,
		ProblemId:       problemId,
		Verdict:         verdict,
		ExecutionTime:   execTime,
		ExecutionMemory: execMem,
		Timestamp:       time.Now().Unix(),
	}

	message, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	accessToken := hex.EncodeToString(expectedMAC)

	return &EnginePayload{
		Data:        data,
		AccessToken: accessToken,
	}, nil
}

func (h *Handler) ProduceVerdict(verdict *structs.Verdict) {
	if verdict == nil || verdict.Submission == nil {
		log.Println("Error: verdict or submission is nil")
		return
	}

	if verdict.Submission.SubmissionId == nil || verdict.Submission.ProblemId == nil {
		log.Println("Error: submission_id or problem_id is nil")
		return
	}

	go func() {
		payload, err := GenerateToken(
			*(verdict.Submission.SubmissionId),
			*(verdict.Submission.ProblemId),
			verdict.Result,
			verdict.MaxTime,
			verdict.MaxRSS,
			h.Config.EngineKey,
		)
		if err != nil {
			log.Println("Error generating token:", err)
			return
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Println("Error marshaling payload:", err)
			return
		}

		endpoint := strings.TrimSuffix(h.Config.ServerEndpoint, "/")
		url := fmt.Sprintf("%s/api/submissions", endpoint)

		req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error creating PUT request:", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Println("Error sending PUT request:", err)
			return
		}
		defer resp.Body.Close()

		bodyResp, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			log.Printf("PUT request failed with status %d: %s", resp.StatusCode, string(bodyResp))
			return
		}

		log.Printf("PUT response status: %s", resp.Status)
		log.Printf("PUT response body: %s", string(bodyResp))
	}()
}
