package hooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

var createSchemaFunc = createSchema
var execSchemaStatement = func(ctx context.Context, db *sql.DB, stmt string) (sql.Result, error) {
	return db.ExecContext(ctx, stmt)
}

type HookJob struct {
	ID         string
	Provider   string
	Action     string
	TargetURLs []string
	Status     string
	LastError  string
}

type AuditRecord struct {
	JobID    string
	Provider string
	Action   string
	Message  string
}

func OpenStore(path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("missing hooks database path")
	}
	if err := osMkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := createSchemaFunc(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) HasTable(ctx context.Context, table string) bool {
	row := s.db.QueryRowContext(ctx, `SELECT 1 FROM sqlite_master WHERE type='table' AND name = ?`, table)
	var one int
	return row.Scan(&one) == nil
}

func (s *Store) HasColumn(ctx context.Context, table, column string) bool {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

func (s *Store) Enqueue(ctx context.Context, job HookJob) (string, error) {
	if s == nil || s.db == nil {
		return "", errors.New("store not initialized")
	}
	if strings.TrimSpace(job.Provider) == "" {
		return "", errors.New("missing provider")
	}
	if len(job.TargetURLs) == 0 {
		return "", errors.New("missing target urls")
	}
	targetURLsJSON, err := json.Marshal(job.TargetURLs)
	if err != nil {
		return "", err
	}
	id := newID()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	status := job.Status
	if status == "" {
		status = "pending"
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO hook_jobs (id, provider, action, target_urls, status, last_error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, job.Provider, job.Action, string(targetURLsJSON), status, job.LastError, now, now)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) ListJobs(ctx context.Context) ([]HookJob, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("store not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, provider, action, target_urls, status, last_error
FROM hook_jobs
ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []HookJob
	for rows.Next() {
		var id, provider, action, targetURLsJSON, status, lastError string
		if err := rows.Scan(&id, &provider, &action, &targetURLsJSON, &status, &lastError); err != nil {
			return nil, err
		}
		var targetURLs []string
		_ = json.Unmarshal([]byte(targetURLsJSON), &targetURLs)
		jobs = append(jobs, HookJob{
			ID:         id,
			Provider:   provider,
			Action:     action,
			TargetURLs: targetURLs,
			Status:     status,
			LastError:  lastError,
		})
	}
	return jobs, nil
}

func (s *Store) SetJobStatus(ctx context.Context, ids []string, status string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("store not initialized")
	}
	if len(ids) == 0 {
		return 0, nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var total int64
	for _, id := range ids {
		res, err := s.db.ExecContext(ctx, `UPDATE hook_jobs SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
		if err != nil {
			return total, err
		}
		affected, _ := res.RowsAffected()
		total += affected
	}
	return total, nil
}

func (s *Store) RecordAudit(ctx context.Context, rec AuditRecord) error {
	if s == nil || s.db == nil {
		return errors.New("store not initialized")
	}
	if strings.TrimSpace(rec.Action) == "" {
		return errors.New("missing action")
	}
	if strings.TrimSpace(rec.Message) == "" {
		return errors.New("missing message")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO hook_audit (id, job_id, provider, action, message, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		newID(), rec.JobID, rec.Provider, rec.Action, rec.Message, now)
	return err
}

func (s *Store) JobCount(ctx context.Context) int {
	if s == nil || s.db == nil {
		return 0
	}
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hook_jobs`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) AuditMessages(ctx context.Context) []string {
	if s == nil || s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT message FROM hook_audit ORDER BY created_at, id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			return nil
		}
		out = append(out, message)
	}
	return out
}

func createSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS hook_jobs (
			id TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			action TEXT,
			target_urls TEXT NOT NULL,
			status TEXT NOT NULL,
			last_error TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS hook_attempts (
			id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			status TEXT NOT NULL,
			error TEXT,
			attempt_no INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS hook_provider_state (
			provider TEXT PRIMARY KEY,
			last_success_at TEXT,
			last_error TEXT,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS hook_audit (
			id TEXT PRIMARY KEY,
			job_id TEXT,
			provider TEXT,
			action TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := execSchemaStatement(ctx, db, stmt); err != nil {
			return err
		}
	}
	return nil
}
