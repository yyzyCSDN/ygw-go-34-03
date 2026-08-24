package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"confighub"
	"confighub/internal/audit"
	"confighub/internal/checkpoint"
	"confighub/internal/client"
	"confighub/internal/metric"
	"confighub/internal/model"
	"confighub/internal/publish"
	"confighub/internal/store"
	"confighub/internal/version"
	"confighub/internal/watch"
)

type Server struct {
	cfg       Config
	st        *store.Store
	versions  *version.Table
	grader    *publish.Grader
	journal   *audit.Journal
	metrics   *metric.Metrics
	cursors   *checkpoint.Cursor
	registry  *watch.Registry
	hub       *watch.Hub
	publisher *publish.Publisher
	client    *client.Client
	evictor   *watch.Evictor
	httpSrv   *http.Server
}

func NewServer(cfg Config) *Server {
	st := store.New()
	versions := version.New()
	grader := publish.NewGrader()
	journal := audit.New()
	metrics := metric.New()
	cursors := checkpoint.NewCursor()
	registry := watch.NewRegistry()
	hub := watch.NewHub(registry)
	publisher := publish.New(st, versions, hub, journal, metrics, grader, cfg.AckTimeout, cfg.MaxRetries)
	cache := client.NewCache()
	cl := client.New(st, versions, grader, cache)
	evictor := watch.NewEvictor(registry, cfg.EvictEvery, cfg.IdleAfter, metrics)
	return &Server{
		cfg:       cfg,
		st:        st,
		versions:  versions,
		grader:    grader,
		journal:   journal,
		metrics:   metrics,
		cursors:   cursors,
		registry:  registry,
		hub:       hub,
		publisher: publisher,
		client:    cl,
		evictor:   evictor,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/api/pull", s.handlePull)
	mux.HandleFunc("/api/delta", s.handleDelta)
	mux.HandleFunc("/api/publish", s.handlePublish)
	mux.HandleFunc("/api/rollback", s.handleRollback)
	mux.HandleFunc("/api/plans", s.handlePlans)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/api/key", s.handleKey)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/watch", s.handleWatch)
	mux.HandleFunc("/", s.handleConsole)
	return mux
}

func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	s.httpSrv = &http.Server{Handler: s.Handler()}
	go s.evictor.Run(ctx)
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpSrv.Serve(ln)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return s.httpSrv.Shutdown(shutCtx)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "ok")
}

func (s *Server) handleConsole(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(web.ConsoleHTML)
}

func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	app := model.AppID(r.URL.Query().Get("app"))
	group := model.GroupID(r.URL.Query().Get("group"))
	etag := r.URL.Query().Get("etag")
	snap, notModified := s.client.Pull(app, group, etag)
	s.metrics.RecordPull()
	s.metrics.RecordLatency(time.Since(start))
	_ = writeJSON(w, map[string]any{
		"not_modified": notModified,
		"revision":     snap.Revision,
		"batch_id":     snap.BatchID,
		"checksum":     snap.Checksum,
		"entries":      snap.Entries,
	})
}

func (s *Server) handleDelta(w http.ResponseWriter, r *http.Request) {
	app := model.AppID(r.URL.Query().Get("app"))
	group := model.GroupID(r.URL.Query().Get("group"))
	from, err := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
	if err != nil {
		http.Error(w, "from is required", http.StatusBadRequest)
		return
	}
	records := s.client.PullDelta(app, group, from)
	_ = writeJSON(w, map[string]any{
		"records": records,
	})
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		App         string            `json:"app"`
		Group       string            `json:"group"`
		Entries     map[string]string `json:"entries"`
		DeleteKeys  []string          `json:"delete_keys"`
		GrayPercent *int              `json:"gray_percent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.GrayPercent != nil {
		if err := publish.ValidatePercent(*req.GrayPercent); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		plan := &model.GrayPlan{
			App:              model.AppID(req.App),
			Group:            model.GroupID(req.Group),
			Percent:          *req.GrayPercent,
			CapturedRevision: s.st.Revision(),
			CreatedAt:        time.Now().UTC(),
		}
		s.grader.Set(plan)
	}
	entries := make(map[model.Key]string, len(req.Entries))
	for k, v := range req.Entries {
		entries[model.Key(k)] = v
	}
	deleteKeys := make([]model.Key, 0, len(req.DeleteKeys))
	for _, k := range req.DeleteKeys {
		deleteKeys = append(deleteKeys, model.Key(k))
	}
	b := model.NewBatch(model.AppID(req.App), model.GroupID(req.Group), entries)
	b.DeleteKeys = deleteKeys
	res := s.publisher.Publish(r.Context(), &b)
	if res.Error != nil {
		s.metrics.RecordError()
		http.Error(w, res.Error.Error(), http.StatusBadGateway)
		return
	}
	_ = writeJSON(w, res)
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_ = writeJSON(w, map[string]any{"plans": s.grader.List()})
	case http.MethodDelete:
		app := model.AppID(r.URL.Query().Get("app"))
		group := model.GroupID(r.URL.Query().Get("group"))
		s.grader.Remove(app, group)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	app := model.AppID(r.URL.Query().Get("app"))
	group := model.GroupID(r.URL.Query().Get("group"))
	_ = writeJSON(w, s.st.Export(app, group))
}

func (s *Server) handleKey(w http.ResponseWriter, r *http.Request) {
	key := model.Key(r.URL.Query().Get("key"))
	entry, err := s.st.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_ = writeJSON(w, entry)
}

func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		App      string `json:"app"`
		Group    string `json:"group"`
		Revision int64  `json:"revision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res := s.publisher.Rollback(r.Context(), model.AppID(req.App), model.GroupID(req.Group), req.Revision)
	if res.Error != nil {
		s.metrics.RecordError()
		http.Error(w, res.Error.Error(), http.StatusBadGateway)
		return
	}
	_ = writeJSON(w, res)
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	appliedTotal := int64(0)
	attemptTotal := int64(0)
	watermark := int64(0)
	for _, c := range s.registry.Snapshot() {
		appliedTotal += c.AppliedCount()
		attemptTotal += c.Attempts()
		if cur := c.Cursor(); cur > watermark {
			watermark = cur
		}
	}
	_ = writeJSON(w, map[string]any{
		"revision":        s.st.Revision(),
		"batch_id":        s.st.BatchID(),
		"entry_count":     s.st.EntryCount(),
		"connections":     s.registry.Count(),
		"sessions":        s.cursors.Count(),
		"session_cursors": s.cursors.Snapshot(),
		"version_latest":  s.versions.Latest(),
		"applied_total":   appliedTotal,
		"attempt_total":   attemptTotal,
		"watermark":       watermark,
		"metrics":         s.metrics.Snapshot(),
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	since := int64(0)
	if raw := r.URL.Query().Get("since"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			since = v
		}
	}
	var records []audit.Record
	if since > 0 {
		records = s.journal.Since(since)
	} else {
		records = s.journal.Tail(20)
	}
	_ = writeJSON(w, map[string]any{
		"audit":           records,
		"audit_count":     s.journal.Count(),
		"latest_revision": s.versions.Latest(),
	})
}

func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	app := model.AppID(r.URL.Query().Get("app"))
	group := model.GroupID(r.URL.Query().Get("group"))
	session := r.URL.Query().Get("session")
	if session == "" {
		session = uuid.NewString()
	}
	id := session + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	conn, err := s.client.Watch(r.Context(), id, session, app, group, s.cursors, s.hub, s.registry, func(ev model.Event) error {
		_ = s.client.ApplyEvent(ev)
		data, _ := json.Marshal(ev)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Kind.String(), data)
		flusher.Flush()
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("watch %s session %s connected", conn.ID(), conn.Session())
	defer func() {
		s.registry.Unregister(conn)
		s.metrics.RecordEviction()
	}()
	<-r.Context().Done()
}

func writeJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}
