package handlers

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:  "healthy",
		Service: "bracket",
	}
	json.NewEncoder(w).Encode(resp)
}
