package images

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDownloader(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)

	downloader := NewDownloader(store)

	assert.NotNil(t, downloader)
	assert.NotNil(t, downloader.client)
	assert.NotNil(t, downloader.active)
}

func TestDownloader_Download_Success(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	// Create test server
	content := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "downloads", "test.img")

	err := downloader.Download(context.Background(), DownloadOptions{
		URL:      server.URL,
		DestPath: destPath,
	})

	require.NoError(t, err)

	// Verify file exists and has correct content
	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	// Verify temp file was cleaned up
	_, err = os.Stat(destPath + ".downloading")
	assert.True(t, os.IsNotExist(err))
}

func TestDownloader_Download_WithProgress(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	content := []byte("test file content for progress tracking")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "test.img")

	var progressCalls int32
	var lastDownloaded int64

	err := downloader.Download(context.Background(), DownloadOptions{
		URL:      server.URL,
		DestPath: destPath,
		OnProgress: func(downloaded, total int64) {
			atomic.AddInt32(&progressCalls, 1)
			lastDownloaded = downloaded
		},
	})

	require.NoError(t, err)
	assert.Greater(t, progressCalls, int32(0))
	assert.Equal(t, int64(len(content)), lastDownloaded)
}

func TestDownloader_Download_WithChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	content := []byte("test content for checksum")
	h := sha256.New()
	h.Write(content)
	expectedHash := fmt.Sprintf("%x", h.Sum(nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "test.img")

	err := downloader.Download(context.Background(), DownloadOptions{
		URL:      server.URL,
		DestPath: destPath,
		SHA256:   expectedHash,
	})

	require.NoError(t, err)
}

func TestDownloader_Download_ChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	content := []byte("test content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "test.img")

	err := downloader.Download(context.Background(), DownloadOptions{
		URL:      server.URL,
		DestPath: destPath,
		SHA256:   "0000000000000000000000000000000000000000000000000000000000000000",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// Verify temp file was cleaned up
	_, err = os.Stat(destPath + ".downloading")
	assert.True(t, os.IsNotExist(err))

	// Verify dest file doesn't exist
	_, err = os.Stat(destPath)
	assert.True(t, os.IsNotExist(err))
}

func TestDownloader_Download_HTTPError(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "test.img")

	err := downloader.Download(context.Background(), DownloadOptions{
		URL:      server.URL,
		DestPath: destPath,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestDownloader_Download_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	destPath := filepath.Join(tmpDir, "test.img")

	// Cancel immediately
	cancel()

	err := downloader.Download(ctx, DownloadOptions{
		URL:      server.URL,
		DestPath: destPath,
	})

	assert.Error(t, err)
}

func TestDownloader_StartBackgroundDownload(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	content := []byte("background download content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "bg-download.img")

	err := downloader.StartBackgroundDownload("test-download", server.URL, destPath)
	require.NoError(t, err)

	// Wait for download to complete
	time.Sleep(500 * time.Millisecond)

	// Verify file exists
	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestDownloader_StartBackgroundDownload_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content"))
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "test.img")

	// Start first download
	err := downloader.StartBackgroundDownload("test-id", server.URL, destPath)
	require.NoError(t, err)

	// Try to start duplicate
	err = downloader.StartBackgroundDownload("test-id", server.URL, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")

	// Cancel to clean up before test exits
	_ = downloader.CancelDownload("test-id")
	time.Sleep(100 * time.Millisecond)
}

func TestDownloader_CancelDownload(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	// Use a channel to coordinate server handling
	requestReceived := make(chan struct{})

	// Server that waits for cancellation signal
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestReceived)
		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "test.img")

	err := downloader.StartBackgroundDownload("cancel-test", server.URL, destPath)
	require.NoError(t, err)

	// Wait for request to reach server
	select {
	case <-requestReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for request")
	}

	// Cancel the download
	err = downloader.CancelDownload("cancel-test")
	require.NoError(t, err)

	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)
}

func TestDownloader_CancelDownload_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	err := downloader.CancelDownload("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download not found")
}

func TestDownloader_GetActiveDownloads_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	downloads, err := downloader.GetActiveDownloads()
	require.NoError(t, err)
	assert.Empty(t, downloads)
}

func TestDownloader_GetActiveDownloads_WithDownloads(t *testing.T) {
	tmpDir := t.TempDir()
	store := settings.NewStoreWithDir(tmpDir)
	downloader := NewDownloader(store)

	// Manually add a download state
	state := settings.NewDownloadState()
	state.ActiveDownloads = append(state.ActiveDownloads, settings.Download{
		ID:        "test-1",
		Status:    settings.StatusDownloading,
		StartedAt: time.Now(),
	})
	state.ActiveDownloads = append(state.ActiveDownloads, settings.Download{
		ID:        "test-2",
		Status:    settings.StatusComplete,
		StartedAt: time.Now(),
	})
	err := store.SaveDownloadState(state)
	require.NoError(t, err)

	downloads, err := downloader.GetActiveDownloads()
	require.NoError(t, err)

	// Should only return downloading ones
	assert.Len(t, downloads, 1)
	assert.Equal(t, "test-1", downloads[0].ID)
}

func TestProgressReader(t *testing.T) {
	content := []byte("test content for progress reader")
	reader := &progressReader{
		reader: &mockReader{data: content},
		total:  int64(len(content)),
	}

	var progressCalls int
	var lastProgress int64

	reader.onProgress = func(downloaded, total int64) {
		progressCalls++
		lastProgress = downloaded
	}

	buf := make([]byte, 1024)
	n, err := reader.Read(buf)

	require.NoError(t, err)
	assert.Equal(t, len(content), n)
	assert.Equal(t, 1, progressCalls)
	assert.Equal(t, int64(len(content)), lastProgress)
}

// mockReader is a simple io.Reader for testing
type mockReader struct {
	data []byte
	pos  int
}

func (m *mockReader) Read(p []byte) (int, error) {
	if m.pos >= len(m.data) {
		return 0, nil
	}
	n := copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}
