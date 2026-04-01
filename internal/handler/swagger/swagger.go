package swagger

import (
	"embed"
	"net/http"

	"github.com/NaphonJangjit/HeartFolio/internal/webhttp"
)

//go:embed openapi.yaml
var specFS embed.FS

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>HeartFolio API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "./openapi.yaml",
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset
      ],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`

func Handler() *webhttp.Router {
	r := webhttp.New()

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerUIHTML))
	})

	r.Get("/openapi.yaml", func(w http.ResponseWriter, req *http.Request) {
		data, err := specFS.ReadFile("openapi.yaml")
		if err != nil {
			http.Error(w, "spec not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(data)
	})

	return r
}
