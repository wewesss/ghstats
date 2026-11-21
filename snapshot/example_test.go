package snapshot_test

import (
	"bytes"
	"fmt"

	"github.com/wewesss/ghstats/snapshot"
)

func ExampleWrite() {
	in := snapshot.Snapshot{
		SchemaVersion: snapshot.SchemaVersion,
		Viewer:        snapshot.Viewer{Login: "octocat", Followers: 3},
	}
	var buf bytes.Buffer
	if err := snapshot.Write(&buf, in); err != nil {
		fmt.Println(err)
		return
	}
	out, err := snapshot.Read(&buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(out.Viewer.Login, out.Viewer.Followers)
	// Output: octocat 3
}
