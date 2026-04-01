package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ conn *sql.DB }

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	conn, err := sql.Open("sqlite", filepath.Join(dataDir, "spur.db"))
	if err != nil {
		return nil, err
	}
	conn.Exec("PRAGMA journal_mode=WAL")
	conn.Exec("PRAGMA busy_timeout=5000")
	conn.SetMaxOpenConns(4)
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error { return db.conn.Close() }

func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    base_path TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS endpoints (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    method TEXT DEFAULT 'GET',
    path TEXT NOT NULL,
    status_code INTEGER DEFAULT 200,
    response_body TEXT DEFAULT '{}',
    response_headers TEXT DEFAULT '{"Content-Type":"application/json"}',
    delay_ms INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_ep_project ON endpoints(project_id);
CREATE INDEX IF NOT EXISTS idx_ep_path ON endpoints(method, path);

CREATE TABLE IF NOT EXISTS request_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    endpoint_id TEXT NOT NULL,
    method TEXT DEFAULT '',
    path TEXT DEFAULT '',
    headers_json TEXT DEFAULT '{}',
    body TEXT DEFAULT '',
    source_ip TEXT DEFAULT '',
    timestamp TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_rlog_ep ON request_log(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_rlog_time ON request_log(timestamp);
`)
	return err
}

// --- Projects ---

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	BasePath  string `json:"base_path"`
	CreatedAt string `json:"created_at"`
	Endpoints int    `json:"endpoint_count"`
}

func (db *DB) CreateProject(name, basePath string) (*Project, error) {
	id := "prj_" + genID(6)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec("INSERT INTO projects (id,name,base_path,created_at) VALUES (?,?,?,?)", id, name, basePath, now)
	if err != nil {
		return nil, err
	}
	return &Project{ID: id, Name: name, BasePath: basePath, CreatedAt: now}, nil
}

func (db *DB) ListProjects() ([]Project, error) {
	rows, err := db.conn.Query(`SELECT p.id, p.name, p.base_path, p.created_at,
		(SELECT COUNT(*) FROM endpoints WHERE project_id=p.id)
		FROM projects p ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		rows.Scan(&p.ID, &p.Name, &p.BasePath, &p.CreatedAt, &p.Endpoints)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (db *DB) GetProject(id string) (*Project, error) {
	var p Project
	err := db.conn.QueryRow(`SELECT p.id, p.name, p.base_path, p.created_at,
		(SELECT COUNT(*) FROM endpoints WHERE project_id=p.id)
		FROM projects p WHERE p.id=?`, id).
		Scan(&p.ID, &p.Name, &p.BasePath, &p.CreatedAt, &p.Endpoints)
	return &p, err
}

func (db *DB) DeleteProject(id string) error {
	db.conn.Exec("DELETE FROM endpoints WHERE project_id=?", id)
	_, err := db.conn.Exec("DELETE FROM projects WHERE id=?", id)
	return err
}

// --- Endpoints ---

type Endpoint struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	Method          string `json:"method"`
	Path            string `json:"path"`
	StatusCode      int    `json:"status_code"`
	ResponseBody    string `json:"response_body"`
	ResponseHeaders string `json:"response_headers"`
	DelayMs         int    `json:"delay_ms"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       string `json:"created_at"`
}

func (db *DB) CreateEndpoint(projectID, method, path string, statusCode int, responseBody, responseHeaders string, delayMs int) (*Endpoint, error) {
	id := "ep_" + genID(8)
	now := time.Now().UTC().Format(time.RFC3339)
	if method == "" {
		method = "GET"
	}
	if statusCode <= 0 {
		statusCode = 200
	}
	if responseBody == "" {
		responseBody = "{}"
	}
	if responseHeaders == "" {
		responseHeaders = `{"Content-Type":"application/json"}`
	}
	_, err := db.conn.Exec("INSERT INTO endpoints (id,project_id,method,path,status_code,response_body,response_headers,delay_ms,created_at) VALUES (?,?,?,?,?,?,?,?,?)",
		id, projectID, method, path, statusCode, responseBody, responseHeaders, delayMs, now)
	if err != nil {
		return nil, err
	}
	return &Endpoint{ID: id, ProjectID: projectID, Method: method, Path: path, StatusCode: statusCode,
		ResponseBody: responseBody, ResponseHeaders: responseHeaders, DelayMs: delayMs, Enabled: true, CreatedAt: now}, nil
}

func (db *DB) ListEndpoints(projectID string) ([]Endpoint, error) {
	rows, err := db.conn.Query("SELECT id,project_id,method,path,status_code,response_body,response_headers,delay_ms,enabled,created_at FROM endpoints WHERE project_id=? ORDER BY path, method", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Endpoint
	for rows.Next() {
		var e Endpoint
		var en int
		rows.Scan(&e.ID, &e.ProjectID, &e.Method, &e.Path, &e.StatusCode, &e.ResponseBody, &e.ResponseHeaders, &e.DelayMs, &en, &e.CreatedAt)
		e.Enabled = en == 1
		out = append(out, e)
	}
	return out, rows.Err()
}

func (db *DB) GetEndpoint(id string) (*Endpoint, error) {
	var e Endpoint
	var en int
	err := db.conn.QueryRow("SELECT id,project_id,method,path,status_code,response_body,response_headers,delay_ms,enabled,created_at FROM endpoints WHERE id=?", id).
		Scan(&e.ID, &e.ProjectID, &e.Method, &e.Path, &e.StatusCode, &e.ResponseBody, &e.ResponseHeaders, &e.DelayMs, &en, &e.CreatedAt)
	e.Enabled = en == 1
	return &e, err
}

// MatchEndpoint finds an endpoint by method+path across all projects
func (db *DB) MatchEndpoint(method, path string) (*Endpoint, error) {
	var e Endpoint
	var en int
	err := db.conn.QueryRow("SELECT id,project_id,method,path,status_code,response_body,response_headers,delay_ms,enabled,created_at FROM endpoints WHERE method=? AND path=? AND enabled=1 LIMIT 1",
		method, path).
		Scan(&e.ID, &e.ProjectID, &e.Method, &e.Path, &e.StatusCode, &e.ResponseBody, &e.ResponseHeaders, &e.DelayMs, &en, &e.CreatedAt)
	e.Enabled = en == 1
	return &e, err
}

func (db *DB) UpdateEndpoint(id string, statusCode *int, responseBody, responseHeaders *string, delayMs *int, enabled *bool) (*Endpoint, error) {
	if statusCode != nil {
		db.conn.Exec("UPDATE endpoints SET status_code=? WHERE id=?", *statusCode, id)
	}
	if responseBody != nil {
		db.conn.Exec("UPDATE endpoints SET response_body=? WHERE id=?", *responseBody, id)
	}
	if responseHeaders != nil {
		db.conn.Exec("UPDATE endpoints SET response_headers=? WHERE id=?", *responseHeaders, id)
	}
	if delayMs != nil {
		db.conn.Exec("UPDATE endpoints SET delay_ms=? WHERE id=?", *delayMs, id)
	}
	if enabled != nil {
		en := 0
		if *enabled {
			en = 1
		}
		db.conn.Exec("UPDATE endpoints SET enabled=? WHERE id=?", en, id)
	}
	return db.GetEndpoint(id)
}

func (db *DB) DeleteEndpoint(id string) error {
	db.conn.Exec("DELETE FROM request_log WHERE endpoint_id=?", id)
	_, err := db.conn.Exec("DELETE FROM endpoints WHERE id=?", id)
	return err
}

func (db *DB) TotalEndpoints() int {
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM endpoints").Scan(&count)
	return count
}

// --- Request log ---

type RequestEntry struct {
	ID          int    `json:"id"`
	EndpointID  string `json:"endpoint_id"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	HeadersJSON string `json:"headers"`
	Body        string `json:"body"`
	SourceIP    string `json:"source_ip"`
	Timestamp   string `json:"timestamp"`
}

func (db *DB) LogRequest(endpointID, method, path, headersJSON, body, ip string) {
	db.conn.Exec("INSERT INTO request_log (endpoint_id,method,path,headers_json,body,source_ip) VALUES (?,?,?,?,?,?)",
		endpointID, method, path, headersJSON, body, ip)
}

func (db *DB) ListRequestLog(endpointID string, limit int) ([]RequestEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query("SELECT id,endpoint_id,method,path,headers_json,body,source_ip,timestamp FROM request_log WHERE endpoint_id=? ORDER BY timestamp DESC LIMIT ?", endpointID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RequestEntry
	for rows.Next() {
		var r RequestEntry
		rows.Scan(&r.ID, &r.EndpointID, &r.Method, &r.Path, &r.HeadersJSON, &r.Body, &r.SourceIP, &r.Timestamp)
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Stats ---

func (db *DB) Stats() map[string]any {
	var projects, endpoints, requests int
	db.conn.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projects)
	db.conn.QueryRow("SELECT COUNT(*) FROM endpoints").Scan(&endpoints)
	db.conn.QueryRow("SELECT COUNT(*) FROM request_log").Scan(&requests)
	return map[string]any{"projects": projects, "endpoints": endpoints, "requests_logged": requests}
}

func (db *DB) Cleanup(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	res, err := db.conn.Exec("DELETE FROM request_log WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func genID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
