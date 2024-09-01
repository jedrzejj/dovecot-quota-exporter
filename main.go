package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	cliFlagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	metricsPath := cliFlagSet.String("metrics", "/metrics", "Metrics path")
	listenAddress := cliFlagSet.String("listen", "0.0.0.0:9901", "Metrics listen address")
	redisAddress := cliFlagSet.String("redis", "127.0.0.1:6379", "Redis address")
	redisDb := cliFlagSet.Int("db", 0, "Redis Database ID (default 0)")

	err := cliFlagSet.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("Flags parsing failed with error: %v", err)

		return
	}

	log.Print("Exporter starting")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exporter := New(*redisAddress, *redisDb)
	exporter.Start(ctx)

	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Dovecot Exporter</title></head>
			<body>
			<h1>Dovecot Quota Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
	})

	srv := http.Server{
		Addr: *listenAddress,
	}

	go func() {
		err = srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Listen error: %v", err)
		}
		log.Print("Listener closed")
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Print("Exporter stopping")

	err = srv.Shutdown(ctx)
	if err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	cancel()
}
