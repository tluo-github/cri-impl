package timeutil

import (
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"time"
)

func WithTimeout(d time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()

	ch := make(chan error, 1)
	func() {
		ch <- fn()
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "Time out")
	}
}
