package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/baystation12/byond-go/byond"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
)

type Config struct {
	Host string `json:"host"`
}

type Status map[string]string

func (s Status) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func NewStatus() Status {
	return Status{"online": "false"}
}

/*func NewResponse(success bool, message interface{}) render.Renderer {
	return &Response{
		Success: success,
		Message: message,
	}
}*/

type Server struct {
	conf   *Config
	client *byond.QueryClient

	status Status
	mu     sync.RWMutex
}

func (s *Server) Start() {
	s.Update()

	clock := time.Tick(15 * time.Second)
	for _ = range clock {
		s.Update()
	}
}

func (s *Server) Update() {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	newStatus := NewStatus()

	queryResp, err := s.client.Query(ctx, []byte("status"), true)
	if err == nil && string(queryResp) != "ERROR" && string(queryResp) != "" {
		newStatus["online"] = "false"

		parsed, _ := url.ParseQuery(string(queryResp))
		for k, v := range parsed {
			newStatus[k] = v[0]
		}
	}

	s.mu.Lock()
	s.status = newStatus
	s.mu.Unlock()
}

func (s *Server) GetStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	render.Render(w, r, render.Renderer(s.status))
}

func main() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Printf("config read error: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err = json.Unmarshal(file, &config); err != nil {
		fmt.Printf("config parse error: %v\n", err)
		os.Exit(1)
	}

	client := byond.NewQueryClient(config.Host)

	server := &Server{
		conf:   &config,
		client: client,

		status: NewStatus(),
	}
	go server.Start()

	r := chi.NewRouter()

	cors := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(cors.Handler)

	r.Get("/status", server.GetStatus)

	http.ListenAndServe(":3888", r)
}
