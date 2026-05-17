package api

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/boris/googrec/internal/auth"
)

const (
	apiBase = "https://pixelrecorder-pa.clients6.google.com/$rpc/java.com.google.wireless.android.pixel.recorder.protos.PlaybackService"
	origin  = "https://recorder.google.com"
)

var (
	uuidRe       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	cdFilenameRe = regexp.MustCompile(`filename="?([^";\n]+)"?`)
)

func ValidateUUID(id string) error {
	if !uuidRe.MatchString(strings.ToLower(id)) {
		return fmt.Errorf("invalid recording ID %q — expected UUID format", id)
	}
	return nil
}

func ComputeSAPISIDHash(sapisid string) string {
	ts := time.Now().Unix()
	input := fmt.Sprintf("%d %s %s", ts, sapisid, origin)
	h := sha1.Sum([]byte(input))
	return fmt.Sprintf("SAPISIDHASH %d_%x", ts, h)
}

type Client struct {
	auth *auth.AuthData
	http *http.Client
}

func NewClient() (*Client, error) {
	a, err := auth.Load()
	if err != nil {
		return nil, fmt.Errorf("load auth: %w", err)
	}
	if a == nil {
		return nil, fmt.Errorf("not authenticated — run `googrec auth` first")
	}
	if a.SAPISID == "" {
		return nil, fmt.Errorf("SAPISID missing — run `googrec auth` to re-authenticate")
	}
	if a.APIKey == "" {
		return nil, fmt.Errorf("no API key configured — run `googrec auth --api-key <key>`")
	}
	return &Client{
		auth: a,
		http: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) rpcRequest(ctx context.Context, method string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := apiBase + "/" + method
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json+protobuf")
	req.Header.Set("X-Goog-Api-Key", c.auth.APIKey)
	req.Header.Set("Authorization", ComputeSAPISIDHash(c.auth.SAPISID))
	req.Header.Set("X-Goog-AuthUser", fmt.Sprintf("%d", c.auth.AuthUser))
	req.Header.Set("X-User-Agent", "grpc-web-javascript/0.1")
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	req.Header.Set("Cookie", c.auth.Cookies)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", method, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("API %s: %d %s — %s", method, resp.StatusCode, resp.Status, preview)
	}

	return respBody, nil
}

func (c *Client) ListRecordings(ctx context.Context, limit int) ([]Recording, error) {
	nowSec := time.Now().Unix()
	body := []any{
		[]any{map[string]any{"1": nowSec}},
		limit,
	}

	data, err := c.rpcRequest(ctx, "GetRecordingList", body)
	if err != nil {
		return nil, err
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse recording list: %w", err)
	}
	if len(raw) == 0 {
		return nil, nil
	}

	var items [][]json.RawMessage
	if err := json.Unmarshal(raw[0], &items); err != nil {
		return nil, fmt.Errorf("parse recording items: %w", err)
	}

	recordings := make([]Recording, 0, len(items))
	for _, item := range items {
		r, err := parseRecording(item)
		if err != nil {
			continue
		}
		recordings = append(recordings, r)
	}

	if limit > 0 && len(recordings) > limit {
		recordings = recordings[:limit]
	}
	return recordings, nil
}

func parseRecording(item []json.RawMessage) (Recording, error) {
	if len(item) < 7 {
		return Recording{}, fmt.Errorf("recording has %d fields, need at least 7", len(item))
	}

	var deviceID, title, location string
	json.Unmarshal(item[0], &deviceID)
	json.Unmarshal(item[1], &title)
	json.Unmarshal(item[6], &location)

	var webID string
	if len(item) > 13 {
		json.Unmarshal(item[13], &webID)
	}

	var createdTs []json.RawMessage
	json.Unmarshal(item[2], &createdTs)
	createdMs := parseTimestamp(createdTs)

	var durTs []json.RawMessage
	json.Unmarshal(item[3], &durTs)
	durationMs := parseTimestamp(durTs)

	id := webID
	if id == "" {
		id = deviceID
	}
	if title == "" {
		title = time.UnixMilli(createdMs).Format(time.DateTime)
	}

	return Recording{
		ID:            id,
		DeviceID:      deviceID,
		Title:         title,
		Date:          time.UnixMilli(createdMs).UTC().Format(time.RFC3339),
		Duration:      FormatDuration(durationMs),
		DurationMs:    durationMs,
		Location:      location,
		HasTranscript: true, // TODO: derive from response when field is identified
	}, nil
}

func parseTimestamp(ts []json.RawMessage) int64 {
	if len(ts) < 2 {
		return 0
	}
	var secStr string
	var nano float64
	json.Unmarshal(ts[0], &secStr)
	json.Unmarshal(ts[1], &nano)

	var sec int64
	fmt.Sscanf(secStr, "%d", &sec)
	return sec*1000 + int64(nano)/1_000_000
}

func (c *Client) GetTranscript(ctx context.Context, recordingID string) (*Transcript, error) {
	if err := ValidateUUID(recordingID); err != nil {
		return nil, err
	}

	data, err := c.rpcRequest(ctx, "GetTranscription", []any{recordingID})
	if err != nil {
		return nil, err
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse transcript: %w", err)
	}
	if len(raw) == 0 {
		return nil, nil
	}

	var segments [][]json.RawMessage
	if err := json.Unmarshal(raw[0], &segments); err != nil {
		return nil, fmt.Errorf("parse transcript segments: %w", err)
	}
	if len(segments) == 0 {
		return nil, nil
	}

	var result []TranscriptSegment
	currentSpeaker := -1
	currentText := ""
	currentStartTime := "00:00"

	for _, seg := range segments {
		if len(seg) < 2 {
			continue
		}

		var words [][]json.RawMessage
		json.Unmarshal(seg[0], &words)

		var speakerID int
		json.Unmarshal(seg[1], &speakerID)

		if speakerID != currentSpeaker && currentText != "" {
			result = append(result, TranscriptSegment{
				Speaker:   fmt.Sprintf("Speaker %d", currentSpeaker+1),
				Text:      strings.TrimSpace(currentText),
				StartTime: currentStartTime,
			})
			currentText = ""
		}

		if speakerID != currentSpeaker {
			currentSpeaker = speakerID
			if len(words) > 0 && len(words[0]) > 2 {
				var msStr string
				json.Unmarshal(words[0][2], &msStr)
				var ms int64
				fmt.Sscanf(msStr, "%d", &ms)
				currentStartTime = FormatTime(ms)
			}
		}

		for _, word := range words {
			var text, formatted string
			if len(word) > 0 {
				json.Unmarshal(word[0], &text)
			}
			if len(word) > 1 {
				json.Unmarshal(word[1], &formatted)
			}
			w := formatted
			if w == "" {
				w = text
			}
			currentText += w + " "
		}
	}

	if currentText != "" {
		result = append(result, TranscriptSegment{
			Speaker:   fmt.Sprintf("Speaker %d", currentSpeaker+1),
			Text:      strings.TrimSpace(currentText),
			StartTime: currentStartTime,
		})
	}

	var rawText strings.Builder
	for i, s := range result {
		if i > 0 {
			rawText.WriteString("\n\n")
		}
		fmt.Fprintf(&rawText, "[%s] (%s)\n%s", s.Speaker, s.StartTime, s.Text)
	}

	return &Transcript{
		RecordingID: recordingID,
		Segments:    result,
		RawText:     rawText.String(),
	}, nil
}

func (c *Client) GetAudio(ctx context.Context, recordingID string, dest io.Writer) (*AudioResult, error) {
	if err := ValidateUUID(recordingID); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://usercontent.recorder.google.com/download/playback/%s?authuser=%d&download=true",
		recordingID, c.auth.AuthUser)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", c.auth.Cookies)
	req.Header.Set("Referer", origin+"/")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("audio download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		preview := string(errBody)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("audio download: %d %s — %s", resp.StatusCode, resp.Status, preview)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/mp4"
	}

	filename := recordingID + ".m4a"
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if m := cdFilenameRe.FindStringSubmatch(cd); m != nil {
			filename = m[1]
		}
	}

	n, err := io.Copy(dest, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stream audio: %w", err)
	}

	return &AudioResult{
		ContentType:  contentType,
		Filename:     filename,
		BytesWritten: n,
	}, nil
}

func (c *Client) TestAuth(ctx context.Context) error {
	_, err := c.ListRecordings(ctx, 1)
	return err
}
