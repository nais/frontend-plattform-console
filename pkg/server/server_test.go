package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/unleash"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockUnleashService struct {
	Instances []*unleash.UnleashInstance
}

func (s *MockUnleashService) List(ctx context.Context) ([]*unleash.UnleashInstance, error) {
	return s.Instances, nil
}

func (s *MockUnleashService) Get(ctx context.Context, teamName string) (*unleash.UnleashInstance, error) {
	for _, instance := range s.Instances {
		if instance.TeamName == teamName {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("instance not found")
}

func (s *MockUnleashService) Create(ctx context.Context, teamName string) error {
	s.Instances = append(s.Instances, &unleash.UnleashInstance{
		TeamName:  teamName,
		CreatedAt: metav1.Now(),
	})

	return nil
}

func (s *MockUnleashService) Delete(ctx context.Context, teamName string) error {
	for i, instance := range s.Instances {
		if instance.TeamName == teamName {
			s.Instances = append(s.Instances[:i], s.Instances[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("instance not found")
}

func TestHealthzRoute(t *testing.T) {
	config := &config.Config{}
	logger := logrus.New()
	service := &MockUnleashService{}

	router := setupRouter(config, logger, service)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestMetricsRoute(t *testing.T) {
	t.Skip()

	config := &config.Config{}
	logger := logrus.New()
	service := &MockUnleashService{}

	router := setupRouter(config, logger, service)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "go_gc_duration_seconds")
}

func TestUnleashIndex(t *testing.T) {
	config := &config.Config{
		Server: config.ServerConfig{
			TemplatesDir: "../../templates",
		},
	}
	logger := logrus.New()
	service := &MockUnleashService{
		Instances: []*unleash.UnleashInstance{
			{
				TeamName:  "team1",
				CreatedAt: metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			{
				TeamName:  "team2",
				CreatedAt: metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
	}

	router := setupRouter(config, logger, service)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 301, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/unleash/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<a class=\"header\" href=\"team1\">team1</a>")
	assert.Contains(t, w.Body.String(), "<a class=\"header\" href=\"team2\">team2</a>")
}

func TestUnleashNew(t *testing.T) {
	config := &config.Config{
		Server: config.ServerConfig{
			TemplatesDir: "../../templates",
		},
	}
	logger := logrus.New()
	service := &MockUnleashService{
		Instances: []*unleash.UnleashInstance{},
	}

	router := setupRouter(config, logger, service)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash/new", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">New Unleash Instance</h1>")
	assert.Contains(t, w.Body.String(), "<form class=\"ui form\" method=\"POST\">")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/new", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "<p>Missing team name</p>")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/new", strings.NewReader("team-name=my-team"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "/unleash", w.Header().Get("Location"))
	assert.Equal(t, 1, len(service.Instances))
	assert.Equal(t, "my-team", service.Instances[0].TeamName)
}
