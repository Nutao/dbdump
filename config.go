package main

import "github.com/Nutao/dbdump/formatter"

var (
	gConfig = &config{}
)

//
type config struct {
	DBType   string // db类型
	Host     string
	Port     uint
	User     string
	Password string
	Database string
	Tables   []string

	Output       string
	Formatter    formatter.Formatter
	FormatType   string
	FormatConfig string
}
