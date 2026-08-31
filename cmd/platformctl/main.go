package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/belimm01/platform-blueprint/internal/claim"
	"github.com/belimm01/platform-blueprint/internal/render"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		file := fs.String("file", "service.yaml", "path to a service claim")
		_ = fs.Parse(os.Args[2:])
		c, err := claim.Load(*file)
		if err == nil {
			err = c.Validate()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid service claim:", err)
			os.Exit(1)
		}
		fmt.Printf("valid service claim: %s/%s\n", c.Owner, c.Name)
	case "render":
		fs := flag.NewFlagSet("render", flag.ExitOnError)
		file := fs.String("file", "service.yaml", "path to a service claim")
		out := fs.String("out", "-", "output file, or - for stdout")
		_ = fs.Parse(os.Args[2:])
		c, err := claim.Load(*file)
		if err == nil {
			err = c.Validate()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid service claim:", err)
			os.Exit(1)
		}
		data, err := render.Manifests(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "render manifests:", err)
			os.Exit(1)
		}
		if *out == "-" {
			_, _ = os.Stdout.Write(data)
			return
		}
		if err := os.WriteFile(*out, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write manifests:", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: platformctl <validate|render> [-file service.yaml] [-out manifests.yaml]")
}
