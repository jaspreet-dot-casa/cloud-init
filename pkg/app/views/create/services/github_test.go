package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		switch r.URL.Path {
		case "/testuser.keys":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ssh-rsa AAAAB... key1\nssh-ed25519 AAAAC... key2"))
		case "/notfound.keys":
			w.WriteHeader(http.StatusNotFound)
		case "/error.keys":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client pointing at test server
	client := &GitHubClient{
		httpClient: server.Client(),
		userAgent:  "test-agent",
		baseURL:    server.URL,
	}

	t.Run("successful fetch", func(t *testing.T) {
		keys, err := client.FetchSSHKeys("testuser")
		require.NoError(t, err)
		assert.Len(t, keys, 2)
		assert.Equal(t, "ssh-rsa AAAAB... key1", keys[0])
		assert.Equal(t, "ssh-ed25519 AAAAC... key2", keys[1])
	})

	t.Run("user not found", func(t *testing.T) {
		keys, err := client.FetchSSHKeys("notfound")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Nil(t, keys)
	})

	t.Run("server error", func(t *testing.T) {
		keys, err := client.FetchSSHKeys("error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
		assert.Nil(t, keys)
	})
}

func TestGitHubClient_FetchProfile(t *testing.T) {
	// Mock server for profile
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/testuser":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 12345, "login": "testuser", "name": "Test User", "email": "test@example.com"}`))
		case "/users/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/users/error":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client pointing at test server
	client := &GitHubClient{
		httpClient: server.Client(),
		userAgent:  "test-agent",
		apiBaseURL: server.URL,
	}

	t.Run("successful fetch", func(t *testing.T) {
		profile, err := client.FetchProfile("testuser")
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, int64(12345), profile.ID)
		assert.Equal(t, "testuser", profile.Login)
		assert.Equal(t, "Test User", profile.Name)
		assert.Equal(t, "test@example.com", profile.Email)
	})

	t.Run("user not found", func(t *testing.T) {
		profile, err := client.FetchProfile("notfound")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Nil(t, profile)
	})

	t.Run("server error", func(t *testing.T) {
		profile, err := client.FetchProfile("error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
		assert.Nil(t, profile)
	})
}

func TestUserData_Struct(t *testing.T) {
	data := &UserData{
		Keys:    []string{"key1", "key2"},
		Profile: &Profile{ID: 123, Login: "test"},
	}

	assert.Len(t, data.Keys, 2)
	assert.NotNil(t, data.Profile)
	assert.Nil(t, data.KeysErr)
	assert.Nil(t, data.ProfileErr)
}

func TestGitHubClient_FetchUserData(t *testing.T) {
	tests := []struct {
		name           string
		keysHandler    func(w http.ResponseWriter)
		profileHandler func(w http.ResponseWriter)
		expectKeys     []string
		expectKeysErr  bool
		expectProfile  *Profile
		expectProfErr  bool
	}{
		{
			name: "both succeed",
			keysHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ssh-rsa key1\nssh-ed25519 key2"))
			},
			profileHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 123, "login": "user", "name": "User", "email": "user@example.com"}`))
			},
			expectKeys:    []string{"ssh-rsa key1", "ssh-ed25519 key2"},
			expectKeysErr: false,
			expectProfile: &Profile{ID: 123, Login: "user", Name: "User", Email: "user@example.com"},
			expectProfErr: false,
		},
		{
			name: "keys fail profile succeeds",
			keysHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNotFound)
			},
			profileHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 456, "login": "user2"}`))
			},
			expectKeys:    nil,
			expectKeysErr: true,
			expectProfile: &Profile{ID: 456, Login: "user2"},
			expectProfErr: false,
		},
		{
			name: "keys succeed profile fails",
			keysHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ssh-rsa keyonly"))
			},
			profileHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectKeys:    []string{"ssh-rsa keyonly"},
			expectKeysErr: false,
			expectProfile: nil,
			expectProfErr: true,
		},
		{
			name: "both fail",
			keysHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			profileHandler: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectKeys:    nil,
			expectKeysErr: true,
			expectProfile: nil,
			expectProfErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server that handles both endpoints
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/testuser.keys" {
					tt.keysHandler(w)
				} else if r.URL.Path == "/users/testuser" {
					tt.profileHandler(w)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Create client pointing at test server
			client := &GitHubClient{
				httpClient: server.Client(),
				userAgent:  "test-agent",
				baseURL:    server.URL,
				apiBaseURL: server.URL,
			}

			data := client.FetchUserData("testuser")
			require.NotNil(t, data)

			// Check keys
			if tt.expectKeysErr {
				assert.Error(t, data.KeysErr)
				assert.Nil(t, data.Keys)
			} else {
				assert.NoError(t, data.KeysErr)
				assert.Equal(t, tt.expectKeys, data.Keys)
			}

			// Check profile
			if tt.expectProfErr {
				assert.Error(t, data.ProfileErr)
				assert.Nil(t, data.Profile)
			} else {
				assert.NoError(t, data.ProfileErr)
				require.NotNil(t, data.Profile)
				assert.Equal(t, tt.expectProfile.ID, data.Profile.ID)
				assert.Equal(t, tt.expectProfile.Login, data.Profile.Login)
			}
		})
	}
}
