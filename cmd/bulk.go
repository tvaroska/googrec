package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/boris/googrec/internal/api"
)

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9 _-]`)

func safeFilename(title string, maxLen int) string {
	s := unsafeChars.ReplaceAllString(title, "")
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return strings.TrimSpace(s)
}

type bulkConfig struct {
	outDir      string
	limit       int
	since       string
	skipExisting bool
	concurrency int
}

func filterBySince(recordings []api.Recording, since string) ([]api.Recording, error) {
	if since == "" {
		return recordings, nil
	}
	sinceDate, err := time.Parse("2006-01-02", since)
	if err != nil {
		sinceDate, err = time.Parse(time.RFC3339, since)
		if err != nil {
			return nil, fmt.Errorf("invalid date %q — use YYYY-MM-DD or ISO 8601", since)
		}
	}
	var filtered []api.Recording
	for _, r := range recordings {
		t, _ := time.Parse(time.RFC3339, r.Date)
		if !t.Before(sinceDate) {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

type downloadJob struct {
	index     int
	recording api.Recording
	filepath  string
}

type downloadResult struct {
	index   int
	title   string
	status  string // "done", "skipped", "failed"
	detail  string
}

func runBulkDownload(
	client *api.Client,
	cfg bulkConfig,
	ext string,
	downloadFn func(client *api.Client, r api.Recording, filepath string) (string, error),
) error {
	outDir, err := filepath.Abs(cfg.outDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	fmt.Printf("Fetching recording list (limit: %d)...\n", cfg.limit)
	recordings, err := client.ListRecordings(cfg.limit)
	if err != nil {
		return err
	}

	recordings, err = filterBySince(recordings, cfg.since)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d recording(s) to download.\n\n", len(recordings))
	if len(recordings) == 0 {
		return nil
	}

	// Build jobs
	var jobs []downloadJob
	var skippedExisting int
	for i, r := range recordings {
		safe := safeFilename(r.Title, 80)
		t, _ := time.Parse(time.RFC3339, r.Date)
		dateStr := t.Format("2006-01-02")
		filename := fmt.Sprintf("%s %s.%s", dateStr, safe, ext)
		fp := filepath.Join(outDir, filename)

		if cfg.skipExisting {
			if _, err := os.Stat(fp); err == nil {
				skippedExisting++
				continue
			}
		}

		jobs = append(jobs, downloadJob{index: i + 1, recording: r, filepath: fp})
	}

	total := len(jobs) + skippedExisting
	var downloaded, skipped, failed int64
	skipped = int64(skippedExisting)

	conc := cfg.concurrency
	if conc <= 0 {
		conc = 3
	}
	if conc > len(jobs) {
		conc = len(jobs)
	}

	var wg sync.WaitGroup
	jobCh := make(chan downloadJob, len(jobs))
	resultCh := make(chan downloadResult, len(jobs))

	for i := 0; i < conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				detail, err := downloadFn(client, job.recording, job.filepath)
				if err != nil {
					resultCh <- downloadResult{job.index, job.recording.Title, "failed", err.Error()}
				} else if detail == "skipped" {
					resultCh <- downloadResult{job.index, job.recording.Title, "skipped", ""}
				} else {
					resultCh <- downloadResult{job.index, job.recording.Title, "done", detail}
				}
			}
		}()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for r := range resultCh {
		truncTitle := r.title
		if len(truncTitle) > 60 {
			truncTitle = truncTitle[:60]
		}
		switch r.status {
		case "done":
			atomic.AddInt64(&downloaded, 1)
			d := atomic.LoadInt64(&downloaded) + atomic.LoadInt64(&skipped) + atomic.LoadInt64(&failed)
			fmt.Printf("  [%d/%d] %s... %s\n", d, total, truncTitle, r.detail)
		case "skipped":
			atomic.AddInt64(&skipped, 1)
		case "failed":
			atomic.AddInt64(&failed, 1)
			d := atomic.LoadInt64(&downloaded) + atomic.LoadInt64(&skipped) + atomic.LoadInt64(&failed)
			fmt.Printf("  [%d/%d] %s... FAILED: %s\n", d, total, truncTitle, r.detail)
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Downloaded: %d\n", downloaded)
	fmt.Printf("Skipped:    %d\n", skipped)
	fmt.Printf("Failed:     %d\n", failed)
	fmt.Printf("Output dir: %s\n", outDir)

	return nil
}
