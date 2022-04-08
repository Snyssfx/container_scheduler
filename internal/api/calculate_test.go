package api

import (
	"net/http"
	"testing"

	"go.uber.org/zap"
)

func TestServer_calculateHandler(t *testing.T) {
	type fields struct {
		l             *zap.SugaredLogger
		port          int
		server        *http.Server
		containersMap containersMap
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				l:             tt.fields.l,
				port:          tt.fields.port,
				server:        tt.fields.server,
				containersMap: tt.fields.containersMap,
			}
			s.calculateHandler(tt.args.w, tt.args.r)
		})
	}
}
