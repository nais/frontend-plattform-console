package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func HealthHandler(c *gin.Context) {
	c.String(200, "OK")
}

func ErrorHandler(c *gin.Context) {
	c.Next()

	log := c.MustGet("log").(*logrus.Logger)

	errorToPrint := c.Errors.ByType(gin.ErrorTypePublic).Last()
	if errorToPrint != nil {
		log.WithError(errorToPrint.Err).Error(errorToPrint.Meta)
		c.HTML(500, "error.html", gin.H{
			"title": "Error",
			"error": errorToPrint.Meta,
		})
	}
}
