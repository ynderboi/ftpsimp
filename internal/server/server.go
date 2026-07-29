package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed web/*
var webFS embed.FS

type Options struct {
	PIN      string
	AuthOn   bool
	ReadOnly bool
}

type Server struct {
	mu       sync.RWMutex
	root     string
	addr     string
	handler  http.Handler
	http     *http.Server
	onRoot   func(string) error
	pin      string
	authOn   bool
	readOnly bool
	sessions *sessionStore

	runMu   sync.Mutex
	running bool
}

type entry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

func New(root, addr string, onRoot func(string) error, opt Options) *Server {
	pin := strings.TrimSpace(opt.PIN)
	if opt.AuthOn && pin == "" {
		pin = generatePIN()
	}
	s := &Server{
		root:     root,
		addr:     addr,
		onRoot:   onRoot,
		pin:      pin,
		authOn:   opt.AuthOn,
		readOnly: opt.ReadOnly,
		sessions: newSessionStore(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/logout", s.handleLogout)
	mux.HandleFunc("/api/list", s.handleList)
	mux.HandleFunc("/api/download", s.handleDownload)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/mkdir", s.handleMkdir)
	mux.HandleFunc("/api/delete", s.handleDelete)
	mux.HandleFunc("/api/info", s.handleInfo)
	mux.HandleFunc("/api/settings", s.handleSettings)

	static, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(static)))

	s.handler = s.withMiddleware(mux)
	s.http = s.newHTTP()
	return s
}

func (s *Server) newHTTP() *http.Server {
	s.mu.RLock()
	addr := s.addr
	s.mu.RUnlock()
	return &http.Server{
		Addr:              addr,
		Handler:           s.handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

// Start begins serving in a background goroutine. Safe to call again after Stop.
func (s *Server) Start() error {
	s.runMu.Lock()
	if s.running {
		s.runMu.Unlock()
		return nil
	}
	hs := s.newHTTP()
	s.http = hs
	s.running = true
	s.runMu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		err := hs.ListenAndServe()
		s.runMu.Lock()
		if s.http == hs {
			s.running = false
		}
		s.runMu.Unlock()
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-time.After(150 * time.Millisecond):
		return nil
	}
}

// ListenAndServe starts the server and blocks (legacy). Prefer Start for TUI.
func (s *Server) ListenAndServe() error {
	s.runMu.Lock()
	s.http = s.newHTTP()
	s.running = true
	hs := s.http
	s.runMu.Unlock()
	err := hs.ListenAndServe()
	s.runMu.Lock()
	if s.http == hs {
		s.running = false
	}
	s.runMu.Unlock()
	return err
}

func (s *Server) Stop(ctx context.Context) error {
	s.runMu.Lock()
	hs := s.http
	running := s.running
	s.runMu.Unlock()
	if !running || hs == nil {
		return nil
	}
	err := hs.Shutdown(ctx)
	s.runMu.Lock()
	s.running = false
	s.runMu.Unlock()
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.Stop(ctx)
}

func (s *Server) Running() bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return s.running
}

func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

func (s *Server) PIN() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pin
}

func (s *Server) AuthOn() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authOn
}

func (s *Server) ReadOnly() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readOnly
}

func (s *Server) SessionCount() int {
	return s.sessions.count()
}

func (s *Server) SetReadOnly(v bool) {
	s.mu.Lock()
	s.readOnly = v
	s.mu.Unlock()
}

func (s *Server) SetAuth(on bool) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authOn = on
	if on && strings.TrimSpace(s.pin) == "" {
		s.pin = generatePIN()
	}
	if !on {
		s.sessions.clear()
	}
	return s.pin
}

func (s *Server) SetPIN(pin string) error {
	pin = strings.TrimSpace(pin)
	if pin == "" {
		return fmt.Errorf("empty pin")
	}
	s.mu.Lock()
	s.pin = pin
	s.mu.Unlock()
	s.sessions.clear()
	return nil
}

func (s *Server) RotatePIN() string {
	pin := generatePIN()
	s.mu.Lock()
	s.pin = pin
	s.authOn = true
	s.mu.Unlock()
	s.sessions.clear()
	return pin
}

func (s *Server) Root() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.root
}

func (s *Server) SetRoot(root string) error {
	abs, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("папка не найдена: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("путь не является папкой")
	}
	// Resolve symlinks so later checks stay inside the real root.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	s.mu.Lock()
	s.root = abs
	s.mu.Unlock()
	if s.onRoot != nil {
		if err := s.onRoot(abs); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		pathOnly := r.URL.Path
		if strings.HasPrefix(pathOnly, "/api/") {
			switch pathOnly {
			case "/api/status", "/api/login":
				// public
			case "/api/logout":
				// session optional
			default:
				if s.AuthOn() && !s.authenticated(r) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}
			if s.ReadOnly() && isWriteAPI(pathOnly, r.Method) {
				http.Error(w, "read-only mode", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func isWriteAPI(path, method string) bool {
	switch path {
	case "/api/upload", "/api/mkdir", "/api/delete":
		return true
	case "/api/settings":
		return method == http.MethodPost
	default:
		return false
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	authed := s.authenticated(r)
	out := map[string]any{
		"authRequired":  s.AuthOn(),
		"authenticated": authed,
		"readOnly":      s.ReadOnly(),
		"settingsLocal": true,
	}
	if authed {
		out["root"] = s.Root()
	}
	writeJSON(w, out)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.AuthOn() {
		writeJSON(w, map[string]any{"ok": true, "authRequired": false})
		return
	}
	var body struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if !pinEqual(body.PIN, s.PIN()) {
		time.Sleep(200 * time.Millisecond)
		http.Error(w, "invalid pin", http.StatusUnauthorized)
		return
	}
	id, exp := s.sessions.create()
	s.setSessionCookie(w, id, exp)
	writeJSON(w, map[string]any{
		"ok":       true,
		"token":    id,
		"root":     s.Root(),
		"readOnly": s.ReadOnly(),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if id := s.sessionID(r); id != "" {
		s.sessions.revoke(id)
	}
	s.clearSessionCookie(w)
	writeJSON(w, map[string]string{"ok": "1"})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"root":     s.Root(),
		"readOnly": s.ReadOnly(),
	})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, map[string]any{
			"root":          s.Root(),
			"readOnly":      s.ReadOnly(),
			"settingsLocal": true,
			"canChangeRoot": isLocalClient(r),
		})
	case http.MethodPost:
		if !isLocalClient(r) {
			http.Error(w, "смена корневой папки только с этого компьютера", http.StatusForbidden)
			return
		}
		var body struct {
			Root string `json:"root"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if err := s.SetRoot(body.Root); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"root": s.Root()})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolve(rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st, err := os.Stat(abs)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !st.IsDir() {
		http.Error(w, "not a directory", http.StatusBadRequest)
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]entry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		childRel := path.Join(cleanRel(rel), name)
		out = append(out, entry{
			Name:    name,
			Path:    childRel,
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	writeJSON(w, map[string]any{
		"path":    cleanRel(rel),
		"entries": out,
	})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.resolve(rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	name := filepath.Base(abs)
	ctype := mime.TypeByExtension(filepath.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", contentDisposition(name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", st.Size()))
	f, err := os.Open(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rel := r.URL.Query().Get("path")
	overwrite := r.URL.Query().Get("overwrite") == "1"
	abs, err := s.resolve(rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st, err := os.Stat(abs)
	if err != nil || !st.IsDir() {
		http.Error(w, "target folder not found", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 512<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid multipart: "+err.Error(), http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "no files", http.StatusBadRequest)
		return
	}
	saved := make([]string, 0, len(files))
	for _, fh := range files {
		src, err := fh.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		base := filepath.Base(fh.Filename)
		if base == "." || base == ".." || base == "" {
			src.Close()
			http.Error(w, "invalid filename", http.StatusBadRequest)
			return
		}
		dstPath := filepath.Join(abs, base)
		if !overwrite {
			if _, err := os.Stat(dstPath); err == nil {
				src.Close()
				http.Error(w, "file exists: "+base+" (use overwrite)", http.StatusConflict)
				return
			}
		}
		dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			src.Close()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			http.Error(w, copyErr.Error(), http.StatusInternalServerError)
			return
		}
		saved = append(saved, base)
	}
	writeJSON(w, map[string]any{"saved": saved})
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}
	parent, err := s.resolve(body.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.Mkdir(filepath.Join(parent, name), 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"ok": "1"})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if cleanRel(body.Path) == "" || body.Path == "." || body.Path == "/" {
		http.Error(w, "cannot delete root", http.StatusBadRequest)
		return
	}
	abs, err := s.resolve(body.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.RemoveAll(abs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"ok": "1"})
}

func (s *Server) resolve(rel string) (string, error) {
	rel = cleanRel(rel)
	root := s.Root()
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootAbs = resolved
	}
	abs := filepath.Join(rootAbs, filepath.FromSlash(rel))
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	// If path exists, resolve symlinks and re-check containment.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	relCheck, err := filepath.Rel(rootAbs, abs)
	if err != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside shared folder")
	}
	return abs, nil
}

func cleanRel(rel string) string {
	rel = strings.TrimSpace(rel)
	rel = strings.ReplaceAll(rel, `\`, `/`)
	rel = path.Clean("/" + rel)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." {
		return ""
	}
	return rel
}

func contentDisposition(name string) string {
	ascii := strings.Map(func(r rune) rune {
		if r < 32 || r == '"' || r == '\\' || r > 126 {
			return -1
		}
		return r
	}, name)
	if ascii == "" {
		ascii = "download"
	}
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", ascii, url.PathEscape(name))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}
