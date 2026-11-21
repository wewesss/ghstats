package collect_test

import (
	"context"
	"fmt"
	"os"

	"github.com/wewesss/ghstats/collect"
)

func ExampleCollect() {
	snap, err := collect.Collect(context.Background(), collect.Options{
		Token:   os.Getenv("GHSTATS_TOKEN"),
		Traffic: true,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(snap.Viewer.Login, len(snap.Repos))
}
