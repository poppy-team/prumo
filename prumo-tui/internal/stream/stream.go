// Package stream carries the client's events into the terminal program.
//
// The services below the view layer publish on brokers; a Bubble Tea model can
// only be woken by a message arriving at the program. Without something that
// walks from one to the other, every event case in the model is dead code and
// the client draws a conversation that never grows, a permission gate that
// never opens and a log page that never fills. This package is that walk, and
// it is the only place in the client that knows the program exists.
package stream

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/raillen/prumo-tui/internal/app"
	"github.com/raillen/prumo-tui/internal/logging"
	"github.com/raillen/prumo-tui/internal/pubsub"
)

// Sender is the half of *tea.Program this package needs: somewhere to put a
// message. It is an interface so the bridge can be exercised without a
// terminal, which is what makes the delivery path testable on its own.
type Sender interface {
	Send(tea.Msg)
}

// Start forwards every broker the app publishes on into sink, until ctx is
// cancelled.
//
// The list is the app's whole publishing surface on purpose: a broker that is
// not forwarded is an event the view can never see, and that is the defect this
// package exists to fix. Adding a broker to the app therefore means adding a
// line here.
func Start(ctx context.Context, source *app.App, sink Sender) {
	forward(ctx, sink, source.Sessions.Subscribe(ctx))
	forward(ctx, sink, source.Messages.Subscribe(ctx))
	forward(ctx, sink, source.Permissions.Subscribe(ctx))
	forward(ctx, sink, source.CoderAgent.Subscribe(ctx))
	forward(ctx, sink, logging.Subscribe(ctx))
}

// forward pumps one broker into the program until either side goes away.
//
// The broker closes a subscription channel when its context ends, and the
// program stops accepting once it has quit, so the loop ends on whichever
// happens first rather than holding the process open.
func forward[T any](ctx context.Context, sink Sender, events <-chan pubsub.Event[T]) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}
				sink.Send(event)
			}
		}
	}()
}
