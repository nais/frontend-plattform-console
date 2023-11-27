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

func (s *MockUnleashService) Create(ctx context.Context, uc *unleash.UnleashConfig) error {
	spec := unleash.UnleashDefinition(s.c, uc)

	s.Instances = append(s.Instances, &unleash.UnleashInstance{
		Name:           uc.Name,
		CreatedAt:      metav1.Now(),
		ServerInstance: &spec,
	})

	return nil
}

func (s *MockUnleashService) Update(ctx context.Context, uc *unleash.UnleashConfig) error {
	spec := unleash.UnleashDefinition(s.c, uc)

	for _, instance := range s.Instances {
		if instance.Name == uc.Name {
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

func unleashConfigToForm(uc *unleash.UnleashConfig) string {
	return fmt.Sprintf("name=%s&custom-version=%s&enable-federation=%t&allowed-teams=%s&allowed-namespaces=%s&allowed-clusters=%s&loglevel=%s&database-pool-max=%d&database-pool-idle-timeout-ms=%d",
		uc.Name,
		uc.CustomVersion,
		uc.EnableFederation,
		uc.AllowedTeams,
		uc.AllowedNamespaces,
		uc.AllowedClusters,
		uc.LogLevel,
		uc.DatabasePoolMax,
		uc.DatabasePoolIdleTimeoutMs,
	)
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

	unleash1 := unleash.UnleashDefinition(c, &unleash.UnleashConfig{
		Name:                      "team-a",
		CustomVersion:             "1.2.3",
		EnableFederation:          true,
		FederationNonce:           "abc123",
		AllowedTeams:              "team-a,team-b",
		AllowedNamespaces:         "ns-a,ns-b",
		AllowedClusters:           "cluster-a,cluster-b",
		LogLevel:                  "debug",
		DatabasePoolMax:           10,
		DatabasePoolIdleTimeoutMs: 100,
	})
	unleash2 := unleash.UnleashDefinition(c, &unleash.UnleashConfig{
		Name:                      "team-b",
		CustomVersion:             "",
		EnableFederation:          false,
		FederationNonce:           "",
		AllowedTeams:              "",
		AllowedNamespaces:         "",
		AllowedClusters:           "",
		LogLevel:                  "warn",
		DatabasePoolMax:           3,
		DatabasePoolIdleTimeoutMs: 1000,
	})

	logger := logrus.New()
	service = &MockUnleashService{
		c: c,
		Instances: []*unleash.UnleashInstance{
			{
				Name:           "team-a",
				CreatedAt:      metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
				ServerInstance: &unleash1,
			},
			{
				Name:           "team-b",
				CreatedAt:      metav1.NewTime(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
				ServerInstance: &unleash2,
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
	assert.Contains(t, w.Body.String(), "<a class=\"header\" href=\"team-a\">team-a</a>")
	assert.Contains(t, w.Body.String(), "<a class=\"header\" href=\"team-b\">team-b</a>")
}

func TestUnleashNew(t *testing.T) {
	_, service, router := newUnleashRoute()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash/new", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">New Unleash Instance</h1>")
	assert.Contains(t, w.Body.String(), "<form class=\"ui form\" method=\"POST\">")
	assert.Contains(t, w.Body.String(), "<input name=\"name\" type=\"text\" value=\"\">")
	// assert.Contains(t, w.Body.String(), "<input name=\"custom-version\" type=\"hidden\" value=\"\">")
	assert.Contains(t, w.Body.String(), "<input name=\"enable-federation\" type=\"checkbox\" value=\"on\" checked>")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-teams\" type=\"hidden\" value=\"\">")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-namespaces\" type=\"hidden\" value=\"\">")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-clusters\" type=\"hidden\" value=\"dev-gcp,prod-gcp\">")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/new", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Code)
	// assert.Contains(t, w.Body.String(), "<form class=\"ui form error\" method=\"POST\">")
	// assert.Contains(t, w.Body.String(), "<div class=\"name field error\">")
	// assert.Contains(t, w.Body.String(), "<p>Input validation failed, see errors in above fields</p>")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/new", strings.NewReader("name=my-name"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "/unleash/my-name", w.Header().Get("Location"))
	assert.Equal(t, 3, len(service.Instances))
	assert.Equal(t, "my-name", service.Instances[2].Name)
	assert.Equal(t, []string{}, service.Instances[2].ServerInstance.Spec.Federation.Clusters)
	assert.Equal(t, []string{}, service.Instances[2].ServerInstance.Spec.Federation.Namespaces)
	assert.Contains(t, service.Instances[2].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "TEAMS_ALLOWED_TEAMS", Value: ""})
	assert.Contains(t, service.Instances[2].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "LOG_LEVEL", Value: "warn"})
	assert.Contains(t, service.Instances[2].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "DATABASE_POOL_MAX", Value: "3"})
	assert.Contains(t, service.Instances[2].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "DATABASE_POOL_IDLE_TIMEOUT_MS", Value: "1000"})

	w = httptest.NewRecorder()
	uc := &unleash.UnleashConfig{
		Name:                      "my-name",
		CustomVersion:             "1.2.3",
		EnableFederation:          true,
		AllowedTeams:              "team-a,team-b",
		AllowedNamespaces:         "ns-a,ns-b",
		AllowedClusters:           "cluster-a,cluster-b",
		LogLevel:                  "debug",
		DatabasePoolMax:           10,
		DatabasePoolIdleTimeoutMs: 100,
	}

	req, _ = http.NewRequest("POST", "/unleash/new", strings.NewReader(unleashConfigToForm(uc)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "/unleash/my-name", w.Header().Get("Location"))
	assert.Equal(t, 4, len(service.Instances))
	assert.Equal(t, "my-name", service.Instances[3].Name)
	assert.Equal(t, "europe-north1-docker.pkg.dev/nais-io/nais/images/unleash-v4:1.2.3", service.Instances[3].ServerInstance.Spec.CustomImage)
	assert.Equal(t, true, service.Instances[3].ServerInstance.Spec.Federation.Enabled, true)
	assert.Equal(t, []string{"cluster-a", "cluster-b"}, service.Instances[3].ServerInstance.Spec.Federation.Clusters)
	assert.Equal(t, []string{"ns-a", "ns-b"}, service.Instances[3].ServerInstance.Spec.Federation.Namespaces)
	assert.Contains(t, service.Instances[3].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "TEAMS_ALLOWED_TEAMS", Value: "team-a,team-b"})
	assert.Contains(t, service.Instances[3].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "LOG_LEVEL", Value: "debug"})
	assert.Contains(t, service.Instances[3].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "DATABASE_POOL_MAX", Value: "10"})
	assert.Contains(t, service.Instances[3].ServerInstance.Spec.ExtraEnvVars, v1.EnvVar{Name: "DATABASE_POOL_IDLE_TIMEOUT_MS", Value: "100"})
}

func TestUnleashEdit(t *testing.T) {
	_, _, router := newUnleashRoute()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash/team-a/edit", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">Edit Unleash: team-a</h1>")
	assert.Contains(t, w.Body.String(), "<form class=\"ui form\" method=\"POST\">")
	assert.Contains(t, w.Body.String(), "<input name=\"name\" type=\"text\" disabled value=\"team-a\">")
	assert.Contains(t, w.Body.String(), "<input name=\"custom-version\" type=\"hidden\" value=\"1.2.3\">")
	assert.Contains(t, w.Body.String(), "<input name=\"enable-federation\" type=\"checkbox\" value=\"on\" checked>")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-teams\" type=\"hidden\" value=\"team-a,team-b\">")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-namespaces\" type=\"hidden\" value=\"ns-a,ns-b\">")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-clusters\" type=\"hidden\" value=\"cluster-a,cluster-b\">")
	assert.Contains(t, w.Body.String(), "<input type=\"radio\" name=\"loglevel\" value=\"debug\" checked=\"checked\" tabindex=\"0\" class=\"hidden\">")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/unleash/team-b/edit", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">Edit Unleash: team-b</h1>")
	assert.Contains(t, w.Body.String(), "<form class=\"ui form\" method=\"POST\">")
	assert.Contains(t, w.Body.String(), "<input name=\"name\" type=\"text\" disabled value=\"team-b\">")
	assert.Contains(t, w.Body.String(), "<input name=\"custom-version\" type=\"hidden\" value=\"\">")
	assert.Contains(t, w.Body.String(), "<input name=\"enable-federation\" type=\"checkbox\" value=\"on\">")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-teams\" type=\"hidden\" value=\"\">")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-namespaces\" type=\"hidden\" value=\"\">")
	assert.Contains(t, w.Body.String(), "<input name=\"allowed-clusters\" type=\"hidden\" value=\"\">")
	assert.Contains(t, w.Body.String(), "<input type=\"radio\" name=\"loglevel\" value=\"warn\" checked=\"checked\" tabindex=\"0\" class=\"hidden\">")
}

func TestUnleashGet(t *testing.T) {
	_, _, router := newUnleashRoute()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash/does-not-exist/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 301, w.Code)
	assert.Equal(t, "/unleash?status=not-found", w.Header().Get("Location"))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/unleash/team-a", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 301, w.Code)
	assert.Equal(t, "/unleash/team-a/", w.Header().Get("Location"))

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/unleash/team-a/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">Unleash: team-a</h1>")
}

func TestUnleashDelete(t *testing.T) {
	_, service, router := newUnleashRoute()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/unleash/team-a/delete", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "<h1 class=\"ui header\">Delete Unleash: team-a</h1>")
	assert.Contains(t, w.Body.String(), "<form class=\"ui form\" method=\"POST\">")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/team-a/delete", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "Instance name does not match")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/unleash/team-a/delete", strings.NewReader("name=team-a"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)
	assert.Equal(t, 302, w.Code)
	assert.Equal(t, "/unleash", w.Header().Get("Location"))
	assert.Equal(t, 1, len(service.Instances))
}
