package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"layerone-scout/internal/app"
	"layerone-scout/internal/config"
	"layerone-scout/internal/model"
)

type Options struct {
	Addr         string
	Token        string
	RateLimit    int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func Run(cfg config.Config, app *app.App) error {
	opts := Options{
		Addr:         cfg.Server.Addr,
		Token:        cfg.Server.Token,
		RateLimit:    cfg.Server.RateLimit,
		ReadTimeout:  parseDuration(cfg.Server.ReadTimeout, 10*time.Second),
		WriteTimeout: parseDuration(cfg.Server.WriteTimeout, 10*time.Second),
		IdleTimeout:  parseDuration(cfg.Server.IdleTimeout, 30*time.Second),
	}

	mux := http.NewServeMux()
	rl := newRateLimiter(opts.RateLimit)
	handler := withLogging(withRateLimit(rl, withAuth(opts.Token, mux)))

	mux.HandleFunc("/", dashboard)
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/readyz", readyz)

	mux.HandleFunc("/api/persons", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			list, err := app.ListPersons()
			if err != nil {
				httpError(w, err, http.StatusInternalServerError)
				return
			}
			page, size := paginate(r)
			if size > 0 {
				start := (page - 1) * size
				if start < len(list) {
					end := start + size
					if end > len(list) {
						end = len(list)
					}
					list = list[start:end]
				} else {
					list = []model.Person{}
				}
			}
			writeJSON(w, list)
		case http.MethodPost:
			var in struct {
				Name      string `json:"name"`
				Username  string `json:"username"`
				Platform  string `json:"platform"`
				Bio       string `json:"bio"`
				Followers int    `json:"followers"`
				Following int    `json:"following"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				httpError(w, fmt.Errorf("body inválido"), http.StatusBadRequest)
				return
			}
			p, err := app.AddPerson(in.Name, in.Username, in.Platform, in.Bio, in.Followers, in.Following)
			if err != nil {
				httpError(w, err, http.StatusInternalServerError)
				return
			}
			writeJSON(w, p)
		case http.MethodDelete:
			personID := r.URL.Query().Get("person")
			if strings.TrimSpace(personID) == "" {
				http.Error(w, "falta person", http.StatusBadRequest)
				return
			}
			deleted, err := app.DeletePerson(personID)
			if err != nil {
				httpError(w, err, http.StatusNotFound)
				return
			}
			writeJSON(w, deleted)
		default:
			http.Error(w, "método no soportado", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/persons/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "método no soportado", http.StatusMethodNotAllowed)
			return
		}
		var in struct {
			Platform string `json:"platform"`
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "body inválido", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		p, err := app.FetchAndAddProfile(ctx, in.Platform, in.Username)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, p)
	})

	mux.HandleFunc("/api/persons/analyze", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "método no soportado", http.StatusMethodNotAllowed)
			return
		}
		var in struct{ Person string `json:"person"` }
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "body inválido", http.StatusBadRequest)
			return
		}
		p, err := app.AnalyzePerson(in.Person)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, p)
	})

	mux.HandleFunc("/api/persons/report", func(w http.ResponseWriter, r *http.Request) {
		personID := r.URL.Query().Get("person")
		if strings.TrimSpace(personID) == "" {
			http.Error(w, "falta person", http.StatusBadRequest)
			return
		}
		report, err := app.ReportPerson(personID)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "text/plain; charset=utf-8")
		fmt.Fprint(w, report)
	})

	srv := &http.Server{
		Addr:              opts.Addr,
		Handler:           handler,
		ReadTimeout:       opts.ReadTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      opts.WriteTimeout,
		IdleTimeout:       opts.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Servidor escuchando en %s\n", opts.Addr)
	if opts.Token == "" {
		fmt.Println("⚠️  Advertencia: API sin autenticación (token vacío)")
	}

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	fmt.Fprint(w, dashboardHTML)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func readyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ready")
}

func withAuth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if got != token {
			http.Error(w, "no autorizado", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRateLimit(rl *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			http.Error(w, "demasiadas requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		fmt.Printf("[%s] %s %s %s\n", time.Now().Format("2006-01-02 15:04:05"), r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		fmt.Printf("[%s] completado en %v\n", time.Now().Format("2006-01-02 15:04:05"), time.Since(start))
	})
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func httpError(w http.ResponseWriter, err error, code int) {
	http.Error(w, err.Error(), code)
}

func paginate(r *http.Request) (page, size int) {
	pageStr := r.URL.Query().Get("page")
	sizeStr := r.URL.Query().Get("size")
	page = 1
	size = 0
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}
	if page < 1 {
		page = 1
	}
	if size < 0 {
		size = 0
	}
	return
}

func parseDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	perMin  float64
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newRateLimiter(perMinute int) *rateLimiter {
	if perMinute <= 0 {
		perMinute = 120
	}
	return &rateLimiter{buckets: map[string]*bucket{}, perMin: float64(perMinute)}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.perMin, last: now}
		rl.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Minutes()
	b.tokens += elapsed * rl.perMin
	if b.tokens > rl.perMin {
		b.tokens = rl.perMin
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

const dashboardHTML = `<!doctype html>
<html>
<head><title>LayerOne Scout</title></head>
<body>
<h1>LayerOne Scout - Motor de perfilado psicológico</h1>
<p>Usa la API REST:</p>
<ul>
<li><code>GET /api/persons</code> - listar perfiles (con paginación: ?page=1&size=10)</li>
<li><code>POST /api/persons</code> - crear perfil manual</li>
<li><code>DELETE /api/persons?person=id</code> - eliminar perfil</li>
<li><code>POST /api/persons/fetch</code> - obtener perfil público (body: {"platform":"instagram","username":"..."})</li>
<li><code>POST /api/persons/analyze</code> - analizar perfil (body: {"person":"id"})</li>
<li><code>GET /api/persons/report?person=id</code> - obtener informe en texto</li>
</ul>
</body>
</html>`
