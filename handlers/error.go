package handlers

import "log"

// FailOnError logs and exits if an error is encountered
func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
