package server

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/server/routes"
)

func Run(config *config.Config) {
	router := gin.Default()
	router.Use(errorHandler)
	router.Static("/assets", "./assets")

	router.Use(func(c *gin.Context) {
		c.Set("config", config)
		c.Next()
	})

	router.HTMLRender = loadTemplates("./templates")
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "Frontend Plattform",
		})
	})

	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "OK")
	})

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
		}
	}

	fmt.Printf("Listening on %s", config.GetServerAddr())
	if err := router.Run(config.GetServerAddr()); err != nil {
		log.Fatal(err)
	}
}

func errorHandler(c *gin.Context) {
	c.Next()

	errorToPrint := c.Errors.ByType(gin.ErrorTypePublic).Last()
	if errorToPrint != nil {
		fmt.Println("here")
		c.HTML(500, "error.html", gin.H{
			"title": "Error",
			"error": errorToPrint.Meta,
		})
	}
}

func loadTemplates(templatesDir string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	layouts, err := filepath.Glob(templatesDir + "/layouts/*.html")
	if err != nil {
		panic(err.Error())
	}

	includes, err := filepath.Glob(templatesDir + "/includes/*.html")
	if err != nil {
		panic(err.Error())
	}

	// Generate our templates map from our layouts/ and includes/ directories
	for _, include := range includes {
		layoutCopy := make([]string, len(layouts))
		copy(layoutCopy, layouts)
		files := append(layoutCopy, include)
		r.AddFromFiles(filepath.Base(include), files...)
	}
	return r
}
