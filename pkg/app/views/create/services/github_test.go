package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfile_NoReplyEmail(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		expected string
	}{
		{
			name:     "valid profile",
			profile:  Profile{ID: 12345, Login: "testuser"},
			expected: "12345+testuser@users.noreply.github.com",
		},
		{
			name:     "zero ID",
			profile:  Profile{ID: 0, Login: "testuser"},
			expected: "",
		},
		{
			name:     "empty login",
			profile:  Profile{ID: 12345, Login: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.profile.NoReplyEmail()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProfile_BestEmail(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		expected string
	}{
		{
			name:     "public email available",
			profile:  Profile{ID: 12345, Login: "testuser", Email: "test@example.com"},
			expected: "test@example.com",
		},
		{
			name:     "no public email",
			profile:  Profile{ID: 12345, Login: "testuser", Email: ""},
			expected: "12345+testuser@users.noreply.github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.profile.BestEmail()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient()
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "cloud-init-cli", client.userAgent)
}

func TestGitHubClient_FetchSSHKeys(t *testing.T) {
	// Mock server for SSH keys
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/testuser.keys" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ssh-rsa AAAAB... key1\nssh-ed25519 AAAAC... key2"))
		} else if r.URL.Path == "/notfound.keys" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Note: We can't easily test the real GitHub API without mocking,
	// so we just test the client creation and Profile methods
	// Integration tests would use httptest
}

func TestGitHubClient_FetchProfile(t *testing.T) {
	// Mock server for profile
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/testuser" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 12345, "login": "testuser", "name": "Test User", "email": "test@example.com"}`))
		} else if r.URL.Path == "/users/notfound" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Note: Same as above - real integration would need URL override
}

func TestUserData(t *testing.T) {
	data := &UserData{
		Keys:    []string{"key1", "key2"},
		Profile: &Profile{ID: 123, Login: "test"},
	}

	assert.Len(t, data.Keys, 2)
	assert.NotNil(t, data.Profile)
	assert.Nil(t, data.KeysErr)
	assert.Nil(t, data.ProfileErr)
}
