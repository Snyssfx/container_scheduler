//go:generate minimock -i containersMap -o ./mock/ -s ".go" -g

package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Server struct {
	l             *zap.SugaredLogger
	port          int
	server        *http.Server
	containersMap containersMap
}

type containersMap interface {
	Calculate(ctx context.Context, seed, input int) (int, error)
}

// NewServer creates new Server.
func NewServer(logger *zap.SugaredLogger, containersMap containersMap, port int) *Server {
	return &Server{
		l:             logger,
		port:          port,
		server:        nil,
		containersMap: containersMap,
	}
}

// Serve starts the Server.
func (s *Server) Serve() {
	r := mux.NewRouter()
	r.HandleFunc("/calculate/{seed:[0-9]+}/{user_input:[0-9]+}", s.calculateHandler)

	s.server = &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", s.port), Handler: r}

	err := s.server.ListenAndServe()
	if err != nil {
		s.l.Errorf("cannot serve main server: %s", err.Error())
	}
}

// Close should be called before shutdown.
func (s *Server) Close() error {
	return s.server.Close()
}
