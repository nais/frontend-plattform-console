package handler

import (
	"fmt"
	"html/template"
	"regexp"

	"github.com/gin-gonic/gin"
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
	obj := unleash.UnleashSpec(h.config, "my-unleash", "", "", "", "")
	yamlString, err := utils.StructToYaml(obj)
	if err != nil {
		h.logger.WithError(err).Error("Error converting Unleash struct to yaml")
		yamlString = "Parse error - see logs"
	}

	c.HTML(200, "unleash-form.html", gin.H{
		"title":  "New Unleash Instance",
		"action": "create",
		"yaml":   yamlString,
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

	c.Set("unleashInstance", instance)
	c.Next()
}

func (h *Handler) UnleashInstanceShow(c *gin.Context) {
	// ctx := c.Request.Context()

	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)
	instanceYaml, err := utils.StructToYaml(instance.ServerInstance)
	if err != nil {
		h.logger.WithError(err).Error("Error converting Unleash struct to yaml")
		instanceYaml = "Parse error - see logs"
	}

	// @TODO get more info about the instance
	// h.unleashService.
	// instance.GetDatabase()
	// instance.GetDatabaseUser()

	c.HTML(200, "unleash-show.html", gin.H{
		"title":        "Unleash: " + instance.TeamName,
		"instance":     instance,
		"instanceYaml": template.HTML(instanceYaml),
	})
}

func (h *Handler) UnleashInstanceEdit(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)

	name, customVersion, allowedTeams, allowedNamespaces, allowedClusters := unleash.UnleashVariables(instance.ServerInstance)

	c.HTML(200, "unleash-form.html", gin.H{
		"title":              "Edit Unleash: " + instance.TeamName,
		"action":             "edit",
		"name":               name,
		"customImageName":    unleash.UnleashCustomImageName,
		"customImageVersion": customVersion,
		"allowedTeams":       allowedTeams,
		"allowedNamespaces":  allowedNamespaces,
		"allowedClusters":    allowedClusters,
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

	instance, exists := c.Get("unleashInstance")
	if exists {
		instance, ok := instance.(*unleash.UnleashInstance)
		if !ok {
			c.Error(fmt.Errorf("could not convert instance to UnleashInstance"))
			return
		}

		name = instance.TeamName
		title = "Edit Unleash: " + name
		action = "edit"
	} else {
		name = c.PostForm("name")
		title = "New Unleash Instance"
		action = "create"
	}

	customImageVersion := c.PostForm("custom-image-version")
	allowedTeams := c.PostForm("allowed-teams")
	allowedNamespaces := c.PostForm("allowed-namespaces")
	allowedClusters := c.PostForm("allowed-clusters")

	nameError := !nameValidator.MatchString(name)
	customImageVersionError := !versionValidator.MatchString(customImageVersion)
	allowedTeamsError := !listValidator.MatchString(allowedTeams)
	allowedNamespacesError := !listValidator.MatchString(allowedNamespaces)
	allowedClustersError := !listValidator.MatchString(allowedClusters)

	if nameError || customImageVersionError || allowedTeamsError || allowedNamespacesError || allowedClustersError {
		c.HTML(400, "unleash-form.html", gin.H{
			"title":                   title,
			"action":                  action,
			"name":                    name,
			"customImageVersion":      customImageVersion,
			"customImageName":         unleash.UnleashCustomImageName,
			"allowedTeams":            allowedTeams,
			"allowedNamespaces":       allowedNamespaces,
			"allowedClusters":         allowedClusters,
			"nameError":               nameError,
			"customImageVersionError": customImageVersionError,
			"allowedTeamsError":       allowedTeamsError,
			"allowedNamespacesError":  allowedNamespacesError,
			"allowedClustersError":    allowedClustersError,
			"error":                   "Input validation failed, see errors in above fields",
		})
		return
	}

	if action == "edit" {
		err = h.unleashService.Update(ctx, name, customImageVersion, allowedTeams, allowedNamespaces, allowedClusters)
	} else {
		err = h.unleashService.Create(ctx, name, customImageVersion, allowedTeams, allowedNamespaces, allowedClusters)
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
		"title": "Delete Unleash: " + instance.TeamName,
		"name":  instance.TeamName,
	})
}

func (h *Handler) UnleashInstanceDeletePost(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)

	ctx := c.Request.Context()
	name := regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(c.PostForm("name"), "")

	if name != instance.TeamName {
		c.HTML(400, "unleash-delete.html", gin.H{
			"title": "Delete Unleash: " + instance.TeamName,
			"name":  instance.TeamName,
			"error": "Instance name does not match",
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
