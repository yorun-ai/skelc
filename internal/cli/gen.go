package cli

import (
	"context"
	"strings"

	ucli "github.com/urfave/cli/v3"
	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

const (
	commandGen         = "gen"
	commandGenGo       = "go"
	commandGenGoModule = "go-module"
	commandGenSkel     = "skel"
	commandGenTS       = "ts"

	flagGenPub            = "pub"
	flagGenSkelIn         = "skel-in"
	flagGenSkelOut        = "skel-out"
	flagGenGoOut          = "go-out"
	flagGenGoPubOut       = "go-pub-out"
	flagGenTSOut          = "ts-out"
	flagGenGoModulePrefix = "go-module-prefix"
	flagGenGoModule       = "go-module"
	flagGenGoPubModule    = "go-pub-module"
	flagGenGoImport       = "go-import"
	flagGenTSAsModule     = "ts-as-module"
	flagGenTSModuleScope  = "ts-module-scope"
	flagGenTSModule       = "ts-module"
	flagGenTSImport       = "ts-import"
	flagGenSkelImport     = "skel-import"
	flagGenGoVineVersion  = "go-vine-version"
	flagGenNoClean        = "no-clean"
)

func newGenCommand() *ucli.Command {
	return &ucli.Command{
		Name:               commandGen,
		Usage:              "generate code from skel definitions",
		HideHelpCommand:    true,
		CustomHelpTemplate: groupCommandHelpTemplate,
		Commands: []*ucli.Command{
			newGenSkelCommand(),
			newGenGoCommand(),
			newGenGoModuleCommand(),
			newGenTSCommand(),
		},
	}
}

func newGenGoCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandGenGo,
		Usage: "generate non-module Go code from skel definitions",
		Flags: newGenGoFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			input, option := parseGenGoCommand(cmd)
			option.CompilerVersion = compilerVersion()
			result, err := skelc.CompileGolang(input, option)
			if err != nil {
				panic(err)
			}
			printWarnings(cmd, result.Warnings)
			return nil
		},
	}
}

func newGenGoModuleCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandGenGoModule,
		Usage: "generate Go module code from skel definitions",
		Flags: newGenGoModuleFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			input, option := parseGenGoModuleCommand(cmd)
			option.CompilerVersion = compilerVersion()
			result, err := skelc.CompileGolang(input, option)
			if err != nil {
				panic(err)
			}
			printWarnings(cmd, result.Warnings)
			return nil
		},
	}
}

func newGenTSCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandGenTS,
		Usage: "generate TypeScript code from skel definitions",
		Flags: newGenTSFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			input, option := parseGenTSCommand(cmd)
			result, err := skelc.CompileTypeScript(input, option)
			if err != nil {
				panic(err)
			}
			printWarnings(cmd, result.Warnings)
			return nil
		},
	}
}

func newGenSkelCommand() *ucli.Command {
	return &ucli.Command{
		Name:  commandGenSkel,
		Usage: "generate pub skel definitions from skel definitions",
		Flags: newGenSkelFlags(),
		Action: func(_ context.Context, cmd *ucli.Command) error {
			input, option := parseGenSkelCommand(cmd)
			result, err := skelc.CompileSkeleton(input, option)
			if err != nil {
				panic(err)
			}
			printWarnings(cmd, result.Warnings)
			return nil
		},
	}
}

func compilerVersion() string {
	return mustDebugBuildInfo().Version
}

func newGenGoFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagGenSkelIn, Usage: "skeleton input file or directory"},
		&ucli.StringFlag{Name: flagGenGoOut, Usage: "Go output directory"},
		&ucli.StringFlag{Name: flagGenGoVineVersion, Usage: "Vine module version for generated Go code"},
		&ucli.BoolFlag{Name: flagGenNoClean, Usage: "do not clean existing output directory files"},
	}
}

func parseGenGoCommand(cmd *ucli.Command) (skelc.Input, skelc.GolangOption) {
	checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s %s", commandGen, commandGenGo)

	input := skelc.Input{
		SkelIn: cmd.String(flagGenSkelIn),
	}

	option := skelc.GolangOption{
		Out:         cmd.String(flagGenGoOut),
		VineVersion: strings.TrimSpace(cmd.String(flagGenGoVineVersion)),
		NoClean:     cmd.Bool(flagGenNoClean),
	}
	validateGoGenOption(input, option)
	return input, option
}

func newGenGoModuleFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.StringFlag{Name: flagGenSkelIn, Usage: "skeleton input file or directory"},
		&ucli.StringSliceFlag{Name: flagGenSkelImport, Usage: "skel import mapping in domain=path form"},
		&ucli.StringFlag{Name: flagGenGoOut, Usage: "Go output directory"},
		&ucli.StringFlag{Name: flagGenGoModule, Usage: "Go module path"},
		&ucli.StringFlag{Name: flagGenGoPubOut, Usage: "Go pub output directory"},
		&ucli.StringFlag{Name: flagGenGoPubModule, Usage: "Go pub module path"},
		&ucli.StringSliceFlag{Name: flagGenGoImport, Usage: "Go import mapping in domain=package form"},
		&ucli.StringFlag{Name: flagGenGoModulePrefix, Usage: "Go module path prefix"},
		&ucli.StringFlag{Name: flagGenGoVineVersion, Usage: "Vine module version for generated Go modules"},
		&ucli.BoolFlag{Name: flagGenNoClean, Usage: "do not clean existing output directory files"},
	}
}

func parseGenGoModuleCommand(cmd *ucli.Command) (skelc.Input, skelc.GolangOption) {
	checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s %s", commandGen, commandGenGoModule)

	skelImports := parseMappingFlags(cmd.StringSlice(flagGenSkelImport), flagGenSkelImport)
	goImports := parseMappingFlags(cmd.StringSlice(flagGenGoImport), flagGenGoImport)

	input := skelc.Input{
		SkelIn:      cmd.String(flagGenSkelIn),
		SkelImports: skelImports,
	}

	option := skelc.GolangOption{
		Out:          cmd.String(flagGenGoOut),
		AsModule:     true,
		PubOut:       cmd.String(flagGenGoPubOut),
		ModulePrefix: cmd.String(flagGenGoModulePrefix),
		Module:       cmd.String(flagGenGoModule),
		PubModule:    cmd.String(flagGenGoPubModule),
		Imports:      goImports,
		VineVersion:  strings.TrimSpace(cmd.String(flagGenGoVineVersion)),
		NoClean:      cmd.Bool(flagGenNoClean),
	}
	validateGoGenOption(input, option)
	return input, option
}

func newGenSkelFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.BoolFlag{Name: flagGenPub, Usage: "required, generate only pub skel definitions"},
		&ucli.StringFlag{Name: flagGenSkelIn, Usage: "skeleton input file or directory"},
		&ucli.StringFlag{Name: flagGenSkelOut, Usage: "skel output directory"},
		&ucli.BoolFlag{Name: flagGenNoClean, Usage: "do not clean existing output directory files"},
		&ucli.StringSliceFlag{Name: flagGenSkelImport, Usage: "skel import mapping in domain=path form"},
	}
}

func parseGenSkelCommand(cmd *ucli.Command) (skelc.Input, skelc.SkeletonOption) {
	checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s %s", commandGen, commandGenSkel)

	skelImports := parseMappingFlags(cmd.StringSlice(flagGenSkelImport), flagGenSkelImport)
	input := skelc.Input{
		SkelIn:      cmd.String(flagGenSkelIn),
		SkelImports: skelImports,
	}
	option := skelc.SkeletonOption{
		PubOnly: cmd.Bool(flagGenPub),
		Out:     cmd.String(flagGenSkelOut),
		NoClean: cmd.Bool(flagGenNoClean),
	}
	validateSkelGenOption(input, option)
	return input, option
}

func newGenTSFlags() []ucli.Flag {
	return []ucli.Flag{
		&ucli.BoolFlag{Name: flagGenPub, Usage: "generate only pub TypeScript code"},
		&ucli.StringFlag{Name: flagGenSkelIn, Usage: "skeleton input file or directory"},
		&ucli.StringFlag{Name: flagGenTSOut, Usage: "TypeScript output directory"},
		&ucli.BoolFlag{Name: flagGenNoClean, Usage: "do not clean existing output directory files"},
		&ucli.StringSliceFlag{Name: flagGenSkelImport, Usage: "skel import mapping in domain=path form"},
		&ucli.BoolFlag{Name: flagGenTSAsModule, Usage: "generate TypeScript module package metadata"},
		&ucli.StringFlag{Name: flagGenTSModuleScope, Usage: "TypeScript module package scope"},
		&ucli.StringFlag{Name: flagGenTSModule, Usage: "TypeScript module package name"},
		&ucli.StringSliceFlag{Name: flagGenTSImport, Usage: "TypeScript import mapping in domain=package form"},
	}
}

func parseGenTSCommand(cmd *ucli.Command) (skelc.Input, skelc.TypeScriptOption) {
	checkutil.Check(cmd.Args().Len() == 0, "unexpected args for %s %s", commandGen, commandGenTS)

	skelImports := parseMappingFlags(cmd.StringSlice(flagGenSkelImport), flagGenSkelImport)
	tsImports := parseMappingFlags(cmd.StringSlice(flagGenTSImport), flagGenTSImport)
	input := skelc.Input{
		SkelIn:      cmd.String(flagGenSkelIn),
		SkelImports: skelImports,
	}
	option := skelc.TypeScriptOption{
		PubOnly:     cmd.Bool(flagGenPub),
		Out:         cmd.String(flagGenTSOut),
		AsModule:    cmd.Bool(flagGenTSAsModule),
		ModuleScope: cmd.String(flagGenTSModuleScope),
		Module:      cmd.String(flagGenTSModule),
		Imports:     tsImports,
		NoClean:     cmd.Bool(flagGenNoClean),
	}
	validateTypeScriptGenOption(input, option)
	return input, option
}
