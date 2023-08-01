package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/unleash"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockUnleashService struct {
	c         *config.Config
	Instances []*unleash.UnleashInstance
}

func (s *MockUnleashService) List(ctx context.Context) ([]*unleash.UnleashInstance, error) {
	return s.Instances, nil
}

func (s *MockUnleashService) Get(ctx context.Context, name string) (*unleash.UnleashInstance, error) {
	for _, instance := range s.Instances {
		if instance.Name == name {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("instance not found")
}

func (s *MockUnleashService) Create(ctx context.Context, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters string) error {
	spec := unleash.UnleashDefinition(s.c, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters)

	s.Instances = append(s.Instances, &unleash.UnleashInstance{
		Name:           name,
		CreatedAt:      metav1.Now(),
		ServerInstance: &spec,
	})

	return nil
}

func (s *MockUnleashService) Update(ctx context.Context, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters string) error {
	spec := unleash.UnleashDefinition(s.c, name, customVersion, allowedTeams, allowedNamespaces, allowedClusters)

	for _, instance := range s.Instances {
		if instance.Name == name {
			instance.ServerInstance = &spec
			return nil
		}
	}

	return fmt.Errorf("instance not found")
}

func (s *MockUnleashService) Delete(ctx context.Context, name string) error {
	for i, instance := range s.Instances {
		if instance.Name == name {
			s.Instances = append(s.Instances[:i], s.Instances[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("instance not found")
}

func TestHealthzRoute(t *testing.T) {
	config := &config.Config{}
	logger := logrus.New()
	service := &MockUnleashService{c: config}

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
	service := &MockUnleashService{c: config}

	router := setupRouter(config, logger, service)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "go_gc_duration_seconds")
}

func newUnleashRoute() (c *config.Config, service *MockUnleashService, router *gin.Engine) {
	c = &config.Config{
		Server: config.ServerConfig{
			TemplatesDir: "../../templates",
		},
	}
	logger := logrus.New()
	service = &MockUnleashService{
		c: c,
		Instances: []*unleash.UnleashInstance{
			{
				Name:           "team1",
				CreatedAt:      metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
				ServerInstance: &unleashv1.Unleash{},
			},
			{
				Name:           "team2",
				CreatedAt:      metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
				ServerInstance: &unleashv1.Unleash{},
			},
		},
	}

	router = setupRouter(c, logger, service)

	return
}

func TestUnleashIndex(t *testing.T) {
	_, _, router := newUnleashRoute()

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
	_, service, router := newUnleashRoute()

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
	assert.Contains(t, w.Body.String(), "<form class=\"ui form error\" method=\"POST\">")
	assert.Contains(t, w.Body.String(), "<div class=\"name field error\">")
	assert.Contains(t, w.Body.String(), "<p>Input validation failed, see errors in above fields</p>")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/new", strings.NewReader("name=my-name"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "/unleash/my-name", w.Header().Get("Location"))
	assert.Equal(t, 3, len(service.Instances))
	assert.Equal(t, "my-name", service.Instances[2].Name)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/new", strings.NewReader("name=my-name&custom-image-version=1.2.3&allowed-teams=team1,team2&allowed-namespaces=ns1,ns2&allowed-clusters=cluster1,cluster2"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "/unleash/my-name", w.Header().Get("Location"))
	assert.Equal(t, 4, len(service.Instances))
	assert.Equal(t, "my-name", service.Instances[3].Name)
	assert.Equal(t, "europe-north1-docker.pkg.dev/nais-io/nais/images/unleash-v4:1.2.3", service.Instances[3].ServerInstance.Spec.CustomImage)
	assert.Contains(t, service.Instances[3].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "TEAMS_ALLOWED_TEAMS", Value: "team1,team2"})
	assert.Contains(t, service.Instances[3].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "TEAMS_ALLOWED_NAMESPACES", Value: "ns1,ns2"})
	assert.Contains(t, service.Instances[3].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "TEAMS_ALLOWED_CLUSTERS", Value: "cluster1,cluster2"})
}

func TestUnleashGet(t *testing.T) {
	_, _, router := newUnleashRoute()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash/does-not-exist/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 301, w.Code)
	assert.Equal(t, "/unleash?status=not-found", w.Header().Get("Location"))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/unleash/team1", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 301, w.Code)
	assert.Equal(t, "/unleash/team1/", w.Header().Get("Location"))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/unleash/team1/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">Unleash: team1</h1>")
}

func TestUnleashDelete(t *testing.T) {
	c := &config.Config{
		Server: config.ServerConfig{
			TemplatesDir: "../../templates",
		},
	}
	logger := logrus.New()
	service := &MockUnleashService{
		c: c,
		Instances: []*unleash.UnleashInstance{
			{
				Name:      "team1",
				CreatedAt: metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			{
				Name:      "team2",
				CreatedAt: metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
	}

	router := setupRouter(c, logger, service)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash/team1/delete", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">Delete Unleash: team1</h1>")
	assert.Contains(t, w.Body.String(), "<form class=\"ui form\" method=\"POST\">")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/team1/delete", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "Instance name does not match")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/team1/delete", strings.NewReader("name=team1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "/unleash", w.Header().Get("Location"))
	assert.Equal(t, 1, len(service.Instances))
}
