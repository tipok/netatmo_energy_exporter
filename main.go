package main

import (
	"context"
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	netatmo "github.com/tipok/netatmo_exporter/netatmo-api"
)

func init() {
	prometheus.MustRegister(version.NewCollector("netatmo_exporter"))
}

func main() {
	var clientId string
	var clientSecret string
	var username string
	var password string
	var listen string
	flag.StringVar(&clientId, "client-id", "", "Netatmo API client ID")
	flag.StringVar(&clientSecret, "client-secret", "", "Netatmo API client secret")
	flag.StringVar(&username, "username", "", "Netatmo username")
	flag.StringVar(&password, "password", "", "Netatmo password")
	flag.StringVar(&listen, "listen", ":2112", "Address to listen on")
	flag.Parse()

	if clientId == "" {
		log.Fatal("Netatmo API client ID has to be provided.")
	}

	if clientSecret == "" {
		log.Fatal("Netatmo API client secret has to be provided.")
	}

	if username == "" {
		log.Fatal("Netatmo username has to be provided.")
	}

	if password == "" {
		log.Fatal("Netatmo password has to be provided.")
	}

	cnf := &netatmo.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Username:     username,
		Password:     password,
		Scopes:       []string{netatmo.ReadStation, netatmo.ReadThermostat},
	}
	client, err := netatmo.NewClient(context.Background(), cnf)
	if err != nil {
		log.Fatal(err)
	}

	collector := newCollector(client)
	prometheus.MustRegister(collector)

	sig := make(chan os.Signal, 1)
	signal.Notify(
		sig,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer signal.Stop(sig)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    listen,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
