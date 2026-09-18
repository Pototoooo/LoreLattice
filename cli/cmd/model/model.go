// Package modelcmd holds the `lorelattice model` command tree: list / view.
//
// These are read-only discovery commands. The model id is a required input to
// `lorelattice agent create --model <id>` and to a knowledge base's embedding /
// summary model settings, so the CLI must offer a first-class way to find it
// rather than forcing callers down to the raw `lorelattice api` escape hatch.
//
// The directory is named `model/` to match the cobra subcommand; the Go
// package is `modelcmd` to avoid colliding with the SDK's Model type.
package modelcmd

import (
	"github.com/spf13/cobra"

	"github.com/Pototoooo/lorelattice/cli/internal/cmdutil"
	sdk "github.com/Pototoooo/lorelattice/client"
)

// NewCmd builds the `lorelattice model` parent and registers leaves. Called from
// cli/cmd/root.go.
func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage models (list / view / create / update / delete)",
		Long: `List, inspect, register, and delete the models configured on the server. Use
the model id to back a knowledge base's embedding / summary config ('lorelattice kb
init') or an agent ('lorelattice agent create --model <id>').`,
	}
	cmd.AddCommand(NewCmdList(f))
	cmd.AddCommand(NewCmdView(f))
	cmd.AddCommand(NewCmdCreate(f))
	cmd.AddCommand(NewCmdUpdate(f))
	cmd.AddCommand(NewCmdDelete(f))
	return cmd
}

// modelLabel prefers display_name, falling back to the raw name.
func modelLabel(m sdk.Model) string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.Name
}
