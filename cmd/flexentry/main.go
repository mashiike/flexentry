package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fujiwara/logutils"
	"github.com/mashiike/flexentry"
)

var (
	Version string = "current"
)

func main() {
	showVersion := flag.Bool("v", false, "show version")
	showHelp := flag.Bool("h", false, "show help")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Usage of flexenty:")
		fmt.Fprintln(flag.CommandLine.Output(), "flexentry [opts] [commands...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	logLevel := "info"
	if l := os.Getenv("FLEXENTRY_LOG_LEVEL"); l != "" {
		logLevel = l
	}
	filter := &logutils.LevelFilter{
		Levels: []logutils.LogLevel{"debug", "info", "warn", "error"},
		ModifierFuncs: []logutils.ModifierFunc{
			logutils.Color(color.FgHiBlack),
			nil,
			logutils.Color(color.FgYellow),
			logutils.Color(color.FgRed, color.BgBlack),
		},
		MinLevel: logutils.LogLevel(strings.ToLower(logLevel)),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
	log.Println("[debug] flexentry version:", Version)
	if *showVersion {
		fmt.Println("flexentry:", Version)
		return
	}
	if *showHelp {
		flag.Usage()
		return
	}
	entrypoint := flexentry.Entrypoint{
		Executer: flexentry.NewSSMWrapExecuter(
			flexentry.NewShellExecuter(),
			time.Minute,
		),
	}
	if err := entrypoint.Run(context.Background(), flag.Args()...); err != nil {
		log.Fatalln("[error] ", err)
	}
}
