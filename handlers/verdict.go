package handlers

import (
	"log"

	"github.com/judgenot0/judge-deamon/structs"
)

func ProduceVerdict(submission *structs.Submission, finalResult string, maxTime float64, maxRSS int) {

	log.Printf("Submission %s (User %d): %s (Time: %.3fs, Memory: %dKB)",
		submission.Id, submission.UserId, finalResult, maxTime, maxRSS)

	//TODO: Add API Call to update DB

}
