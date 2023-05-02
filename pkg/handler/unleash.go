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

	if err := h.unleashService.Create(ctx, teamName); err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error creating unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}

func (h *Handler) UnleashIndex(c *gin.Context) {
	ctx := c.Request.Context()
	instances, err := h.unleashService.List(ctx)
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
	teamName := c.Param("id")
	ctx := c.Request.Context()

	// @TODO check if user is allowed to access this instance

	instance, err := h.unleashService.Get(ctx, teamName)
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
	// ctx := c.Request.Context()

	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)

	// @TODO get more info about the instance

	c.HTML(200, "unleash-show.html", gin.H{
		"title":    "Unleash: " + instance.TeamName,
		"instance": instance,
	})
}

func (h *Handler) UnleashInstanceDelete(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)
	c.HTML(200, "unleash-form.html", gin.H{
		"title":  "Delete Unleash: " + instance.TeamName,
		"action": "delete",
	})
}

func (h *Handler) UnleashInstanceDeletePost(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)

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

	if err := h.unleashService.Delete(ctx, instance.TeamName); err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error deleting unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}
