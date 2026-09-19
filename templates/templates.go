package templates

import "embed"

// EmailFS embeds all HTML email templates for deployment
//
//go:embed emails/*.html
var EmailFS embed.FS
