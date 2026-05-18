package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tvaroska/googrec/internal/auth"
)

func TestComputeSAPISIDHash(t *testing.T) {
	hash := ComputeSAPISIDHash("test-sapisid")
	if !strings.HasPrefix(hash, "SAPISIDHASH ") {
		t.Errorf("hash should start with 'SAPISIDHASH ', got %q", hash)
	}
	parts := strings.SplitN(hash, " ", 2)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	tsParts := strings.SplitN(parts[1], "_", 2)
	if len(tsParts) != 2 {
		t.Fatalf("expected timestamp_hash, got %q", parts[1])
	}
	if len(tsParts[1]) != 40 {
		t.Errorf("SHA1 hex should be 40 chars, got %d", len(tsParts[1]))
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"bf3451e0-4ea6-424e-8e77-fbef4c0fe17c", false},
		{"BF3451E0-4EA6-424E-8E77-FBEF4C0FE17C", false},
		{"not-a-uuid", true},
		{"", true},
		{"bf3451e04ea6424e8e77fbef4c0fe17c", true},
	}
	for _, tt := range tests {
		err := ValidateUUID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateUUID(%q): err=%v, wantErr=%v", tt.id, err, tt.wantErr)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "0:00"},
		{5230, "0:05"},
		{60000, "1:00"},
		{323000, "5:23"},
		{3723000, "1:02:03"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "00:00"},
		{5000, "00:05"},
		{65000, "01:05"},
		{3600000, "60:00"},
	}
	for _, tt := range tests {
		got := FormatTime(tt.ms)
		if got != tt.want {
			t.Errorf("FormatTime(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestParseRecording(t *testing.T) {
	raw := `["device-id-123","Test Recording",["1706140050","207000000"],["1992","0"],0,0,"",null,null,null,null,"",null,"bf3451e0-4ea6-424e-8e77-fbef4c0fe17c"]`

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r, err := parseRecording(items)
	if err != nil {
		t.Fatalf("parseRecording: %v", err)
	}

	if r.ID != "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c" {
		t.Errorf("ID = %q", r.ID)
	}
	if r.DeviceID != "device-id-123" {
		t.Errorf("DeviceID = %q", r.DeviceID)
	}
	if r.Title != "Test Recording" {
		t.Errorf("Title = %q", r.Title)
	}
	if r.Duration != "33:12" {
		t.Errorf("Duration = %q, want 33:12", r.Duration)
	}
}

func TestParseRecordingMissingWebID(t *testing.T) {
	raw := `["device-id","Title",["1706140050","0"],["60","0"],0,0,""]`

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r, err := parseRecording(items)
	if err != nil {
		t.Fatalf("parseRecording: %v", err)
	}

	if r.ID != "device-id" {
		t.Errorf("ID should fall back to deviceID, got %q", r.ID)
	}
}

func TestParseRecordingTooFewFields(t *testing.T) {
	raw := `["device-id","Title",["1706140050","0"]]`

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	_, err := parseRecording(items)
	if err == nil {
		t.Fatal("expected error for too few fields")
	}
}

func TestParseRecordingEmptyTitle(t *testing.T) {
	raw := `["device-id","",["1706140050","0"],["60","0"],0,0,""]`

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r, err := parseRecording(items)
	if err != nil {
		t.Fatalf("parseRecording: %v", err)
	}

	if r.Title == "" {
		t.Error("empty title should be replaced with formatted date")
	}
}

func TestParseRecordingWithLocation(t *testing.T) {
	raw := `["device-id","Meeting",["1706140050","0"],["60","0"],37.7749,-122.4194,"San Francisco",null,null,null,null,"",null,"bf3451e0-4ea6-424e-8e77-fbef4c0fe17c"]`

	var items []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	r, err := parseRecording(items)
	if err != nil {
		t.Fatalf("parseRecording: %v", err)
	}

	if r.Location != "San Francisco" {
		t.Errorf("Location = %q, want %q", r.Location, "San Francisco")
	}
}

func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	return &Client{
		auth: &auth.AuthData{
			SAPISID:  "test-sapisid",
			Cookies:  "SAPISID=test-sapisid",
			APIKey:   "test-key",
			AuthUser: 0,
		},
		http: server.Client(),
	}
}

func TestGetTranscriptParsing(t *testing.T) {
	response := `[[
		[
			[["hello","Hello","0","500"],["world","world.","500","1000"]],
			0,
			"en"
		],
		[
			[["goodbye","Goodbye","2000","2500"],["now","now.","2500","3000"]],
			1,
			"en"
		]
	]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	transcript, err := client.GetTranscript(context.Background(), "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c")
	if err != nil {
		t.Fatalf("GetTranscript: %v", err)
	}
	if transcript == nil {
		t.Fatal("expected transcript, got nil")
	}

	if len(transcript.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(transcript.Segments))
	}

	if transcript.Segments[0].Speaker != "Speaker 1" {
		t.Errorf("segment 0 speaker = %q, want %q", transcript.Segments[0].Speaker, "Speaker 1")
	}
	if transcript.Segments[0].Text != "Hello world." {
		t.Errorf("segment 0 text = %q, want %q", transcript.Segments[0].Text, "Hello world.")
	}
	if transcript.Segments[0].StartTime != "00:00" {
		t.Errorf("segment 0 startTime = %q, want %q", transcript.Segments[0].StartTime, "00:00")
	}

	if transcript.Segments[1].Speaker != "Speaker 2" {
		t.Errorf("segment 1 speaker = %q, want %q", transcript.Segments[1].Speaker, "Speaker 2")
	}
	if transcript.Segments[1].Text != "Goodbye now." {
		t.Errorf("segment 1 text = %q, want %q", transcript.Segments[1].Text, "Goodbye now.")
	}
	if transcript.Segments[1].StartTime != "00:02" {
		t.Errorf("segment 1 startTime = %q, want %q", transcript.Segments[1].StartTime, "00:02")
	}

	if !strings.Contains(transcript.RawText, "[Speaker 1]") {
		t.Error("RawText should contain speaker labels")
	}
}

func TestGetTranscriptSameSpeakerMerge(t *testing.T) {
	response := `[[
		[
			[["first","First","0","500"]],
			0,
			"en"
		],
		[
			[["second","second","500","1000"]],
			0,
			"en"
		]
	]]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	transcript, err := client.GetTranscript(context.Background(), "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c")
	if err != nil {
		t.Fatalf("GetTranscript: %v", err)
	}

	if len(transcript.Segments) != 1 {
		t.Fatalf("same speaker segments should merge, got %d segments", len(transcript.Segments))
	}
	if transcript.Segments[0].Text != "First second" {
		t.Errorf("merged text = %q, want %q", transcript.Segments[0].Text, "First second")
	}
}

func TestGetTranscriptEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	transcript, err := client.GetTranscript(context.Background(), "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c")
	if err != nil {
		t.Fatalf("GetTranscript: %v", err)
	}
	if transcript != nil {
		t.Errorf("expected nil for empty response, got %+v", transcript)
	}
}

func TestGetTranscriptAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`unauthorized`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	_, err := client.GetTranscript(context.Background(), "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain status code, got: %v", err)
	}
}

func TestGetTranscriptInvalidUUID(t *testing.T) {
	client := &Client{
		auth: &auth.AuthData{},
		http: http.DefaultClient,
	}
	_, err := client.GetTranscript(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestGetAudioStreaming(t *testing.T) {
	audioData := "fake-audio-content-for-test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Disposition", `attachment; filename="recording.m4a"`)
		w.Write([]byte(audioData))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origAudioBase := audioBase
	defer func() { setAudioBase(origAudioBase) }()
	setAudioBase(server.URL)

	var buf strings.Builder
	result, err := client.GetAudio(context.Background(), "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c", &buf)
	if err != nil {
		t.Fatalf("GetAudio: %v", err)
	}

	if buf.String() != audioData {
		t.Errorf("audio data = %q, want %q", buf.String(), audioData)
	}
	if result.Filename != "recording.m4a" {
		t.Errorf("filename = %q, want %q", result.Filename, "recording.m4a")
	}
	if result.ContentType != "audio/mp4" {
		t.Errorf("content type = %q, want %q", result.ContentType, "audio/mp4")
	}
	if result.BytesWritten != int64(len(audioData)) {
		t.Errorf("bytes written = %d, want %d", result.BytesWritten, len(audioData))
	}
}

func TestGetAudioNoContentDisposition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("audio"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origAudioBase := audioBase
	defer func() { setAudioBase(origAudioBase) }()
	setAudioBase(server.URL)

	result, err := client.GetAudio(context.Background(), "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c", io.Discard)
	if err != nil {
		t.Fatalf("GetAudio: %v", err)
	}

	if result.Filename != "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c.m4a" {
		t.Errorf("filename should fall back to ID.m4a, got %q", result.Filename)
	}
}

func disableRetryDelay(t *testing.T) {
	t.Helper()
	orig := retryBaseDelay
	retryBaseDelay = time.Millisecond
	t.Cleanup(func() { retryBaseDelay = orig })
}

func TestRetryOnServerError(t *testing.T) {
	disableRetryDelay(t)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(500)
			w.Write([]byte(`internal error`))
			return
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	_, err := client.ListRecordings(context.Background(), 5)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryExhausted(t *testing.T) {
	disableRetryDelay(t)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(503)
		w.Write([]byte(`unavailable`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	_, err := client.ListRecordings(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !strings.Contains(err.Error(), "after 3 retries") {
		t.Errorf("error should mention retries, got: %v", err)
	}
	if attempts != 4 {
		t.Errorf("expected 4 attempts (1 + 3 retries), got %d", attempts)
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(403)
		w.Write([]byte(`forbidden`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	_, err := client.ListRecordings(context.Background(), 5)
	if err == nil {
		t.Fatal("expected error on 403")
	}
	if attempts != 1 {
		t.Errorf("should not retry 403, got %d attempts", attempts)
	}
}

func TestRetryOn429(t *testing.T) {
	disableRetryDelay(t)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(429)
			w.Write([]byte(`rate limited`))
			return
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	_, err := client.ListRecordings(context.Background(), 5)
	if err != nil {
		t.Fatalf("expected success after 429 retry, got: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func makeRecordingJSON(id, title string, timestampSec int64) string {
	return fmt.Sprintf(`["%s","%s",["%d","0"],["60","0"],0,0,"",null,null,null,null,"",null,"%s"]`,
		id, title, timestampSec, id)
}

func setPageSize(t *testing.T, size int) {
	t.Helper()
	orig := defaultPageSize
	defaultPageSize = size
	t.Cleanup(func() { defaultPageSize = orig })
}

func TestPagination(t *testing.T) {
	setPageSize(t, 2)
	page1 := fmt.Sprintf(`[[%s,%s]]`,
		makeRecordingJSON("aaaaaaaa-1111-1111-1111-111111111111", "First", 1700000000),
		makeRecordingJSON("bbbbbbbb-2222-2222-2222-222222222222", "Second", 1699000000))
	page2 := fmt.Sprintf(`[[%s]]`,
		makeRecordingJSON("cccccccc-3333-3333-3333-333333333333", "Third", 1698000000))

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Write([]byte(page1))
		} else {
			w.Write([]byte(page2))
		}
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	recordings, err := client.ListRecordings(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}

	if len(recordings) != 3 {
		t.Fatalf("expected 3 recordings across 2 pages, got %d", len(recordings))
	}
	if recordings[0].Title != "First" || recordings[2].Title != "Third" {
		t.Errorf("unexpected order: %v", recordings)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestPaginationWithLimit(t *testing.T) {
	page := fmt.Sprintf(`[[%s,%s]]`,
		makeRecordingJSON("aaaaaaaa-1111-1111-1111-111111111111", "First", 1700000000),
		makeRecordingJSON("bbbbbbbb-2222-2222-2222-222222222222", "Second", 1699000000))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(page))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	recordings, err := client.ListRecordings(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}

	if len(recordings) != 1 {
		t.Fatalf("expected 1 recording with limit=1, got %d", len(recordings))
	}
}

func TestGetAudioRetry(t *testing.T) {
	disableRetryDelay(t)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(500)
			w.Write([]byte(`error`))
			return
		}
		w.Header().Set("Content-Type", "audio/mp4")
		w.Write([]byte("audio-data"))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origAudioBase := audioBase
	defer func() { setAudioBase(origAudioBase) }()
	setAudioBase(server.URL)

	var buf strings.Builder
	result, err := client.GetAudio(context.Background(), "bf3451e0-4ea6-424e-8e77-fbef4c0fe17c", &buf)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if buf.String() != "audio-data" {
		t.Errorf("audio data = %q", buf.String())
	}
	if result.BytesWritten != 10 {
		t.Errorf("bytes = %d, want 10", result.BytesWritten)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestListRecordingsRPCHeaders(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestClient(t, server)
	origBase := apiBase
	defer func() { setAPIBase(origBase) }()
	setAPIBase(server.URL)

	_, err := client.ListRecordings(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}

	if ct := capturedHeaders.Get("Content-Type"); ct != "application/json+protobuf" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json+protobuf")
	}
	if key := capturedHeaders.Get("X-Goog-Api-Key"); key != "test-key" {
		t.Errorf("X-Goog-Api-Key = %q, want %q", key, "test-key")
	}
	auth := capturedHeaders.Get("Authorization")
	if !strings.HasPrefix(auth, "SAPISIDHASH ") {
		t.Errorf("Authorization should start with SAPISIDHASH, got %q", auth)
	}
}
