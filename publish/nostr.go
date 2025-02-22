// Copyright 2023 Wayback Archiver. All rights reserved.
// Use of this source code is governed by the GNU GPL v3
// license that can be found in the LICENSE file.

package publish // import "github.com/wabarc/wayback/publish"

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/wabarc/logger"
	"github.com/wabarc/wayback"
	"github.com/wabarc/wayback/config"
	"github.com/wabarc/wayback/errors"
	"github.com/wabarc/wayback/metrics"
	"github.com/wabarc/wayback/template/render"
	"golang.org/x/sync/errgroup"
)

var _ Publisher = (*nostrBot)(nil)

type nostrBot struct {
	client *nostr.Relay
}

// NewNostr returns a Nostr client.
func NewNostr(client *nostr.Relay) *nostrBot {
	if !config.Opts.PublishToNostr() {
		logger.Error("Missing required environment variable, abort.")
		return new(nostrBot)
	}

	return &nostrBot{client: client}
}

// Publish publish text to the Nostr of given cols and args.
// A context should contain a `reduxer.Reduxer` via `publish.PubBundle` struct.
func (n *nostrBot) Publish(ctx context.Context, cols []wayback.Collect, args ...string) error {
	metrics.IncrementPublish(metrics.PublishNostr, metrics.StatusRequest)

	if len(cols) == 0 {
		return errors.New("publish to nostr: collects empty")
	}

	rdx, _, err := extract(ctx, cols)
	if err != nil {
		logger.Warn("extract data failed: %v", err)
	}

	body := render.ForPublish(&render.Nostr{Cols: cols, Data: rdx}).String()
	if err = n.publish(ctx, strings.TrimSpace(body)); err != nil {
		metrics.IncrementPublish(metrics.PublishNostr, metrics.StatusFailure)
		return errors.New("publish to nostr failed: %v", err)
	}
	metrics.IncrementPublish(metrics.PublishNostr, metrics.StatusSuccess)
	return nil
}

func (n *nostrBot) publish(ctx context.Context, note string) error {
	if !config.Opts.PublishToNostr() {
		return fmt.Errorf("publish to nostr abort")
	}

	if note == "" {
		return fmt.Errorf("nostr validation failed: note can't be blank")
	}
	logger.Debug("send to nostr, note:\n%s", note)

	sk := config.Opts.NostrPrivateKey()
	if strings.HasPrefix(sk, "nsec") {
		if _, s, e := nip19.Decode(sk); e == nil {
			sk = s.(string)
		} else {
			return fmt.Errorf("decode private key failed")
		}
	}
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		return fmt.Errorf("failed to get public key: %v", err)
	}
	ev := nostr.Event{
		Kind:      1,
		Content:   note,
		CreatedAt: time.Now(),
		PubKey:    pk,
		// Tags:      nostr.Tags{[]string{"foo", "bar"}},
	}
	if err := ev.Sign(sk); err != nil {
		return fmt.Errorf("calling sign err: %v", err)
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, relay := range config.Opts.NostrRelayURL() {
		logger.Debug(`publish note to relay: %s`, relay)
		relay := relay
		g.Go(func() error {
			defer func() {
				// recover from upstream panic
				if r := recover(); r != nil {
					logger.Error("publish to %s failed: %v", relay, r)
				}
			}()
			client := relayConnect(ctx, relay)
			if client.Connection == nil {
				return fmt.Errorf("publish to %s failed: %v", relay, <-client.ConnectionError)
			}
			// send the text note
			status := client.Publish(ctx, ev)
			if status != nostr.PublishStatusSucceeded {
				return fmt.Errorf("published to %s status is %s, not %s", relay, status, nostr.PublishStatusSucceeded)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func relayConnect(ctx context.Context, url string) *nostr.Relay {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	relay, err := nostr.RelayConnect(ctx, url)
	if err != nil {
		logger.Error("Connect to Nostr relay server got unpredictable error: %v", err)
	}
	return relay
}
