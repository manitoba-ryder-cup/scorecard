package rest

import (
	"testing"
)

// Without a write or idle deadline a slow or idle client holds its connection open
// indefinitely, so a handful of them can exhaust the server.
func TestServerSetsConnectionDeadlines(t *testing.T) {
	server := NewServer(&Config{})

	if server.ReadHeaderTimeout == 0 {
		t.Error("ReadHeaderTimeout is unset")
	}
	if server.WriteTimeout == 0 {
		t.Error("WriteTimeout is unset; a slow reader can hold a connection indefinitely")
	}
	if server.IdleTimeout == 0 {
		t.Error("IdleTimeout is unset; keep-alive connections are never reaped")
	}
}
