package handlers

import "github.com/judgenot0/judge-deamon/config"

type Handler struct {
	Config *config.Config
}

func NewHandler(config *config.Config) *Handler {
	return &Handler{
		Config: config,
	}
}
