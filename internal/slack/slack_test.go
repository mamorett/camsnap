package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsSlackID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"C01234567", true},
		{"U01234567", true},
		{"D01234567", true},
		{"G01234567", true},
		{"W01234567", true},
		{"c01234567", false}, // must be uppercase
		{"short", false},
		{"too_long_id_here", false},
		{"invalid!@#", false},
	}

	for _, tt := range tests {
		if got := isSlackID(tt.id); got != tt.want {
			t.Errorf("isSlackID(%q) = %v; want %v", tt.id, got, tt.want)
		}
	}
}

func TestResolveChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/conversations.list":
			resp := ConversationsListResponse{
				OK: true,
				Channels: []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					{ID: "C111", Name: "general"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/users.list":
			resp := UsersListResponse{
				OK: true,
				Members: []struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Profile struct {
						DisplayName string `json:"display_name"`
					} `json:"profile"`
				}{
					{ID: "U222", Name: "alice"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/conversations.open":
			resp := ConversationsOpenResponse{
				OK: true,
				Channel: struct {
					ID string `json:"id"`
				}{ID: "D333"},
			}
			json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	uploader := NewSlackUploader("test-token")
	uploader.APIBase = server.URL

	tests := []struct {
		nameOrID string
		want     string
		wantErr  bool
	}{
		{"C111", "C111", false},     // Direct ID
		{"#general", "C111", false}, // Name with #
		{"general", "C111", false},  // Name without #
		{"@alice", "D333", false},   // User with @
		{"alice", "D333", false},    // User without @ (after channel lookup fails)
		{"U222", "D333", false},     // User ID
		{"nonexistent", "", true},   // Not found
	}

	for _, tt := range tests {
		got, err := uploader.ResolveChannel(tt.nameOrID)
		if (err != nil) != tt.wantErr {
			t.Errorf("ResolveChannel(%q) error = %v, wantErr %v", tt.nameOrID, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ResolveChannel(%q) = %q, want %q", tt.nameOrID, got, tt.want)
		}
	}
}
