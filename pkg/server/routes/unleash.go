package routes

import (
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/unleash"
	"github.com/sirupsen/logrus"
	admin "google.golang.org/api/sqladmin/v1beta4"
)

func UnleashIndex(c *gin.Context) {
	ctx := c.Request.Context()

	config := c.MustGet("config").(*config.Config)
	googleClient := c.MustGet("googleClient").(*admin.Service)
	SQLInstance := c.MustGet("unleashSQLInstance").(*admin.DatabaseInstance)

	instances, err := unleash.GetInstances(ctx, googleClient, SQLInstance, config.Unleash.InstanceNamespace)
	if err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error getting unleash instances")
		return
	}

	status := template.HTMLEscapeString(c.Query("status"))
	c.HTML(200, "unleash-index.html", gin.H{
		"title":     "Unleash as a Service (UaaS))",
		"instances": instances,
		"status":    status,
	})
}

func UnleashNew(c *gin.Context) {
	c.HTML(200, "unleash-form.html", gin.H{
		"title":  "New Unleash Instance",
		"action": "create",
	})
}

func UnleashNewPost(c *gin.Context) {
	teamName := regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(c.PostForm("team-name"), "")
	ctx := c.Request.Context()

	config := c.MustGet("config").(*config.Config)
	googleClient := c.MustGet("googleClient").(*admin.Service)
	kubeClient := c.MustGet("kubeClient").(ctrl.Client)
	SQLInstance := c.MustGet("unleashSQLInstance").(*admin.DatabaseInstance)

	if teamName == "" {
		c.HTML(400, "unleash-form.html", gin.H{
			"title": "New Unleash Instance",
			"error": "Missing team name",
		})
		return
	}

	err := unleash.CreateInstance(ctx, googleClient, SQLInstance, teamName, config, kubeClient)
	if err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error creating unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}

func UnleashInstanceMiddleware(c *gin.Context) {
	databaseName := c.Param("id")
	ctx := c.Request.Context()

	log := c.MustGet("log").(*logrus.Logger)
	config := c.MustGet("config").(*config.Config)
	googleClient := c.MustGet("googleClient").(*admin.Service)
	SQLInstance := c.MustGet("unleashSQLInstance").(*admin.DatabaseInstance)

	instance, err := unleash.GetInstance(ctx, googleClient, SQLInstance, databaseName, config.Unleash.InstanceNamespace)
	if err != nil {
		log.Info(err)
		c.Redirect(404, "/unleash?status=not-found")
		c.Abort()
		return
	}

	c.Set("unleashInstance", &instance)
	c.Next()
}

func UnleashInstanceShow(c *gin.Context) {
	ctx := c.Request.Context()
	log := c.MustGet("log").(*logrus.Logger)
	googleClient := c.MustGet("googleClient").(*admin.Service)

	instance := c.MustGet("unleashInstance").(*unleash.Unleash)
	err := instance.GetDatabaseUser(ctx, googleClient)
	if err != nil {
		log.WithError(err).Errorf("Error getting database user for instance %s", instance.Database.Name)
	}

	c.HTML(200, "unleash-show.html", gin.H{
		"title":    "Unleash: " + instance.TeamName,
		"instance": instance,
	})
}

func UnleashInstanceDelete(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.Unleash)

	c.HTML(200, "unleash-form.html", gin.H{
		"title":  "Delete Unleash: " + instance.TeamName,
		"action": "delete",
	})
}

func UnleashInstanceDeletePost(c *gin.Context) {
	ctx := c.Request.Context()
	teamName := regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(c.PostForm("team-name"), "")
	kubeClient := c.MustGet("kubeClient").(ctrl.Client)
	googleClient := c.MustGet("googleClient").(*admin.Service)
	instance := c.MustGet("unleashInstance").(*unleash.Unleash)

	if teamName != instance.TeamName {
		c.HTML(400, "unleash-form.html", gin.H{
			"title":  "Delete Unleash: " + instance.TeamName,
			"action": "delete",
			"error":  "Missing confirmation",
		})
		return
	}

	if err := instance.Delete(ctx, googleClient, kubeClient); err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error deleting unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}
