package main

import "embed"

//go:embed *.html *.js *.css
var FrontendFS embed.FS
