package server

import (
	"context"
	"fmt"
	"os"

	fqdnV1alpha3 "github.com/GoogleCloudPlatform/gke-fqdnnetworkpolicies-golang/api/v1alpha3"
	"github.com/gin-gonic/gin"
	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/server/routes"
	"github.com/nais/bifrost/pkg/server/utils"
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
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	return log
}

func Run(config *config.Config) {
	router := gin.Default()

	log := initLogger()
	gin.DefaultWriter = log.Writer()

	kubeClient, err := initKubenetesClient()
	if err != nil {
		log.Fatal(err)
	}

	googleClient, err := initGoogleClient(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	unleashInstance, err := initUnleashSQLInstance(context.Background(), googleClient, config)
	if err != nil {
		log.Fatal(err)
	}

	router.Use(func(c *gin.Context) {
		c.Set("config", config)
		c.Set("log", log)
		c.Set("kubeClient", kubeClient)
		c.Set("googleClient", googleClient)
		c.Set("unleashSQLInstance", unleashInstance)
		c.Next()
	})

	router.Use(routes.ErrorHandler)
	router.Static("/assets", "./assets")

	router.HTMLRender = utils.LoadTemplates("./templates")
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "Frontend Plattform",
		})
	})

	router.GET("/healthz", routes.HealthHandler)

	unleash := router.Group("/unleash")
	{
		unleash.GET("/", routes.UnleashIndex)
		unleash.GET("/new", routes.UnleashNew)
		unleash.POST("/new", routes.UnleashNewPost)

		unleashInstance := unleash.Group("/:id")
		unleashInstance.Use(routes.UnleashInstanceMiddleware)
		{
			unleashInstance.GET("/", routes.UnleashInstanceShow)
			unleashInstance.GET("/delete", routes.UnleashInstanceDelete)
			unleashInstance.POST("/delete", routes.UnleashInstanceDeletePost)
		}
	}

	fmt.Printf("Listening on %s", config.GetServerAddr())
	if err := router.Run(config.GetServerAddr()); err != nil {
		log.Fatal(err)
	}
}
