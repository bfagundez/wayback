// Copyright 2022 Wayback Archiver. All rights reserved.
// Use of this source code is governed by the GNU GPL v3
// license that can be found in the LICENSE file.

package service // import "github.com/wabarc/wayback/service"

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/wabarc/helper"
	"github.com/wabarc/logger"
	"github.com/wabarc/wayback"
	"github.com/wabarc/wayback/config"
	"github.com/wabarc/wayback/reduxer"
)

func TestWayback(t *testing.T) {
	defer helper.CheckTest(t)

	// Don't wayback to any slot to speed up testing.
	os.Clearenv()
	os.Setenv("WAYBACK_ENABLE_IA", "false")
	os.Setenv("WAYBACK_ENABLE_IS", "false")
	os.Setenv("WAYBACK_ENABLE_IP", "false")
	os.Setenv("WAYBACK_ENABLE_PH", "false")

	parser := config.NewParser()
	var err error
	if config.Opts, err = parser.ParseEnvironmentVariables(); err != nil {
		t.Fatalf("Parse environment variables or flags failed, error: %v", err)
	}
	logger.SetLogLevel(logger.LevelFatal)

	u, _ := url.Parse("https://example.com/")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer helper.CheckContext(ctx, t)

	urls := []*url.URL{u}
	do := func(cols []wayback.Collect, rdx reduxer.Reduxer) error {
		time.Sleep(3 * time.Second)
		return nil
	}
	w := Wayback(ctx, urls, do)

	if w == nil {
		t.Fatal("Unexpected wayback exceeded")
	}
}

func TestWaybackWithoutReduxer(t *testing.T) {
	defer helper.CheckTest(t)

	// Don't wayback to any slot to speed up testing.
	os.Clearenv()
	os.Setenv("WAYBACK_ENABLE_IA", "false")
	os.Setenv("WAYBACK_ENABLE_IS", "false")
	os.Setenv("WAYBACK_ENABLE_IP", "false")
	os.Setenv("WAYBACK_ENABLE_PH", "false")
	os.Setenv("WAYBACK_STORAGE_DIR", "")

	parser := config.NewParser()
	var err error
	if config.Opts, err = parser.ParseEnvironmentVariables(); err != nil {
		t.Fatalf("Parse environment variables or flags failed, error: %v", err)
	}
	logger.SetLogLevel(logger.LevelFatal)

	u, _ := url.Parse("https://example.com/")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	defer helper.CheckContext(ctx, t)

	urls := []*url.URL{u}
	do := func(cols []wayback.Collect, rdx reduxer.Reduxer) error {
		time.Sleep(3 * time.Second)
		return nil
	}
	w := Wayback(ctx, urls, do)

	if w == nil {
		t.Fatal("Unexpected wayback exceeded")
	}
}
