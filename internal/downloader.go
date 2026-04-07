package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chsoares/flame/internal/ui"
)

type downloadFeedback struct {
	displayName string
	quiet       bool
}

// DownloadFile downloads a file from URL with CLI progress indication.
func DownloadFile(ctx context.Context, url, destPath string) error {
	return downloadFile(ctx, url, destPath, downloadFeedback{})
}

// DownloadFileQuiet downloads a file from URL without emitting CLI spinner/stdout noise.
func DownloadFileQuiet(ctx context.Context, url, destPath, displayName string) error {
	return downloadFile(ctx, url, destPath, downloadFeedback{displayName: displayName, quiet: true})
}

func downloadFile(ctx context.Context, url, destPath string, feedback downloadFeedback) error {
	displayName := feedback.displayName
	if displayName == "" {
		displayName = filepath.Base(url)
	}

	var spinner *ui.Spinner
	if !feedback.quiet {
		spinner = ui.NewSpinner()
		spinner.Start(fmt.Sprintf("Downloading %s...", displayName))
		defer spinner.Stop()
	}

	// Make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Download with progress
	size := resp.ContentLength
	downloaded := int64(0)
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("download cancelled by user")
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)

			if spinner != nil && size > 0 {
				percent := int(float64(downloaded) / float64(size) * 100)
				kb := downloaded / 1024
				spinner.Update(fmt.Sprintf("Downloading... %d%% (%d KB)", percent, kb))
			} else if spinner != nil {
				kb := downloaded / 1024
				spinner.Update(fmt.Sprintf("Downloading... %d KB", kb))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("download interrupted: %w", err)
		}
	}

	if spinner != nil {
		spinner.Stop()
	}

	// Format size
	sizeStr := formatBytes(downloaded)
	if !feedback.quiet {
		fmt.Println(ui.Success(fmt.Sprintf("Downloaded %s (%s)", displayName, sizeStr)))
	}

	return nil
}

// formatBytes formats bytes into human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
