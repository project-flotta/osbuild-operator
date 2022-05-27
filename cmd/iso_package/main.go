package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2" // imports as package "cli"
)

const (
	armArchValue = "aarch"
	x86ArchValue = "x86"
)

var (
	app *cli.App

	InfoLogger    = log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime)
	WarningLogger = log.New(os.Stderr, "WARNING: ", log.Ldate|log.Ltime)
	ErrorLogger   = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime)

	allowedArchs = []string{x86ArchValue, armArchValue}
)

func init() {
	app = cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "source",
			Aliases:  []string{"s"},
			Usage:    "ISO source url to use as based image",
			Required: true,
		},

		&cli.StringFlag{
			Name:     "kickstart",
			Aliases:  []string{"ks"},
			Usage:    "Kickstart file to append to the iso",
			Required: true,
		},

		&cli.StringFlag{
			Name:  "arch",
			Usage: "Arch to compile to, possible values: x86, aarch. Default x86",
			Value: "x86",
		},
	}
}

func main() {
	app.Action = func(c *cli.Context) error {
		arch := c.String("arch")
		archMatched := false
		for _, possibleArch := range allowedArchs {
			if arch == possibleArch {
				archMatched = true
				break
			}
		}

		if !archMatched {
			log.Fatalf("Arch value '%s' is not allowed. Allowed arches are: %v", arch, allowedArchs)
		}

		cfg := Config{
			Source:    c.String("source"),
			Kickstart: c.String("kickstart"),
			Arch:      arch,
		}
		_, err := cfg.Run()
		return err
	}

	err := app.Run(os.Args)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

}
