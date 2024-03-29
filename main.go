package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	netatmo "github.com/tipok/netatmo_exporter/netatmo-api"
)

func main() {
	var clientID string
	var clientSecret string
	var username string
	var password string
	var listen string
	var refreshToken string
	flag.StringVar(&clientID, "client-id", "", "Netatmo API client ID")
	flag.StringVar(&clientSecret, "client-secret", "", "Netatmo API client secret")
	flag.StringVar(&username, "username", "", "Netatmo username")
	flag.StringVar(&password, "password", "", "Netatmo password")
	flag.StringVar(&refreshToken, "refresh-token", "", "Netatmo refresh-token")
	flag.StringVar(&listen, "listen", ":2112", "Address to listen on")
	flag.Parse()

	if clientID == "" {
		log.Fatal("Netatmo API client ID has to be provided.")
	}

	if clientSecret == "" {
		log.Fatal("Netatmo API client secret has to be provided.")
	}

	refreshTokenUsed := false
	if refreshToken != "" {
		refreshTokenUsed = true
	}

	if username == "" && !refreshTokenUsed {
		log.Fatal("Netatmo username has to be provided.")
	}

	if password == "" && !refreshTokenUsed {
		log.Fatal("Netatmo password has to be provided.")
	}

	prometheus.MustRegister(version.NewCollector("netatmo_exporter"))

	cnf := &netatmo.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Username:     username,
		Password:     password,
		RefreshToken: refreshToken,
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
		log.Printf("Error during shutdown: %v", err)
	}
}
