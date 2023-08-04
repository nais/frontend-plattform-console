package handler

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nais/bifrost/pkg/github"
	"github.com/nais/bifrost/pkg/unleash"
	"github.com/nais/bifrost/pkg/utils"
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
	unleashVersions, err := github.UnleashVersions()
	if err != nil {
		h.logger.WithError(err).Error("Error getting Unleash versions from Github")
		unleashVersions = []github.UnleashVersion{}
	}

	obj := unleash.UnleashDefinition(h.config, &unleash.UnleashConfig{Name: "my-unleash"})
	yamlString, err := utils.StructToYaml(obj)
	if err != nil {
		h.logger.WithError(err).Error("Error converting Unleash struct to yaml")
		yamlString = "Parse error - see logs"
	}

	c.HTML(200, "unleash-form.html", gin.H{
		"title":           "New Unleash Instance",
		"action":          "create",
		"customImageName": unleash.UnleashCustomImageName,
		"unleashVersions": unleashVersions,
		"logLevel":        "warn",
		"yaml":            yamlString,
	})
}

func (h *Handler) UnleashInstanceMiddleware(c *gin.Context) {
	teamName := c.Param("id")
	ctx := c.Request.Context()

	// @TODO check if user is allowed to access this instance

	instance, err := h.unleashService.Get(ctx, teamName)
	if err != nil {
		h.logger.Info(err)
		c.Redirect(301, "/unleash?status=not-found")
		c.Abort()
		return
	}

	c.Set("unleashInstance", instance)
	c.Next()
}

func splitNoEmpty(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	res := strings.Split(s, sep)
	for i := 0; i < len(res); i++ {
		if res[i] == "" {
			res = append(res[:i], res[i+1:]...)
		}
	}

	return res
}

func (h *Handler) UnleashInstanceShow(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)
	instanceYaml, err := utils.StructToYaml(instance.ServerInstance)
	if err != nil {
		h.logger.WithError(err).Error("Error converting Unleash struct to yaml")
		instanceYaml = "Parse error - see logs"
	}

	uc := unleash.UnleashVariables(instance.ServerInstance, false)

	c.HTML(200, "unleash-show.html", gin.H{
		"title":                    "Unleash: " + instance.Name,
		"instance":                 instance,
		"unleashCustomVersion":     uc.CustomVersion,
		"unleashAllowedTeams":      splitNoEmpty(uc.AllowedTeams, ","),
		"unleashAllowedNamespaces": splitNoEmpty(uc.AllowedNamespaces, ","),
		"unleashAllowedClusters":   splitNoEmpty(uc.AllowedClusters, ","),
		"unleashLogLevel":          uc.LogLevel,
		"googleProjectID":          h.config.Google.ProjectID,
		"sqlInstanceID":            h.config.Unleash.SQLInstanceID,
		"sqlInstanceAddress":       h.config.Unleash.SQLInstanceAddress,
		"sqlInstanceRegion":        h.config.Unleash.SQLInstanceRegion,
		"sqlDatabaseName":          instance.Name,
		"sqlDatabaseUser":          instance.Name,
		"sqlDatabaseSecret":        instance.Name,

		"instanceYaml": template.HTML(instanceYaml),
	})
}

func (h *Handler) UnleashInstanceEdit(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)

	uc := unleash.UnleashVariables(instance.ServerInstance, true)

	unleashVersions, err := github.UnleashVersions()
	if err != nil {
		h.logger.WithError(err).Error("Error getting Unleash versions from Github")
		unleashVersions = []github.UnleashVersion{}
	}

	c.HTML(200, "unleash-form.html", gin.H{
		"title":             "Edit Unleash: " + instance.Name,
		"action":            "edit",
		"name":              uc.Name,
		"customImageName":   unleash.UnleashCustomImageName,
		"customVersion":     uc.CustomVersion,
		"unleashVersions":   unleashVersions,
		"allowedTeams":      uc.AllowedTeams,
		"allowedNamespaces": uc.AllowedNamespaces,
		"allowedClusters":   uc.AllowedClusters,
		"logLevel":          uc.LogLevel,
	})
}

func (h *Handler) UnleashInstancePost(c *gin.Context) {
	var (
		name, title, action string
		err                 error
	)

	ctx := c.Request.Context()

	nameValidator := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	versionValidator := regexp.MustCompile(`^[a-zA-Z0-9-_\.+]*$`)
	listValidator := regexp.MustCompile(`^[a-zA-Z0-9-,]*$`)
	loglevelValidator := regexp.MustCompile(`^(debug|info|warn|error|fatal|panic)$`)

	instance, exists := c.Get("unleashInstance")
	if exists {
		instance, ok := instance.(*unleash.UnleashInstance)
		if !ok {
			c.Error(fmt.Errorf("could not convert instance to UnleashInstance"))
			return
		}

		name = instance.Name
		title = "Edit Unleash: " + name
		action = "edit"
	} else {
		name = c.PostForm("name")
		title = "New Unleash Instance"
		action = "create"
	}

	uc := &unleash.UnleashConfig{
		Name:              name,
		CustomVersion:     c.PostForm("custom-version"),
		AllowedTeams:      c.PostForm("allowed-teams"),
		AllowedNamespaces: c.PostForm("allowed-namespaces"),
		AllowedClusters:   c.PostForm("allowed-clusters"),
		LogLevel:          c.PostForm("loglevel"),
	}

	if uc.LogLevel == "" {
		uc.LogLevel = "warn"
	}

	nameError := !nameValidator.MatchString(name)
	customVersionError := !versionValidator.MatchString(uc.CustomVersion)
	allowedTeamsError := !listValidator.MatchString(uc.AllowedTeams)
	allowedNamespacesError := !listValidator.MatchString(uc.AllowedNamespaces)
	allowedClustersError := !listValidator.MatchString(uc.AllowedClusters)
	loglevelError := !loglevelValidator.MatchString(uc.LogLevel)

	if nameError || customVersionError || allowedTeamsError || allowedNamespacesError || allowedClustersError || loglevelError {
		unleashVersions, err := github.UnleashVersions()
		if err != nil {
			h.logger.WithError(err).Error("Error getting Unleash versions from Github")
			unleashVersions = []github.UnleashVersion{}
		}

		c.HTML(400, "unleash-form.html", gin.H{
			"title":                  title,
			"action":                 action,
			"name":                   name,
			"customVersion":          uc.CustomVersion,
			"unleashVersions":        unleashVersions,
			"customImageName":        unleash.UnleashCustomImageName,
			"allowedTeams":           uc.AllowedTeams,
			"allowedNamespaces":      uc.AllowedNamespaces,
			"allowedClusters":        uc.AllowedClusters,
			"logLevel":               uc.LogLevel,
			"nameError":              nameError,
			"customVersionError":     customVersionError,
			"allowedTeamsError":      allowedTeamsError,
			"allowedNamespacesError": allowedNamespacesError,
			"allowedClustersError":   allowedClustersError,
			"loglevelError":          loglevelError,
			"error":                  "Input validation failed, see errors in above fields",
		})
		return
	}

	if action == "edit" {
		err = h.unleashService.Update(ctx, uc)
	} else {
		err = h.unleashService.Create(ctx, uc)
	}

	if err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error persisting Unleash instance, check server logs")
		return
	}

	c.Redirect(302, "/unleash/"+name)
}

func (h *Handler) UnleashInstanceDelete(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)

	c.HTML(200, "unleash-delete.html", gin.H{
		"title": "Delete Unleash: " + instance.Name,
		"name":  instance.Name,
	})
}

func (h *Handler) UnleashInstanceDeletePost(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)

	ctx := c.Request.Context()
	name := regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(c.PostForm("name"), "")

	if name != instance.Name {
		c.HTML(400, "unleash-delete.html", gin.H{
			"title": "Delete Unleash: " + instance.Name,
			"name":  instance.Name,
			"error": "Instance name does not match",
		})
		return
	}

	if err := h.unleashService.Delete(ctx, instance.Name); err != nil {
		c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error deleting unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}
