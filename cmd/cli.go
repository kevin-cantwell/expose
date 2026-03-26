package cmd

// CLI is the root Kong command tree.
type CLI struct {
	Serve   ServeCmd   `cmd:"serve"   help:"Run the expose tunnel server"`
	Login   LoginCmd   `cmd:"login"   help:"Authenticate with GitHub"`
	Ls      LsCmd      `cmd:"ls"      help:"List active tunnels on this machine"`
	Start   StartCmd   `cmd:"start"   help:"Start a tunnel as a background process"`
	Stop    StopCmd    `cmd:"stop"    help:"Stop a background tunnel"`
	Restart RestartCmd `cmd:"restart" help:"Restart a background tunnel"`
	Logs    LogsCmd    `cmd:"logs"    help:"View or follow logs for a tunnel"`
	Tunnel  TunnelCmd  `cmd:"" default:"withargs" help:"Tunnel a local port (default command)"`
}
