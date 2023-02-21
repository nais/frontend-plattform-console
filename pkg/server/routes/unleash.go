package routes

import (
	"fmt"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/nais/frontend-plattform-console/pkg/config"
	"github.com/nais/frontend-plattform-console/pkg/unleash"
)

func UnleashIndex(c *gin.Context) {
	ctx := c.Request.Context()
	config := c.MustGet("config").(*config.Config)
	projectID := config.Google.ProjectID
	instanceID := config.Google.SQLInstanceID

	instances, err := unleash.GetInstances(ctx, projectID, instanceID)
	if err != nil {
		fmt.Printf("Error getting unleash instances: %v", err)
		c.Error(err).Meta = "Error getting unleash instances"
		return
	}

	fmt.Printf("instances: %v", instances)

	status := template.HTMLEscapeString(c.Query("status"))
	c.HTML(200, "unleash-index.html", gin.H{
		"title":     "Unleash as a Service (UaaS))",
		"instances": instances,
		"status":    status,
	})
}

func UnleashNew(c *gin.Context) {
	c.HTML(200, "unleash-form.html", gin.H{
		"title": "New Unleash Instance",
	})
}

func UnleashNewPost(c *gin.Context) {
	teamName := c.PostForm("team-name")
	ctx := c.Request.Context()
	config := c.MustGet("config").(*config.Config)
	projectID := config.Google.ProjectID
	instanceID := config.Google.SQLInstanceID

	if teamName == "" {
		c.HTML(400, "unleash-form.html", gin.H{
			"title": "New Unleash Instance",
			"error": "Missing team name",
		})
		return
	}

	_, err := unleash.CreateInstance(ctx, projectID, instanceID, teamName)
	if err != nil {
		fmt.Printf("Error creating unleash instance: %v", err)
		c.Error(err).Meta = "Error creating unleash instance"
		return
	}

	c.Redirect(302, "/unleash")
}

func UnleashInstanceMiddleware(c *gin.Context) {
	databaseID := c.Param("id")
	ctx := c.Request.Context()
	config := c.MustGet("config").(*config.Config)
	projectID := config.Google.ProjectID
	instanceID := config.Google.SQLInstanceID

	instance, err := unleash.GetInstance(ctx, projectID, instanceID, databaseID)
	if err != nil {
		fmt.Printf("Error getting unleash instance: %v", err)
		c.Error(err).SetType(gin.ErrorTypePublic).SetMeta("Error getting unleash instance")
		return
	}

	c.Set("instance", &instance)
	c.Next()
}

func UnleashInstanceShow(c *gin.Context) {
	instance := c.MustGet("instance").(*unleash.Unleash)

	c.HTML(200, "unleash-show.html", gin.H{
		"title":    "Unleash: " + instance.DatabaseName,
		"instance": instance,
	})
}

func UnleashInstanceDelete(c *gin.Context) {
	instance := c.MustGet("instance").(*unleash.Unleash)

	c.HTML(200, "unleash-delete.html", gin.H{
		"title":    "Delete Unleash: " + instance.DatabaseName,
		"instance": instance,
	})
}

func UnleashInstanceDeletePost(c *gin.Context) {
	ctx := c.Request.Context()
	instance := c.MustGet("instance").(*unleash.Unleash)
	confirm := c.PostForm("confirm")

	fmt.Printf("confirm: %v", confirm)
	return

	err := unleash.DeleteInstance(ctx, instance.ProjectId, instance.Instance, instance.DatabaseName)
	if err != nil {
		fmt.Printf("Error deleting unleash instance: %v", err)
		c.Error(err).Meta = "Error deleting unleash instance"
		return
	}

	c.Redirect(302, "/unleash")
}
