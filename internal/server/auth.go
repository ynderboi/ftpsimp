package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookie = "ftpsimp_sess"
	sessionTTL    = 24 * time.Hour
)

type sessionStore struct {
	mu   sync.Mutex
	byID map[string]time.Time
}

func newSessionStore() *sessionStore {
	return &sessionStore{byID: make(map[string]time.Time)}
}

func (s *sessionStore) create() (string, time.Time) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	exp := time.Now().Add(sessionTTL)
	s.mu.Lock()
	s.byID[id] = exp
	s.mu.Unlock()
	return id, exp
}

func (s *sessionStore) valid(id string) bool {
	if id == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.byID[id]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.byID, id)
		return false
	}
	return true
}

func (s *sessionStore) revoke(id string) {
	s.mu.Lock()
	delete(s.byID, id)
	s.mu.Unlock()
}

func (s *sessionStore) clear() {
	s.mu.Lock()
	s.byID = make(map[string]time.Time)
	s.mu.Unlock()
}

func (s *sessionStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	n := 0
	for id, exp := range s.byID {
		if now.After(exp) {
			delete(s.byID, id)
			continue
		}
		n++
	}
	return n
}

func generatePIN() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "482910"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func pinEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if len(a) != len(b) {
		// Constant-time-ish length pad to avoid trivial leak; still compare.
		max := len(a)
		if len(b) > max {
			max = len(b)
		}
		aa := make([]byte, max)
		bb := make([]byte, max)
		copy(aa, a)
		copy(bb, b)
		return subtle.ConstantTimeCompare(aa, bb) == 1 && len(a) == len(b)
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func clientIP(r *http.Request) net.IP {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = h
	}
	return net.ParseIP(host)
}

// isLocalClient reports whether the request comes from this machine
// (loopback or any of this host's own interface addresses).
// Opening http://192.168.x.x from the host PC counts as local;
// a phone on Wi‑Fi does not.
func isLocalClient(r *http.Request) bool {
	ip := clientIP(r)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		var local net.IP
		switch v := a.(type) {
		case *net.IPNet:
			local = v.IP
		case *net.IPAddr:
			local = v.IP
		}
		if local != nil && local.Equal(ip) {
			return true
		}
	}
	return false
}

func (s *Server) sessionID(r *http.Request) string {
	if ah := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(ah), "bearer ") {
		return strings.TrimSpace(ah[7:])
	}
	if t := strings.TrimSpace(r.Header.Get("X-Session-Token")); t != "" {
		return t
	}
	if t := strings.TrimSpace(r.URL.Query().Get("token")); t != "" {
		return t
	}
	if c, err := r.Cookie(sessionCookie); err == nil && c != nil {
		return c.Value
	}
	return ""
}

func (s *Server) authenticated(r *http.Request) bool {
	if !s.AuthOn() {
		return true
	}
	return s.sessions.valid(s.sessionID(r))
}

func (s *Server) setSessionCookie(w http.ResponseWriter, id string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    id,
		Path:     "/",
		Expires:  exp,
		MaxAge:   int(time.Until(exp).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
