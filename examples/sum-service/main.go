package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type SumResponse struct {
	A      float64 `json:"a"`
	B      float64 `json:"b"`
	Result float64 `json:"result"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func sumHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	aStr := r.URL.Query().Get("a")
	bStr := r.URL.Query().Get("b")

	if aStr == "" || bStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "parameters 'a' and 'b' are required"})
		return
	}

	a, err := strconv.ParseFloat(aStr, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("invalid value for 'a': %s", aStr)})
		return
	}

	b, err := strconv.ParseFloat(bStr, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: fmt.Sprintf("invalid value for 'b': %s", bStr)})
		return
	}

	json.NewEncoder(w).Encode(SumResponse{A: a, B: b, Result: a + b})
}

func main() {
	http.HandleFunc("/sum", sumHandler)
	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}
