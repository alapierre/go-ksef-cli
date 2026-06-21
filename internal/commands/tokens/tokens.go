package tokens

type Cmd struct {
	Export ExportCmd `cmd:"" help:"Export stored KSeF tokens to encrypted backup"`
	Import ImportCmd `cmd:"" help:"Import stored KSeF tokens from encrypted backup"`
}
