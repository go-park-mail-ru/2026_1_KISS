package docs

import "embed"

//go:embed swagger.json swagger-ui/*
var Assets embed.FS
