package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"

	"security-toolbox/internal/server"
)

// The frontend build output is copied into ./web by the Docker build and
// embedded here so the whole app ships as a single static binary.
//
//go:embed all:web
var webFS embed.FS

func main() {
	addr := flag.String("addr", envOr("LD_ADDR", ":8080"), "listen address")
	flag.Parse()

	// Sub-FS rooted at web/ so paths map to "/", "/assets/...".
	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("embed sub fs: %v", err)
	}

	h := server.New(staticFS)

	log.Printf("security-toolbox listening on %s (all conversion runs locally; no data is stored)", *addr)
	if err := http.ListenAndServe(*addr, h); err != nil {
		log.Fatal(err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
