// Copyright 2021 Wayback Archiver. All rights reserved.
// Use of this source code is governed by the GNU GPL v3
// license that can be found in the LICENSE file.

package publish // import "github.com/wabarc/wayback/publish"

import (
	"context"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/wabarc/logger"
	"github.com/wabarc/wayback"
	"github.com/wabarc/wayback/config"
	"github.com/wabarc/wayback/errors"
	"github.com/wabarc/wayback/reduxer"
	"golang.org/x/sync/errgroup"

	discord "github.com/bwmarrin/discordgo"
	mstdn "github.com/mattn/go-mastodon"
	nostr "github.com/nbd-wtf/go-nostr"
	slack "github.com/slack-go/slack"
	irc "github.com/thoj/go-ircevent"
	telegram "gopkg.in/telebot.v3"
	matrix "maunium.net/go/mautrix"
)

// Flag represents a type of uint8
type Flag uint8

const (
	FlagWeb      Flag = iota // FlagWeb publish from httpd service
	FlagTelegram             // FlagTelegram publish from telegram service
	FlagTwitter              // FlagTwitter publish from twitter srvice
	FlagMastodon             // FlagMastodon publish from mastodon service
	FlagDiscord              // FlagDiscord publish from discord service
	FlagMatrix               // FlagMatrix publish from matrix service
	FlagSlack                // FlagSlack publish from slack service
	FlagNostr                // FlagSlack publish from nostr
	FlagIRC                  // FlagIRC publish from relaychat service
)

var maxDelayTime = 10

// PubBundle represents a context key with value of `reduxer.Reduxer`.
type PubBundle struct{}

// Publisher is the interface that wraps the basic Publish method.
//
// Publish publish message to serveral media platforms, e.g. Telegram channel, GitHub Issues, etc.
// The cols must either be a []wayback.Collect, args use for specific service.
type Publisher interface {
	Publish(ctx context.Context, cols []wayback.Collect, args ...string) error
}

// String returns the flag as a string.
func (f Flag) String() string {
	switch f {
	case FlagWeb:
		return "httpd"
	case FlagTelegram:
		return "telegram"
	case FlagTwitter:
		return "twiter"
	case FlagMastodon:
		return "mastodon"
	case FlagDiscord:
		return "discord"
	case FlagMatrix:
		return "matrix"
	case FlagSlack:
		return "slack"
	case FlagNostr:
		return "nostr"
	case FlagIRC:
		return "irc"
	default:
		return "unknown"
	}
}

func process(ctx context.Context, pub Publisher, cols []wayback.Collect, args ...string) {
	// Compose the collects into multiple parts by URI
	var parts = make(map[string][]wayback.Collect)
	for _, col := range cols {
		parts[col.Src] = append(parts[col.Src], col)
	}

	f := from(args...)
	g, ctx := errgroup.WithContext(ctx)
	for _, part := range parts {
		logger.Debug("[%s] produce part: %#v", f, part)

		part := part
		g.Go(func() error {
			// Nice for target server. It should be skipped on the testing mode.
			if !strings.HasSuffix(os.Args[0], ".test") {
				rand.Seed(time.Now().UnixNano())
				r := rand.Intn(maxDelayTime) //nolint:gosec,goimports
				w := time.Duration(r) * time.Second
				logger.Debug("[%s] produce sleep %d second", f, r)
				time.Sleep(w)
			}

			ch := make(chan error, 1)
			go func() {
				ch <- pub.Publish(ctx, part, args...)
			}()

			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-ch:
				close(ch)
				return err
			}
		})
	}
	if err := g.Wait(); err != nil {
		logger.Error("[%s] produce failed: %v", f, err)
		return
	}
}

func from(args ...string) (f string) {
	if len(args) > 0 {
		f = args[0]
	}
	return f
}

// To publish to specific destination services
// nolint:gocyclo
func To(ctx context.Context, cols []wayback.Collect, args ...string) {
	f := from(args...)
	channel := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToChannel() {
			logger.Debug("[%s] publishing to telegram channel...", f)
			var bot *telegram.Bot
			if rev, ok := ctx.Value(FlagTelegram).(*telegram.Bot); ok {
				bot = rev
			}
			pub := NewTelegram(bot)
			process(ctx, pub, cols, args...)
		}
	}
	notion := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToNotion() {
			logger.Debug("[%s] publishing to Notion...", f)
			pub := NewNotion(nil)
			process(ctx, pub, cols, args...)
		}
	}
	issue := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToIssues() {
			logger.Debug("[%s] publishing to GitHub issues...", f)
			pub := NewGitHub(nil)
			process(ctx, pub, cols, args...)
		}
	}
	mastodon := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToMastodon() {
			logger.Debug("[%s] publishing to Mastodon...", f)
			var client *mstdn.Client
			if rev, ok := ctx.Value(FlagMastodon).(*mstdn.Client); ok {
				client = rev
			}
			pub := NewMastodon(client)
			process(ctx, pub, cols, args...)
		}
	}
	discord := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToDiscordChannel() {
			logger.Debug("[%s] publishing to Discord channel...", f)
			var s *discord.Session
			if rev, ok := ctx.Value(FlagDiscord).(*discord.Session); ok {
				s = rev
			}
			pub := NewDiscord(s)
			process(ctx, pub, cols, args...)
		}
	}
	matrix := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToMatrixRoom() {
			logger.Debug("[%s] publishing to Matrix room...", f)
			var client *matrix.Client
			if rev, ok := ctx.Value(FlagMatrix).(*matrix.Client); ok {
				client = rev
			}
			pub := NewMatrix(client)
			process(ctx, pub, cols, args...)
		}
	}
	twitter := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToTwitter() {
			logger.Debug("[%s] publishing to Twitter...", f)
			var client *twitter.Client
			if rev, ok := ctx.Value(FlagTwitter).(*twitter.Client); ok {
				client = rev
			}
			pub := NewTwitter(client)
			process(ctx, pub, cols, args...)
		}
	}
	slack := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToSlackChannel() {
			logger.Debug("[%s] publishing to Slack...", f)
			var client *slack.Client
			if rev, ok := ctx.Value(FlagTwitter).(*slack.Client); ok {
				client = rev
			}
			pub := NewSlack(client)
			process(ctx, pub, cols, args...)
		}
	}
	nostr := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToNostr() {
			logger.Debug("[%s] publishing to Nostr...", f)
			var client *nostr.Relay
			if rev, ok := ctx.Value(FlagNostr).(*nostr.Relay); ok {
				client = rev
			}
			pub := NewNostr(client)
			process(ctx, pub, cols, args...)
		}
	}
	irc := func(ctx context.Context, cols []wayback.Collect, args ...string) {
		if config.Opts.PublishToIRCChannel() {
			logger.Debug("[%s] publishing to IRC channel...", f)
			var conn *irc.Connection
			if rev, ok := ctx.Value(FlagIRC).(*irc.Connection); ok {
				conn = rev
			}
			pub := NewIRC(conn)
			process(ctx, pub, cols, args...)
		}
	}
	funcs := map[string]func(context.Context, []wayback.Collect, ...string){
		"channel":  channel,
		"notion":   notion,
		"issue":    issue,
		"mastodon": mastodon,
		"discord":  discord,
		"matrix":   matrix,
		"twitter":  twitter,
		"slack":    slack,
		"nostr":    nostr,
		"irc":      irc,
	}

	g, ctx := errgroup.WithContext(ctx)
	for k, fn := range funcs {
		logger.Debug(`[%s] processing func %s`, f, k)
		fn := fn
		g.Go(func() error {
			fn(ctx, cols, args...)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		logger.Error("[%s] process failed: %v", f, err)
	}
}

func extract(ctx context.Context, cols []wayback.Collect) (rdx reduxer.Reduxer, art reduxer.Artifact, err error) {
	if len(cols) == 0 {
		return rdx, art, errors.New("no collect")
	}

	var ok bool
	var uri = cols[0].Src
	if rdx, ok = ctx.Value(PubBundle{}).(reduxer.Reduxer); ok {
		if bundle, ok := rdx.Load(reduxer.Src(uri)); ok {
			return rdx, bundle.Artifact(), nil
		}
		return rdx, art, errors.New("reduxer data not found")
	}
	return rdx, art, errors.New("invalid reduxer")
}
