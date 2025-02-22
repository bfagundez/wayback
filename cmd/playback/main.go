// Copyright 2020 Wayback Archiver. All rights reserved.
// Use of this source code is governed by the GNU GPL v3
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"github.com/wabarc/playback"
	"github.com/wabarc/wayback"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "playback",
		Short:   "A toolkit to playback archived webpage from time capsules.",
		Example: `  playback https://example.com https://example.org`,
		Version: playback.Version,
		Run: func(cmd *cobra.Command, args []string) {
			handle(cmd, args)
		},
	}

	// nolint:errcheck
	rootCmd.Execute()
}

func handle(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		// nolint:errcheck
		cmd.Usage()
		os.Exit(0)
	}

	urls, err := unmarshalArgs(args)
	if err != nil {
		cmd.Println(err)
		os.Exit(1)
	}

	collects, err := wayback.Playback(context.TODO(), urls...)
	if err != nil {
		cmd.Println(err)
		os.Exit(1)
	}

	for _, collect := range collects {
		fmt.Printf("[%s]\n", collect.Arc)
		for orig, dest := range collect.Dst {
			fmt.Println(orig, "=>", dest)
		}
		fmt.Printf("\n")
	}
}

func unmarshalArgs(args []string) (urls []*url.URL, err error) {
	for _, s := range args {
		uri, er := url.Parse(s)
		if er != nil {
			err = fmt.Errorf("%w: unexpect url: %s", err, s)
			continue
		}
		urls = append(urls, uri)
	}
	return
}
