package main

import (
	"embed"
	"html/template"

	"surat-tanah/internal/handler"
)

//go:embed templates/*.html
var templatesFS embed.FS

// parseTemplates memuat template cetak (html/template) dari embed.
func parseTemplates() *template.Template {
	return template.Must(
		template.New("").Funcs(handler.TemplateFuncs()).ParseFS(templatesFS, "templates/*.html"),
	)
}
