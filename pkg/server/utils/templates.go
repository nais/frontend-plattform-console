package utils

import (
	"path/filepath"
	"text/template"

	"github.com/gin-contrib/multitemplate"
	"github.com/nais/bifrost/pkg/config"
)

func LoadFuncMap(c *config.Config) template.FuncMap {
	return template.FuncMap{
		"version": func() string {
			return c.Meta.Version
		},
		"versionUrl": func() string {
			return c.Meta.VersionUrl()
		},
		"buildDate": func() string {
			return c.Meta.BuildDate()
		},
		"commit": func() string {
			return c.Meta.Commit()
		},
		"commitUrl": func() string {
			return c.Meta.CommitUrl()
		},
		"repo": func() string {
			return c.Meta.Repo
		},
		"repoUrl": func() string {
			return c.Meta.RepoUrl()
		},
	}
}

func LoadTemplates(c *config.Config) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	templatesDir := c.Server.TemplatesDir

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
		r.AddFromFilesFuncs(filepath.Base(include), LoadFuncMap(c), files...)
	}
	return r
}
