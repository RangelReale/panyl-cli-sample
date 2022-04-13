package main

import (
	"fmt"
	"github.com/RangelReale/panyl-plugins/postprocess"
	"os"

	"github.com/RangelReale/ecapplog-go"
	"github.com/RangelReale/panyl"
	panylcli "github.com/RangelReale/panyl-cli"
	panylecapplog "github.com/RangelReale/panyl-ecapplog"
	"github.com/RangelReale/panyl-plugins-ansi/output"
	metadataPlugins "github.com/RangelReale/panyl-plugins/metadata"
	"github.com/RangelReale/panyl-plugins/parse"
	"github.com/RangelReale/panyl/plugins/clean"
	"github.com/RangelReale/panyl/plugins/consolidate"
	"github.com/RangelReale/panyl/plugins/metadata"
	"github.com/RangelReale/panyl/plugins/structure"
	"github.com/spf13/pflag"
)

func main() {
	var client *ecapplog.Client

	cmd := panylcli.New(
		panylcli.WithDeclareGlobalFlags(func(flags *pflag.FlagSet) {
			flags.StringP("application", "a", "", "set application name")
			flags.IntP("start-line", "s", 0, "start line (0 = first line, 1 = second line)")
			flags.IntP("line-amount", "m", 0, "amount of lines to process (0 = all)")
			flags.StringP("output", "o", "console", "output (console, ansi, ecapplog)")
			flags.String("ecappname", "", "set ecapplog app name (default = application flag)")
			flags.Bool("debug-parse", false, "debug parsing")
			flags.Bool("debug-format", false, "debug format")
		}),
		panylcli.WithPluginOptions([]panylcli.PluginOption{
			{
				Name:          "ansiescape",
				Enabled:       true,
				Preset:        true,
				PresetEnabled: true,
			},
			{
				Name:    "json",
				Enabled: true,
			},
			{
				Name:    "consolidate-lines",
				Enabled: false,
			},
			{
				Name:    "dockercompose",
				Enabled: false,
			},
			{
				Name:    "golog",
				Enabled: false,
			},
			{
				Name:    "rubylog",
				Enabled: false,
			},
			{
				Name:    "mongolog",
				Enabled: false,
			},
			{
				Name:    "postgreslog",
				Enabled: false,
			},
			{
				Name:    "nginxerrorlog",
				Enabled: false,
			},
		}),
		panylcli.WithProcessorProvider(func(preset string, pluginsEnabled []string, flags *pflag.FlagSet) (*panyl.Processor, error) {
			parseflags := struct {
				Application string `flag:"application"`
				StartLine   int    `flag:"start-line"`
				LineAmount  int    `flag:"line-amount"`
				Output      string `flag:"output"`
				ECAppName   string `flag:"ecappname"`
				DebugParse  bool   `flag:"debug-parse"`
				DebugFormat bool   `flag:"debug-format"`
			}{}

			err := panylcli.ParseFlags(flags, &parseflags)
			if err != nil {
				return nil, err
			}

			ret := panyl.NewProcessor(panyl.WithLineLimit(parseflags.StartLine, parseflags.LineAmount))
			if preset != "" {
				if preset == "default" {
					pluginsEnabled = append(pluginsEnabled, "json")
				} else if preset == "all" {
					pluginsEnabled = append(pluginsEnabled, "json", "dockercompose",
						"golog", "rubylog", "mongolog", "nginxerrorlog", "postgreslog")
				} else {
					return nil, fmt.Errorf("unknown preset '%s'", preset)
				}
			}

			if parseflags.Application != "" {
				ret.RegisterPlugin(&metadata.ForceApplication{Application: parseflags.Application})
			}

			if parseflags.DebugFormat {
				pluginsEnabled = append(pluginsEnabled, "debugformat")
			}

			for _, plugin := range panylcli.PluginsEnabledUnique(pluginsEnabled) {
				switch plugin {
				case "ansiescape":
					ret.RegisterPlugin(&clean.AnsiEscape{})
				case "json":
					ret.RegisterPlugin(&structure.JSON{})
				case "consolidate-lines":
					ret.RegisterPlugin(&consolidate.JoinAllLines{})
				case "dockercompose":
					ret.RegisterPlugin(&metadataPlugins.DockerCompose{})
				case "golog":
					ret.RegisterPlugin(&parse.GoLog{})
				case "rubylog":
					ret.RegisterPlugin(&parse.RubyLog{})
				case "mongolog":
					ret.RegisterPlugin(&parse.MongoLog{})
				case "nginxerrorlog":
					ret.RegisterPlugin(&parse.NGINXErrorLog{})
				case "postgreslog":
					ret.RegisterPlugin(&parse.PostgresLog{})
				case "debugformat":
					ret.RegisterPlugin(&postprocess.DebugFormat{})
				}
			}

			if parseflags.Output == "ecapplog" {
				ecname := parseflags.ECAppName
				if ecname == "" {
					ecname = parseflags.Application
				}
				if ecname == "" {
					ecname = "panyl-cli-sample"
				}
				client = ecapplog.NewClient(ecapplog.WithAppName(parseflags.ECAppName))
				client.Open()

				ret.IncludeSource = true
			}

			if parseflags.DebugParse {
				switch parseflags.Output {
				case "console":
					ret.Logger = panyl.NewStdLogOutput()
				case "ansi":
					ret.Logger = &output.AnsiLog{}
				case "ecapplog":
					ret.Logger = panylecapplog.NewLog(client,
						panylecapplog.WithSourceCategory("panyl-debug-parse"),
						panylecapplog.WithProcessCategory("panyl-debug-parse"))
				}
			}

			return ret, nil
		}),
		panylcli.WithResultProvider(func(flags *pflag.FlagSet) (panyl.ProcessResult, error) {
			parseflags := struct {
				Output string `flag:"output"`
			}{}

			err := panylcli.ParseFlags(flags, &parseflags)
			if err != nil {
				return nil, err
			}

			switch parseflags.Output {
			case "console":
				return panylcli.NewOutput(), nil
			case "ansi":
				return output.NewAnsiOutput(true), nil
			case "ecapplog":
				return panylecapplog.NewOutput(client,
					panylecapplog.WithApplicationAsCategory(true),
					panylecapplog.WithAppendCategoryToApplication(false),
				), nil
			}
			return nil, fmt.Errorf("unknown output '%s'", parseflags.Output)
		}),
	)

	err := cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
