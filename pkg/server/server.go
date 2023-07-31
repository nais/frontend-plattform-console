package server

import (
	"context"
	"fmt"
	"os"

	fqdnV1alpha3 "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	"github.com/gin-gonic/gin"
	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/handler"
	"github.com/nais/bifrost/pkg/server/utils"
	"github.com/nais/bifrost/pkg/unleash"
	unleashv1 "github.com/nais/unleasherator/api/v1"
	"github.com/sirupsen/logrus"
	admin "google.golang.org/api/sqladmin/v1beta4"
	"k8s.io/apimachinery/pkg/runtime"
	client_go_scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func initGoogleClient(ctx context.Context) (*admin.Service, error) {
	googleClient, err := admin.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return googleClient, nil
}

func initUnleashSQLInstance(ctx context.Context, client *admin.Service, config *config.Config) (*admin.DatabaseInstance, error) {
	instance, err := client.Instances.Get(config.Google.ProjectID, config.Unleash.SQLInstanceID).Do()
	if err != nil {
		return nil, err
	}

	if instance.State != "RUNNABLE" {
		return instance, fmt.Errorf("instance state is not runnable")
	}

	return instance, nil
}

func initKubenetesClient() (ctrl.Client, error) {
	var kubeClient ctrl.Client
	schema := runtime.NewScheme()
	fqdnV1alpha3.AddToScheme(schema)
	unleashv1.AddToScheme(schema)
	client_go_scheme.AddToScheme(schema)
	opts := ctrl.Options{
		Scheme: schema,
	}
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}

		kubeClient, err = ctrl.New(config, opts)
		if err != nil {
			return nil, err
		}
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		kubeClient, err = ctrl.New(config, opts)
		if err != nil {
			return nil, err
		}
	}

	return kubeClient, nil
}

func initLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	return logger
}

func setupRouter(config *config.Config, logger *logrus.Logger, unleashService unleash.IUnleashService) *gin.Engine {
	router := gin.Default()
	gin.DefaultWriter = logger.Writer()

	h := handler.NewHandler(config, logger, unleashService)

	router.Use(h.ErrorHandler)
	router.Static("/assets", "./assets")

	router.HTMLRender = utils.LoadTemplates(config.Server.TemplatesDir)
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "Frontend Plattform",
		})
	})

	router.GET("/healthz", h.HealthHandler)

	unleash := router.Group("/unleash")
	{
		unleash.GET("/", h.UnleashIndex)
		unleash.GET("/new", h.UnleashNew)
		unleash.POST("/new", h.UnleashNewPost)

		unleashInstance := unleash.Group("/:id")
		unleashInstance.Use(h.UnleashInstanceMiddleware)
		{
			unleashInstance.GET("/", h.UnleashInstanceShow)
			unleashInstance.GET("/edit", h.UnleashInstanceEdit)
			unleashInstance.POST("/edit", h.UnleashInstanceEditPost)
			unleashInstance.GET("/delete", h.UnleashInstanceDelete)
			unleashInstance.POST("/delete", h.UnleashInstanceDeletePost)
		}
	}

	return router
}

func Run(config *config.Config) {
	logger := initLogger()

	kubeClient, err := initKubenetesClient()
	if err != nil {
		logger.Fatal(err)
	}

	googleClient, err := initGoogleClient(context.Background())
	if err != nil {
		logger.Fatal(err)
	}

	unleashInstance, err := initUnleashSQLInstance(context.Background(), googleClient, config)
	if err != nil {
		logger.Fatal(err)
	}

	unleashService := unleash.NewUnleashService(googleClient, kubeClient, unleashInstance, config, logger)

	router := setupRouter(config, logger, unleashService)

	logger.Infof("Listening on %s", config.GetServerAddr())
	if err := router.Run(config.GetServerAddr()); err != nil {
		logger.Fatal(err)
	}
}
