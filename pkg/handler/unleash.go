package handler

import (
	"errors"
	"fmt"
	"html/template"
	"regexp"
	"time"

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
		_ = c.Error(err).
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

	uc := unleash.UnleashConfig{
		Name:                      "",
		CustomVersion:             unleashVersions[0].GitTag,
		EnableFederation:          true,
		FederationNonce:           "",
		AllowedTeams:              "",
		AllowedNamespaces:         "",
		AllowedClusters:           "dev-gcp,prod-gcp",
		LogLevel:                  "warn",
		DatabasePoolMax:           0,
		DatabasePoolIdleTimeoutMs: 0,
	}

	c.HTML(200, "unleash-form.html", gin.H{
		"title":           "New Unleash Instance",
		"action":          "create",
		"unleash":         uc,
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

func (h *Handler) UnleashInstanceShow(c *gin.Context) {
	instance := c.MustGet("unleashInstance").(*unleash.UnleashInstance)
	instanceYaml, err := utils.StructToYaml(instance.ServerInstance)
	if err != nil {
		h.logger.WithError(err).Error("Error converting Unleash struct to yaml")
		instanceYaml = "Parse error - see logs"
	}

	uc := unleash.UnleashVariables(instance.ServerInstance, false)

	c.HTML(200, "unleash-show.html", gin.H{
		"title":              "Unleash: " + instance.Name,
		"instance":           instance,
		"unleash":            uc,
		"googleProjectID":    h.config.Google.ProjectID,
		"googleProjectURL":   h.config.GoogleProjectURL(""),
		"sqlInstanceID":      h.config.Unleash.SQLInstanceID,
		"sqlInstanceURL":     h.config.GoogleProjectURL(fmt.Sprintf("sql/instances/%s/overview", h.config.Unleash.SQLInstanceID)),
		"sqlInstanceAddress": h.config.Unleash.SQLInstanceAddress,
		"sqlInstanceRegion":  h.config.Unleash.SQLInstanceRegion,
		"sqlDatabaseName":    instance.Name,
		"sqlDatabaseUser":    instance.Name,
		"sqlDatabaseSecret":  instance.Name,

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
		"title":           "Edit Unleash: " + instance.Name,
		"action":          "edit",
		"unleash":         uc,
		"unleashVersions": unleashVersions,
	})
}

func (h *Handler) UnleashInstancePost(c *gin.Context) {
	var (
		title, action string
		err           error
	)

	ctx := c.Request.Context()
	log := h.logger.WithContext(ctx)
	uc := &unleash.UnleashConfig{}

	instance, exists := c.Get("unleashInstance")
	if exists {
		instance, ok := instance.(*unleash.UnleashInstance)
		if !ok {
			_ = c.Error(fmt.Errorf("could not convert instance to UnleashInstance")).
				SetType(gin.ErrorTypePublic).
				SetMeta("Error parsing existing Unleash instance")
			return
		}
		uc = unleash.UnleashVariables(instance.ServerInstance, true)
	}

	unleashVersions, err := github.UnleashVersions()
	if err != nil {
		log.WithError(err).Error("Error getting Unleash versions from Github")
		unleashVersions = []github.UnleashVersion{
			{
				GitTag:        "v5.10.2-20240329-070801-0180a96",
				ReleaseTime:   time.Date(2024, 3, 29, 7, 8, 1, 0, time.UTC),
				CommitHash:    "0180a96",
				VersionNumber: "5.10.2",
			},
		}
	}

	if err = c.ShouldBind(uc); err != nil {
		log.WithError(err).Error("Error binding post data to Unleash config")

		_ = c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error binding post data to Unleash config")
		return
	}

	if exists {
		uc.Name = instance.(*unleash.UnleashInstance).ServerInstance.GetName()
	} else {
		uc.FederationNonce = utils.RandomString(8)
		uc.SetDefaultValues(unleashVersions)
	}

	//  We are removing the differentiating between teams and namespaces, and merging them into one field
	uc.MergeTeamsAndNamespaces()

	if validationErr := uc.Validate(); validationErr != nil {
		log.WithError(validationErr).Error("Error validating Unleash config")

		if exists {
			title = "Edit Unleash: " + uc.Name
			action = "edit"
		} else {
			title = "New Unleash Instance"
			action = "create"
		}

		if c.ContentType() == "application/json" {
			c.JSON(400, gin.H{
				"error":           "Input validation failed, see errors in details",
				"validationError": validationErr.Error(),
			})
		} else {
			c.HTML(400, "unleash-form.html", gin.H{
				"title":           title,
				"action":          action,
				"unleash":         uc,
				"unleashVersions": unleashVersions,
				"validationError": validationErr,
				"error":           "Input validation failed, see errors in details",
			})
		}
		return
	}

	if exists {
		err = h.unleashService.Update(ctx, uc)
	} else {
		err = h.unleashService.Create(ctx, uc)
	}

	if err != nil {
		var unleashErr *unleash.UnleashError

		reason := "Error persisting Unleash instance, check server logs"
		if errors.As(err, &unleashErr) {
			err = unleashErr.Err
			reason = fmt.Sprintf("Error persisting Unleash instance, %s", unleashErr.Reason)
		}

		_ = c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta(reason)
		return
	}

	if c.ContentType() == "application/json" {
		c.JSON(200, gin.H{
			"message":  "Unleash instance persisted",
			"instance": uc.Name,
		})
		return
	}

	c.Redirect(302, "/unleash/"+uc.Name)
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
		_ = c.Error(err).
			SetType(gin.ErrorTypePublic).
			SetMeta("Error deleting unleash instance")
		return
	}

	c.Redirect(302, "/unleash")
}
