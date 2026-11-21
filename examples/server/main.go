// Command server serves a stats.json report and its cards over HTTP.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/wewesss/ghstats/config"
	"github.com/wewesss/ghstats/render"
	"github.com/wewesss/ghstats/report"
)

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func serve(contentType, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte(body))
	}
}

func main() {
	reportPath := flag.String("report", "", "path to a stats.json file")
	configPath := flag.String("config", "", "path to a ghstats.yaml file (defaults when omitted)")
	flag.Parse()

	data := must(os.ReadFile(*reportPath))
	var rep report.Report
	if err := json.Unmarshal(data, &rep); err != nil {
		log.Fatal(err)
	}
	cfg := config.Default()
	if *configPath != "" {
		cfg = must(config.Load(*configPath))
	}
	cfg.Render.BasePath = "/cards/"

	mux := http.NewServeMux()
	mux.HandleFunc("GET /stats.json", serve("application/json", string(data)))
	mux.HandleFunc("GET /{$}", serve("text/html; charset=utf-8", render.Snippet(cfg)))
	for _, mode := range []render.Mode{render.Dark, render.Light} {
		for name, svg := range must(render.Cards(rep, cfg, mode)) {
			mux.HandleFunc("GET /cards/"+name, serve("image/svg+xml", svg))
		}
	}
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}
