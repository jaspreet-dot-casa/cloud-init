package images

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/globalconfig"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/settings"
)

// ProgressCallback is called with download progress updates.
type ProgressCallback func(downloaded, total int64)

// Downloader handles image downloads.
type Downloader struct {
	store    *settings.Store
	client   *http.Client
	mu       sync.Mutex
	active   map[string]*downloadTask
}

type downloadTask struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// NewDownloader creates a new downloader.
func NewDownloader(store *settings.Store) *Downloader {
	return &Downloader{
		store: store,
		client: &http.Client{
			Timeout: 0, // No timeout for large downloads
		},
		active: make(map[string]*downloadTask),
	}
}

// DownloadOptions configures a download.
type DownloadOptions struct {
	URL        string
	DestPath   string
	SHA256     string // Expected checksum (optional)
	OnProgress ProgressCallback
}

// Download downloads a file with progress tracking.
func (d *Downloader) Download(ctx context.Context, opts DownloadOptions) error {
	// Create destination directory
	destDir := filepath.Dir(opts.DestPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create temporary file
	tmpPath := opts.DestPath + ".downloading"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Track if we successfully renamed the file
	renamed := false
	defer func() {
		out.Close()
		// Only remove temp file if we didn't successfully rename it
		if !renamed {
			os.Remove(tmpPath)
		}
	}()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", opts.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Get total size
	total := resp.ContentLength

	// Create progress reader
	reader := &progressReader{
		reader:     resp.Body,
		total:      total,
		onProgress: opts.OnProgress,
	}

	// Copy with progress
	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Close file before rename
	out.Close()

	// Verify checksum if provided
	if opts.SHA256 != "" {
		hash, err := calculateSHA256(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to calculate checksum: %w", err)
		}
		if hash != opts.SHA256 {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", opts.SHA256, hash)
		}
	}

	// Move to final destination
	if err := os.Rename(tmpPath, opts.DestPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}
	renamed = true

	return nil
}

// DownloadCloudImage downloads a cloud image from the registry.
func (d *Downloader) DownloadCloudImage(ctx context.Context, version, arch string, onProgress ProgressCallback) (*settings.CloudImage, error) {
	registry := NewRegistry()
	info := registry.GetCloudImageInfo(version, arch)
	if info == nil {
		return nil, fmt.Errorf("unknown image: %s %s", version, arch)
	}

	// Determine destination path from global config
	cfg, err := globalconfig.LoadOrCreate()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	imagesDir := cfg.ImagesDir
	if imagesDir == "" {
		imagesDir = globalconfig.DefaultImagesDir()
	}
	destPath := filepath.Join(imagesDir, info.Filename)

	// Download
	err = d.Download(ctx, DownloadOptions{
		URL:        info.URL,
		DestPath:   destPath,
		SHA256:     info.SHA256,
		OnProgress: onProgress,
	})
	if err != nil {
		return nil, err
	}

	// Add to registry
	manager := NewManager(d.store)
	img, err := manager.AddExistingImage(destPath, version, arch)
	if err != nil {
		return nil, fmt.Errorf("failed to register image: %w", err)
	}

	// Update with URL
	img.URL = info.URL
	s, err := d.store.Load()
	if err != nil {
		return img, fmt.Errorf("failed to reload settings: %w", err)
	}
	s.AddCloudImage(*img)
	if err := d.store.Save(s); err != nil {
		return img, fmt.Errorf("failed to save image metadata: %w", err)
	}

	return img, nil
}

// StartBackgroundDownload starts a download in the background.
func (d *Downloader) StartBackgroundDownload(id, url, destPath string) error {
	d.mu.Lock()
	if _, exists := d.active[id]; exists {
		d.mu.Unlock()
		return fmt.Errorf("download already in progress: %s", id)
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := &downloadTask{
		cancel: cancel,
		done:   make(chan struct{}),
	}
	d.active[id] = task
	d.mu.Unlock()

	// Update state
	d.updateDownloadState(id, settings.Download{
		ID:        id,
		URL:       url,
		DestPath:  destPath,
		Status:    settings.StatusDownloading,
		StartedAt: time.Now(),
	})

	// Start download goroutine
	go func() {
		defer close(task.done)
		defer func() {
			d.mu.Lock()
			delete(d.active, id)
			d.mu.Unlock()
		}()

		err := d.Download(ctx, DownloadOptions{
			URL:      url,
			DestPath: destPath,
			OnProgress: func(downloaded, total int64) {
				d.updateDownloadState(id, settings.Download{
					ID:         id,
					URL:        url,
					DestPath:   destPath,
					Downloaded: downloaded,
					TotalBytes: total,
					Status:     settings.StatusDownloading,
				})
			},
		})

		if err != nil {
			d.updateDownloadState(id, settings.Download{
				ID:       id,
				URL:      url,
				DestPath: destPath,
				Status:   settings.StatusError,
				Error:    err.Error(),
			})
		} else {
			d.updateDownloadState(id, settings.Download{
				ID:       id,
				URL:      url,
				DestPath: destPath,
				Status:   settings.StatusComplete,
			})
		}
	}()

	return nil
}

// CancelDownload cancels an active download.
func (d *Downloader) CancelDownload(id string) error {
	d.mu.Lock()
	task, exists := d.active[id]
	d.mu.Unlock()

	if !exists {
		return fmt.Errorf("download not found: %s", id)
	}

	task.cancel()
	<-task.done

	return nil
}

// IsDownloadActive checks if a download with the given ID is currently active.
func (d *Downloader) IsDownloadActive(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, exists := d.active[id]
	return exists
}

// WaitForDownload waits for a download to complete with a timeout.
// Returns true if the download completed, false if timeout elapsed or download not found.
func (d *Downloader) WaitForDownload(id string, timeout time.Duration) bool {
	d.mu.Lock()
	task, exists := d.active[id]
	d.mu.Unlock()

	if !exists {
		return false
	}

	select {
	case <-task.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// GetActiveDownloads returns currently active downloads.
func (d *Downloader) GetActiveDownloads() ([]settings.Download, error) {
	state, err := d.store.LoadDownloadState()
	if err != nil {
		return nil, err
	}

	// Filter to only active downloads
	var active []settings.Download
	for _, dl := range state.ActiveDownloads {
		if dl.Status == settings.StatusDownloading {
			active = append(active, dl)
		}
	}
	return active, nil
}

// updateDownloadState updates the download state file.
func (d *Downloader) updateDownloadState(id string, download settings.Download) {
	state, err := d.store.LoadDownloadState()
	if err != nil {
		state = settings.NewDownloadState()
	}

	// Update or add download
	found := false
	for i := range state.ActiveDownloads {
		if state.ActiveDownloads[i].ID == id {
			state.ActiveDownloads[i] = download
			found = true
			break
		}
	}
	if !found {
		state.ActiveDownloads = append(state.ActiveDownloads, download)
	}

	// Remove completed/errored downloads older than 1 hour
	var filtered []settings.Download
	for _, dl := range state.ActiveDownloads {
		if dl.Status == settings.StatusDownloading || dl.Status == settings.StatusPaused {
			filtered = append(filtered, dl)
		} else if time.Since(dl.StartedAt) < time.Hour {
			filtered = append(filtered, dl)
		}
	}
	state.ActiveDownloads = filtered

	if err := d.store.SaveDownloadState(state); err != nil {
		// Log but don't fail - download state is not critical
		fmt.Printf("warning: failed to save download state for %s: %v\n", id, err)
	}
}

// progressReader wraps a reader and reports progress.
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	onProgress ProgressCallback
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.downloaded += int64(n)
	if r.onProgress != nil {
		r.onProgress(r.downloaded, r.total)
	}
	return n, err
}
