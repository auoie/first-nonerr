package firstnonerr

import (
	"context"
	"errors"
	"math/rand"
)

var (
	ErrNonErrNotFound = errors.New("non-error response not found")
	ErrEmpty          = errors.New("no items to check")
)

// Returns the first non-error result from a function `checker` applied to each entry in a list of `items`.
// If the list of items is empty, it returns `ErrEmpty`.
// If all of the results are errors, it randoms a random error from the error results.
// There are `concurrent` workers to apple the `checker` function.
// If `concurrent` is 0, then it will create `len(items)` workers.
func GetFirstNonError[T, R any](
	ctx context.Context,
	items []T,
	concurrent int,
	checker func(context.Context, T) (R, error)) (R, error) {
	if concurrent <= 0 {
		concurrent = len(items)
	}
	itemCh := make(chan T)
	badResponsesCh := make(chan error)
	randomErrCh := make(chan error)
	firstValidResponseCh := make(chan R)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := 0; i < concurrent; i++ {
		go processItemRequests(ctx, itemCh, badResponsesCh, firstValidResponseCh, checker)
	}
	go sendItemRequests(ctx, items, itemCh)
	go processItemResponses(ctx, items, badResponsesCh, randomErrCh)
	var nilR R
	select {
	case <-ctx.Done():
		return nilR, ErrNonErrNotFound
	case randomErr := <-randomErrCh:
		return nilR, randomErr
	case validItem := <-firstValidResponseCh:
		return validItem, nil
	}
}

func sendItemRequests[T any](
	ctx context.Context,
	items []T,
	itemCh chan<- T) {
	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		case itemCh <- item:
		}
	}
}

func processItemRequests[T, R any](
	ctx context.Context,
	itemCh <-chan T,
	badResponsesCh chan<- error,
	firstValidResponseCh chan<- R,
	checker func(context.Context, T) (R, error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-itemCh:
			response, err := checker(ctx, item)
			if err == nil {
				select {
				case <-ctx.Done():
					return
				case firstValidResponseCh <- response:
					return
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case badResponsesCh <- err:
				}
			}
		}
	}
}

func processItemResponses[T any](
	ctx context.Context,
	items []T,
	badResponsesCh <-chan error,
	finalErrorCh chan<- error) {
	nth := 0.0
	randomError := ErrEmpty
	for range items {
		nth += 1.0
		select {
		case <-ctx.Done():
			return
		case curErr := <-badResponsesCh:
			if nth*rand.Float64() <= 1.0 {
				randomError = curErr
			}
		}
	}
	select {
	case <-ctx.Done():
		return
	case finalErrorCh <- randomError:
	}
}
