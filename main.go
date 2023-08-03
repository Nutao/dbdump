package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Nutao/dbdump/formatter"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func init() {
	// Disable `h` alias for `help` flag
	cli.HelpFlag = &cli.BoolFlag{
		Name:  "help",
		Usage: "show help",
	}
}

func main() {
	tables := cli.StringSlice{}

	app := cli.NewApp()
	app.Usage = "MySQL Data Define Tool"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "dbType",
			Aliases:     []string{"DB"},
			Usage:       "db类型, 支持mysql，pgsql",
			Required:    true,
			Value:       "mysql",
			Destination: &gConfig.DBType,
			DefaultText: "mysql",
		},
		&cli.StringFlag{
			Name:        "host",
			Aliases:     []string{"h"},
			Usage:       "Connect to host.",
			Required:    false,
			Value:       "127.0.0.1",
			Destination: &gConfig.Host,
		},
		&cli.UintFlag{
			Name:        "port",
			Aliases:     []string{"P"},
			Usage:       "Port number to use for connection.",
			Required:    false,
			Value:       3306,
			Destination: &gConfig.Port,
		},
		&cli.StringFlag{
			Name:        "user",
			Aliases:     []string{"u"},
			Usage:       "User for login if not current user.",
			Required:    false,
			Value:       "root",
			Destination: &gConfig.User,
		},
		&cli.StringFlag{
			Name:        "password",
			Aliases:     []string{"p"},
			Usage:       "Password to use when connecting to server.",
			Required:    true,
			Value:       "",
			Destination: &gConfig.Password,
		},
		&cli.StringFlag{
			Name:        "database",
			Aliases:     []string{"D"},
			Usage:       "Database to use.",
			Required:    true,
			Value:       "",
			Destination: &gConfig.Database,
		},
		&cli.StringSliceFlag{
			Name:        "tables",
			Aliases:     []string{"t"},
			Usage:       "Tables to get.",
			Required:    false,
			Value:       nil,
			Destination: &tables,
		},
		&cli.StringFlag{
			Name:        "output",
			Aliases:     []string{"o"},
			Usage:       "Write to file instead of stdout.",
			Required:    false,
			Value:       "",
			Destination: &gConfig.Output,
		},
		&cli.StringFlag{
			Name: "format_type",
			Usage: fmt.Sprintf("Format type of the output(%v).",
				strings.Join(formatter.AllFormatter(), "|")),
			Required:    false,
			Value:       "json",
			Destination: &gConfig.FormatType,
		},
		&cli.StringFlag{
			Name:        "format_config",
			Usage:       "Format config of the output. Filename prepend with @",
			Required:    false,
			Value:       "",
			Destination: &gConfig.FormatConfig,
		},
	}

	// 读取输出模板文件
	app.Before = func(context *cli.Context) error {
		var err error

		gConfig.Tables = tables.Value()
		gConfig.Formatter, err = formatter.NewFormatter(gConfig.FormatType)
		if err != nil {
			return fmt.Errorf("create formatter failed, %w", err)
		}

		formatConfig := []byte(gConfig.FormatConfig)
		if strings.HasPrefix(gConfig.FormatConfig, "@") {
			formatConfig, err = ioutil.ReadFile(gConfig.FormatConfig[1:])
			if err != nil {
				return fmt.Errorf("load format config failed, %w", err)
			}
		}
		if err = gConfig.Formatter.Initialize(formatConfig); err != nil {
			return fmt.Errorf("initialize formatter failed, %w", err)
		}

		if gConfig.DBType == "mysql" {
			app.Action = dump
		} else if gConfig.DBType == "pgsql" {
			app.Action = dumpPGSQL
		} else {
			fmt.Println("输入了不支持的DBType")
			os.Exit(-1)
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
