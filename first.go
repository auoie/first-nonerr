package firstnonerr

import (
	"context"
	"errors"
)

var (
	ErrNonErrNotFound = errors.New("non-error response not found")
)

func GetFirstNonError[T, R any](
	ctx context.Context,
	items []T,
	concurrent int,
	checker func(context.Context, T) (R, error)) (R, error) {
	if concurrent <= 0 {
		concurrent = len(items)
	}
	itemCh := make(chan T)
	badResponsesCh := make(chan struct{})
	firstValidResponseCh := make(chan R)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := 0; i < concurrent; i++ {
		go processItemRequests(ctx, itemCh, badResponsesCh, firstValidResponseCh, checker)
	}
	go sendItemRequests(ctx, items, itemCh)
	go processItemResponses(ctx, items, badResponsesCh, cancel)
	select {
	case <-ctx.Done():
		var nilR R
		return nilR, ErrNonErrNotFound
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
	badResponsesCh chan<- struct{},
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
				case firstValidResponseCh <- response:
				}
			} else {
				select {
				case <-ctx.Done():
				case badResponsesCh <- struct{}{}:
				}
			}
		}
	}
}

func processItemResponses[T any](
	ctx context.Context,
	items []T,
	badResponsesCh <-chan struct{},
	cancel context.CancelFunc) {
	defer cancel()
	for range items {
		select {
		case <-ctx.Done():
			return
		case <-badResponsesCh:
		}
	}
}
