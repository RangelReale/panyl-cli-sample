package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/RangelReale/ecapplog-go"
	panylcli "github.com/RangelReale/panyl-cli/v2"
	panylecapplog "github.com/RangelReale/panyl-ecapplog/v2"
	"github.com/RangelReale/panyl-plugins-ansi/v2/output"
	metadataPlugins "github.com/RangelReale/panyl-plugins/v2/metadata"
	"github.com/RangelReale/panyl-plugins/v2/parse"
	"github.com/RangelReale/panyl-plugins/v2/parseformat"
	"github.com/RangelReale/panyl-plugins/v2/postprocess"
	"github.com/RangelReale/panyl/v2"
	"github.com/RangelReale/panyl/v2/plugins/clean"
	"github.com/RangelReale/panyl/v2/plugins/consolidate"
	"github.com/RangelReale/panyl/v2/plugins/metadata"
	"github.com/RangelReale/panyl/v2/plugins/structure"
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
			flags.StringP("ecappaddress", "", "127.0.0.1:13991", "set ecapplog address")
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
				Name:    "redislog",
				Enabled: false,
			},
			{
				Name:    "nginxerrorlog",
				Enabled: false,
			},
			{
				Name:    "nginxjsonlog",
				Enabled: false,
			},
			{
				Name:    "elasticsearchjson",
				Enabled: false,
			},
		}),
		panylcli.WithProcessorProvider(func(ctx context.Context, preset string, pluginsEnabled []string,
			flags *pflag.FlagSet) (context.Context, *panyl.Processor, []panyl.JobOption, error) {
			parseflags := struct {
				Application  string `flag:"application"`
				StartLine    int    `flag:"start-line"`
				LineAmount   int    `flag:"line-amount"`
				Output       string `flag:"output"`
				ECAppName    string `flag:"ecappname"`
				ECAppAddress string `flag:"ecappaddress"`
				DebugParse   bool   `flag:"debug-parse"`
				DebugFormat  bool   `flag:"debug-format"`
			}{}

			err := panylcli.ParseFlags(flags, &parseflags)
			if err != nil {
				return ctx, nil, nil, err
			}

			jopt := []panyl.JobOption{
				panyl.WithLineLimit(parseflags.StartLine, parseflags.LineAmount),
			}

			ret := panyl.NewProcessor(
				panyl.WithOnJobFinished(panylcli.ExecProcessFinished),
			)
			if preset != "" {
				if preset == "default" {
					pluginsEnabled = append(pluginsEnabled, "json", "detectjson")
				} else if preset == "all" {
					pluginsEnabled = append(pluginsEnabled, "json", "dockercompose",
						"golog", "rubylog", "mongolog", "nginxjsonlog", "nginxerrorlog", "postgreslog", "redislog",
						"elasticsearchjson")
				} else {
					return ctx, nil, nil, fmt.Errorf("unknown preset '%s'", preset)
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
				case "detectjson":
					ret.RegisterPlugin(&postprocess.DetectJSON{})
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
				case "nginxjsonlog":
					ret.RegisterPlugin(&parse.NGINXJsonLog{})
				case "postgreslog":
					ret.RegisterPlugin(&parse.PostgresLog{})
				case "redislog":
					ret.RegisterPlugin(&parse.RedisLog{})
				case "elasticsearchjson":
					ret.RegisterPlugin(&parseformat.ElasticSearchJSON{})
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
				client = ecapplog.NewClient(ecapplog.WithAppName(ecname),
					ecapplog.WithAddress(parseflags.ECAppAddress),
					ecapplog.WithFlushOnClose(true))
				client.Open()

				jopt = append(jopt, panyl.WithIncludeSource(true))
				ctx = panylcli.SLogCLIToContext(ctx, slog.New(ecapplog.NewSLogHandler(client)))
			}

			if parseflags.DebugParse {
				switch parseflags.Output {
				case "console":
					ret.DebugLog = panyl.NewStdDebugLogOutput()
				case "ansi":
					ret.DebugLog = &output.AnsiLog{}
				case "ecapplog":
					ret.DebugLog = panylecapplog.NewLog(client,
						panylecapplog.WithSourceCategory("panyl-debug-parse"),
						panylecapplog.WithProcessCategory("panyl-debug-parse"))
				}
			}

			return ctx, ret, jopt, nil
		}),
		panylcli.WithResultProvider(func(ctx context.Context, flags *pflag.FlagSet) (panyl.Output, error) {
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

	exitCode, err := cmd.Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
	}
	os.Exit(exitCode)
}
