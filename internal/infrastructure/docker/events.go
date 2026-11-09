package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/events"

	"github.com/zpol/doki/internal/domain/models"
)

// Events streams Docker daemon events until ctx is canceled.
func (c *Client) Events(ctx context.Context) (<-chan models.Event, <-chan error) {
	out := make(chan models.Event, 64)
	errs := make(chan error, 1)

	msgCh, errCh := c.cli.Events(ctx, events.ListOptions{})

	go func() {
		defer close(out)
		defer close(errs)
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-errCh:
				if !ok {
					return
				}
				if err != nil {
					errs <- fmt.Errorf("docker events: %w", err)
				}
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				out <- mapEvent(msg)
			}
		}
	}()

	return out, errs
}

func mapEvent(msg events.Message) models.Event {
	name := ""
	if msg.Actor.Attributes != nil {
		name = msg.Actor.Attributes["name"]
	}
	ts := time.Unix(msg.Time, 0).UTC()
	if msg.TimeNano > 0 {
		ts = time.Unix(0, msg.TimeNano).UTC()
	}
	action := string(msg.Action)
	typ := string(msg.Type)
	return models.Event{
		Time:       ts,
		Type:       typ,
		Action:     action,
		ActorID:    msg.Actor.ID,
		ActorName:  name,
		Attributes: msg.Actor.Attributes,
		Message:    fmt.Sprintf("%s %s %s", typ, action, firstNonEmpty(name, short(msg.Actor.ID))),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func short(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}
