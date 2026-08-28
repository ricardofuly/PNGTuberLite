package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// CurrentVersion is the active build version. Injected via -ldflags during build.
var CurrentVersion = "v1.0.0"

// GitHubRepo points to the official repository.
const (
	GitHubOwner = "ricardofuly"
	GitHubRepo  = "PNGTuberLite"
	ReleaseAPI  = "https://api.github.com/repos/ricardofuly/PNGTuberLite/releases/latest"
)

// ReleaseAsset represents an asset in a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// ReleaseInfo represents the latest GitHub release.
type ReleaseInfo struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt time.Time      `json:"published_at"`
	Assets      []ReleaseAsset `json:"assets"`
	IsHotfix    bool           `json:"-"`
}

// UpdateState tracks background update check and progress.
type UpdateState struct {
	mu           sync.RWMutex
	Checked      bool
	Available    bool
	Latest       *ReleaseInfo
	IsUpdating   bool
	Progress     float32
	ErrorMessage string
	Success      bool
}

var globalUpdateState = &UpdateState{}

// GetUpdateState returns the current update status.
func GetUpdateState() *UpdateState {
	return globalUpdateState
}

// CheckForUpdateAsync checks GitHub releases in the background.
func CheckForUpdateAsync() {
	go func() {
		rel, hasUpdate, err := CheckForUpdate()
		globalUpdateState.mu.Lock()
		defer globalUpdateState.mu.Unlock()
		globalUpdateState.Checked = true
		if err != nil {
			globalUpdateState.ErrorMessage = err.Error()
			return
		}
		if hasUpdate && rel != nil {
			globalUpdateState.Available = true
			globalUpdateState.Latest = rel
		}
	}()
}

// CheckForUpdate queries GitHub for the latest release and compares with CurrentVersion.
func CheckForUpdate() (*ReleaseInfo, bool, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", ReleaseAPI, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "PNGTuberLite-AutoUpdater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("falha ao conectar ao GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("resposta inválida do GitHub: status %d", resp.StatusCode)
	}

	var rel ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, false, fmt.Errorf("falha ao processar resposta: %w", err)
	}

	if rel.TagName == "" {
		return nil, false, fmt.Errorf("nenhuma versão encontrada")
	}

	// Check if release is newer
	hasUpdate := isNewerVersion(rel.TagName, CurrentVersion)
	if strings.Contains(strings.ToLower(rel.TagName), "hotfix") || strings.Contains(strings.ToLower(rel.Name), "hotfix") {
		rel.IsHotfix = true
	}

	return &rel, hasUpdate, nil
}

// isNewerVersion returns true if remote tag is newer than local version.
func isNewerVersion(remote, local string) bool {
	remoteClean := strings.TrimPrefix(strings.TrimSpace(remote), "v")
	localClean := strings.TrimPrefix(strings.TrimSpace(local), "v")

	if remoteClean == localClean {
		return false
	}

	// Simple semantic comparison: e.g. "1.0.1" > "1.0.0"
	var rMaj, rMin, rPat int
	var lMaj, lMin, lPat int
	fmt.Sscanf(remoteClean, "%d.%d.%d", &rMaj, &rMin, &rPat)
	fmt.Sscanf(localClean, "%d.%d.%d", &lMaj, &lMin, &lPat)

	if rMaj > lMaj {
		return true
	}
	if rMaj == lMaj && rMin > lMin {
		return true
	}
	if rMaj == lMaj && rMin == lMin && rPat > lPat {
		return true
	}
	return remoteClean != localClean
}

// ApplyUpdate downloads the release asset for the current OS/architecture and updates the binary in-place.
func ApplyUpdate(rel *ReleaseInfo, progressCallback func(percent float32)) error {
	if rel == nil || len(rel.Assets) == 0 {
		return fmt.Errorf("nenhum arquivo disponível para download")
	}

	// Match asset name by OS
	osName := runtime.GOOS
	archName := runtime.GOARCH

	var targetAsset *ReleaseAsset
	for _, asset := range rel.Assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, osName) && (strings.Contains(name, archName) || strings.Contains(name, "amd64") || strings.Contains(name, "x86_64")) {
			targetAsset = &asset
			break
		}
	}

	// Fallback to any matching OS asset
	if targetAsset == nil {
		for _, asset := range rel.Assets {
			if strings.Contains(strings.ToLower(asset.Name), osName) {
				targetAsset = &asset
				break
			}
		}
	}

	if targetAsset == nil {
		return fmt.Errorf("nenhum pacote compatível com %s/%s encontrado na release %s", osName, archName, rel.TagName)
	}

	// Download archive into memory
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", targetAsset.BrowserDownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "PNGTuberLite-AutoUpdater")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao baixar atualização: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erro no download: status %d", resp.StatusCode)
	}

	totalSize := targetAsset.Size
	if totalSize <= 0 {
		totalSize = resp.ContentLength
	}

	// Read with progress tracking
	var buf bytes.Buffer
	progressReader := &progressTracker{
		reader:   resp.Body,
		total:    totalSize,
		callback: progressCallback,
	}

	if _, err := io.Copy(&buf, progressReader); err != nil {
		return fmt.Errorf("falha durante o download: %w", err)
	}

	// Extract binary
	newBinaryBytes, err := extractExecutableFromArchive(buf.Bytes(), targetAsset.Name)
	if err != nil {
		return fmt.Errorf("falha ao extrair executável: %w", err)
	}

	// Replace current running executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("falha ao obter caminho do executável: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("falha ao resolver symlink do executável: %w", err)
	}

	oldPath := execPath + ".old"
	_ = os.Remove(oldPath)

	// Rename current executable to .old
	if err := os.Rename(execPath, oldPath); err != nil {
		// If rename fails, try direct write (on Linux)
		if err := os.WriteFile(execPath, newBinaryBytes, 0755); err != nil {
			return fmt.Errorf("falha ao substituir executável: %w", err)
		}
		return nil
	}

	// Write new binary
	if err := os.WriteFile(execPath, newBinaryBytes, 0755); err != nil {
		// Restore old binary on failure
		_ = os.Rename(oldPath, execPath)
		return fmt.Errorf("falha ao gravar nova versão: %w", err)
	}

	return nil
}

// progressTracker tracks download percentage.
type progressTracker struct {
	reader   io.Reader
	total    int64
	read     int64
	callback func(percent float32)
}

func (pt *progressTracker) Read(p []byte) (int, error) {
	n, err := pt.reader.Read(p)
	pt.read += int64(n)
	if pt.total > 0 && pt.callback != nil {
		percent := float32(pt.read) / float32(pt.total)
		if percent > 1.0 {
			percent = 1.0
		}
		pt.callback(percent)
	}
	return n, err
}

// extractExecutableFromArchive extracts the main binary from .tar.gz or .zip archive.
func extractExecutableFromArchive(data []byte, filename string) ([]byte, error) {
	filename = strings.ToLower(filename)

	if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		gzReader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			baseName := filepath.Base(header.Name)
			if baseName == "pngtuber-lite" || baseName == "pngtuber-lite.exe" {
				return io.ReadAll(tarReader)
			}
		}
	} else if strings.HasSuffix(filename, ".zip") {
		zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range zipReader.File {
			baseName := filepath.Base(f.Name)
			if baseName == "pngtuber-lite" || baseName == "pngtuber-lite.exe" {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
	}

	// If direct binary download
	return data, nil
}
