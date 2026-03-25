package cmd

// CLI is the root Kong command tree.
type CLI struct {
	Serve  ServeCmd  `cmd:"serve" help:"Run the expose tunnel server"`
	Login  LoginCmd  `cmd:"login" help:"Authenticate with GitHub"`
	Ls     LsCmd     `cmd:"ls" help:"List active tunnels on this machine"`
	Tunnel TunnelCmd `cmd:"" default:"withargs" help:"Tunnel a local port (default command)"`
}
