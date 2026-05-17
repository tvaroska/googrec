package api

import (
	"fmt"
	"time"
)

type Recording struct {
	ID            string `json:"id"`
	DeviceID      string `json:"deviceId,omitempty"`
	Title         string `json:"title"`
	Date          string `json:"date"`
	Duration      string `json:"duration"`
	DurationMs    int64  `json:"durationMs"`
	Location      string `json:"location,omitempty"`
	HasTranscript bool   `json:"hasTranscript"`
}

type Transcript struct {
	RecordingID string             `json:"recordingId"`
	Segments    []TranscriptSegment `json:"segments"`
	RawText     string             `json:"rawText"`
}

type TranscriptSegment struct {
	Speaker   string `json:"speaker"`
	Text      string `json:"text"`
	StartTime string `json:"startTime,omitempty"`
}

type AudioResult struct {
	ContentType  string
	Filename     string
	BytesWritten int64
}

func FormatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	total := int(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func FormatTime(ms int64) string {
	total := int(ms / 1000)
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
