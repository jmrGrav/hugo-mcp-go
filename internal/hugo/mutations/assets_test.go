package mutations

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadAssetNominal(t *testing.T) {
	svc := NewAssetService(newTestWorkspace(t))
	raw := mustLoadUploadAssetSource(t, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "upload_asset.after", "static", "images", "oracle", "oracle-phase2.svg.json"))
	got, err := svc.UploadAsset(context.Background(), UploadAssetRequest{
		Filename:  "oracle-phase2.svg",
		Data:      base64.StdEncoding.EncodeToString(raw),
		Subfolder: "images/oracle",
	})
	if err != nil {
		t.Fatalf("UploadAsset() error = %v", err)
	}
	assertUploadAssetResult(t, got, filepath.Join("..", "..", "..", "testdata", "fixtures", "oracle_phase2", "upload_asset.response.json"))
	path := filepath.Join(svc.Stage.StaticRoot, "images", "oracle", "oracle-phase2.svg")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("uploaded file missing: %v", err)
	}
}

func TestUploadAssetRejectsInvalidExtension(t *testing.T) {
	svc := NewAssetService(newTestWorkspace(t))
	_, err := svc.UploadAsset(context.Background(), UploadAssetRequest{
		Filename:  "tool.exe",
		Data:      base64.StdEncoding.EncodeToString([]byte("abc")),
		Subfolder: "images",
	})
	if err == nil {
		t.Fatal("UploadAsset() expected error")
	}
	if got, want := err.Error(), `Unsupported extension ".exe". Allowed: ['.gif', '.jpeg', '.jpg', '.png', '.svg', '.webp']`; got != want {
		t.Fatalf("UploadAsset() error = %q want %q", got, want)
	}
}

func TestUploadAssetRejectsTraversal(t *testing.T) {
	svc := NewAssetService(newTestWorkspace(t))
	_, err := svc.UploadAsset(context.Background(), UploadAssetRequest{
		Filename:  "oracle-phase2.svg",
		Data:      base64.StdEncoding.EncodeToString([]byte("abc")),
		Subfolder: "../escape",
	})
	if err == nil {
		t.Fatal("UploadAsset() expected error")
	}
}

func TestUploadAssetRejectsSymlinkEscape(t *testing.T) {
	svc := NewAssetService(newTestWorkspace(t))
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(svc.Stage.StaticRoot, "images", "escape")); err != nil {
		t.Fatal(err)
	}
	_, err := svc.UploadAsset(context.Background(), UploadAssetRequest{
		Filename:  "oracle-phase2.svg",
		Data:      base64.StdEncoding.EncodeToString([]byte("abc")),
		Subfolder: "images/escape",
	})
	if err == nil {
		t.Fatal("UploadAsset() expected symlink error")
	}
}

func TestUploadAssetRejectsSizeLimit(t *testing.T) {
	svc := NewAssetService(newTestWorkspace(t))
	svc.MaxUploadBytes = 4
	_, err := svc.UploadAsset(context.Background(), UploadAssetRequest{
		Filename:  "oracle-phase2.svg",
		Data:      base64.StdEncoding.EncodeToString([]byte("abcdef")),
		Subfolder: "images",
	})
	if err == nil {
		t.Fatal("UploadAsset() expected size limit error")
	}
}

func TestUploadAssetRejectsEncodedSizeBeforeDecode(t *testing.T) {
	svc := NewAssetService(newTestWorkspace(t))
	svc.MaxUploadBytes = 4
	_, err := svc.UploadAsset(context.Background(), UploadAssetRequest{
		Filename:  "oracle-phase2.svg",
		Data:      strings.Repeat("!", 1024),
		Subfolder: "images",
	})
	if err == nil {
		t.Fatal("UploadAsset() expected size limit error")
	}
	if got, want := err.Error(), "File too large: decoded size exceeds max 4 bytes"; got != want {
		t.Fatalf("UploadAsset() error = %q want %q", got, want)
	}
}

func assertUploadAssetResult(t *testing.T, got UploadAssetResult, snapshotPath string) {
	t.Helper()
	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	var wantResult map[string]any
	if err := json.Unmarshal([]byte(want.Content[0].Text), &wantResult); err != nil {
		t.Fatal(err)
	}
	gotRaw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var gotResult map[string]any
	if err := json.Unmarshal(gotRaw, &gotResult); err != nil {
		t.Fatal(err)
	}
	if !equalJSON(t, gotResult, wantResult) {
		t.Fatalf("UploadAsset() mismatch:\n got=%s\nwant=%s", string(gotRaw), want.Content[0].Text)
	}
}

func mustLoadUploadAssetSource(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var snap struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	return []byte(snap.Content)
}
