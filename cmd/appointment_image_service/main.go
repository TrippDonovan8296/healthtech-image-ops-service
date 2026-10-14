package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/example/healthtech-image-ops-service/internal/appointment"
	"github.com/example/healthtech-image-ops-service/internal/infrai"
)

func main() {
	listen := flag.String("listen", ":8080", "HTTP listen address")
	bucket := flag.String("bucket", "", "storage bucket for optimized assets")
	flag.Parse()

	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	if *bucket == "" {
		log.Fatal("-bucket is required")
	}

	client := infrai.NewClient(apiKey)
	service := appointment.NewService(client, *bucket)
	mux := http.NewServeMux()
	mux.Handle("/appointments/optimize", appointment.NewOptimizeHandler(service))

	log.Printf("listening on %s", *listen)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Fatal(err)
	}
}
