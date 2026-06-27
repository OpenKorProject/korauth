package handler

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed docs/openapi.yaml
var openAPISpec string

func DocsHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
  <title>korauth API</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    html, body {
      height: 100%;
      margin: 0;
      padding: 0;
    }
  </style>
</head>
<body>
  <script id="api-reference" data-url="/openapi.yml"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@latest"></script>
</body>
</html>`
}

func Docs(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, DocsHTML())
}

func OpenAPISpec(c *gin.Context) {
	c.Header("Content-Type", "application/yaml; charset=utf-8")
	c.String(http.StatusOK, openAPISpec)
}
