package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	firstnonerr "github.com/auoie/first-nonerr"
)

var (
	errInvalidUrls = errors.New("no valid url found")
	portPointer    = flag.String("PORT", "3000", "port to check")
)

func main() {
	flag.Parse()
	port := *portPointer
	items := []int{}
	for i := 0; i < 256; i++ {
		items = append(items, i)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	url, err := firstnonerr.GetFirstNonError(
		context.Background(),
		items,
		0,
		func(ctx context.Context, item int) (int, error) {
			url := fmt.Sprint("http://192.168.1.", item, ":", port)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return 0, err
			}
			resp, err := client.Do(req)
			if err != nil {
				return 0, err
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				return item, nil
			}
			return 0, errInvalidUrls
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(fmt.Sprint("http://192.168.1.", url, ":", port))
}
