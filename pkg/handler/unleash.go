package handler

import (
	"html/template"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/nais/bifrost/pkg/unleash"
)

func (h *Handler) HealthHandler(c *gin.Context) {
	c.String(200, "OK")
}

func (h *Handler) ErrorHandler(c *gin.Context) {
	c.Next()

	errorToPrint := c.Errors.ByType(gin.ErrorTypePublic).Last()
	if errorToPrint != nil {
		h.logger.WithError(errorToPrint.Err).Error(errorToPrint.Meta)
		c.HTML(500, "error.html", gin.H{
			"title": "Error",
			"error": errorToPrint.Meta,
		})
	}
}

func (h *Handler) UnleashNewPost(c *gin.Context) {
	teamName := regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(c.PostForm("team-name"), "")
	ctx := c.Request.Context()
	if teamName == "" {
		c.HTML(400, "unleash-form.html", gin.H{
			"title": "New Unleash Instance",
			"error": "Missing team name",
		})
		return
	}

	err := unleash.CreateInstance(ctx, h.googleClient, h.sqlInstance, teamName, h.config, h.kubeClient)
	if err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error creating unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}

func (h *Handler) UnleashIndex(c *gin.Context) {
	ctx := c.Request.Context()
	instances, err := unleash.GetInstances(ctx, h.googleClient, h.sqlInstance, h.config.Unleash.InstanceNamespace)
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

func (h *Handler) UnleashNew(c *gin.Context) {
	c.HTML(200, "unleash-form.html", gin.H{
		"title":  "New Unleash Instance",
		"action": "create",
	})
}

func (h *Handler) UnleashInstanceMiddleware(c *gin.Context) {
	databaseName := c.Param("id")
	ctx := c.Request.Context()

	instance, err := unleash.GetInstance(ctx, h.googleClient, h.sqlInstance, databaseName, h.config.Unleash.InstanceNamespace)
	if err != nil {
		h.logger.Info(err)
		c.Redirect(404, "/unleash?status=not-found")
		c.Abort()
		return
	}

	c.Set("unleashInstance", &instance)
	c.Next()
}

func (h *Handler) UnleashInstanceShow(c *gin.Context) {
	ctx := c.Request.Context()

	instance := c.MustGet("unleashInstance").(*unleash.Unleash)
	err := instance.GetDatabaseUser(ctx, h.googleClient)
	if err != nil {
		h.logger.WithError(err).Errorf("Error getting database user for instance %s", instance.Database.Name)
	}

	c.HTML(200, "unleash-show.html", gin.H{
		"title":    "Unleash: " + instance.TeamName,
		"instance": instance,
	})
}

func (h *Handler) UnleashInstanceDelete(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.Unleash)
	c.HTML(200, "unleash-form.html", gin.H{
		"title":  "Delete Unleash: " + instance.TeamName,
		"action": "delete",
	})
}

func (h *Handler) UnleashInstanceDeletePost(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.Unleash)

	ctx := c.Request.Context()
	teamName := regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(c.PostForm("team-name"), "")

	if teamName != instance.TeamName {
		c.HTML(400, "unleash-form.html", gin.H{
			"title":  "Delete Unleash: " + instance.TeamName,
			"action": "delete",
			"error":  "Missing confirmation",
		})
		return
	}

	if err := instance.Delete(ctx, h.googleClient, h.kubeClient); err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error deleting unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}
