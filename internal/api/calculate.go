package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// calculateHandler parses user input and gets a result from containersMap.
func (s *Server) calculateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	seedStr, inputStr := vars["seed"], vars["user_input"]

	seed, err := strconv.Atoi(seedStr)
	if err != nil {
		s.l.Errorf("cannot parse seed: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	input, err := strconv.Atoi(inputStr)
	if err != nil {
		s.l.Errorf("cannot parse input: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := s.containersMap.Calculate(r.Context(), seed, input)
	if err != nil {
		s.l.Errorf("cannot calculate result: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(strconv.Itoa(result)))
	if err != nil {
		s.l.Errorf("cannot write result: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}
