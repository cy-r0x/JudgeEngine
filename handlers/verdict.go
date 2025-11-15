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
	"sync"
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
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Generate signed token payload
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

		// Marshal payload into JSON
		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Println("Error marshaling payload:", err)
			return
		}

		url := fmt.Sprintf("%s%s", h.Config.ServerEndpoint, "/api/submissions")
		// Create PUT request
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
		if err != nil {
			log.Println("Error creating PUT request:", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// Send request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Error sending PUT request:", err)
			return
		}
		defer resp.Body.Close()

		bodyResp, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response body:", err)
		} else {
			log.Println("PUT response status:", resp.Status)
			log.Println("PUT response body:", string(bodyResp))
		}

	}()

	wg.Wait()
}
