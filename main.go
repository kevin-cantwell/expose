package main

import (
	"github.com/alecthomas/kong"
	"github.com/kevin-cantwell/expose/cmd"
)

func main() {
	var cli cmd.CLI
	ctx := kong.Parse(&cli,
		kong.Name("expose"),
		kong.Description("Expose local ports via secure HTTPS tunnels on your own domain."),
		kong.UsageOnError(),
	)
	ctx.FatalIfErrorf(ctx.Run())
}
