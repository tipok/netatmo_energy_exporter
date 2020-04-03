package main

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	netatmo "github.com/tipok/netatmo_exporter/netatmo-api"
)

const (
	namespace = "netatmo"
	subsystem = ""
)

type metric struct {
	home     *netatmo.Home
	module   *netatmo.Module
	measures *netatmo.ModuleMeasures
}

type Collector struct {
	client        *netatmo.Client
	up            prometheus.Gauge
	fwRevision    *prometheus.Desc
	boilerStatus  *prometheus.Desc
	reachable     *prometheus.Desc
	temperature   *prometheus.Desc
	spTemperature *prometheus.Desc
	sumBoilerOn   *prometheus.Desc
	sumBoilerOff  *prometheus.Desc
	wifiStrength  *prometheus.Desc
	rfStrength    *prometheus.Desc
	batteryLevel  *prometheus.Desc
	lastMeasure   *time.Time
}

func newCollector(client *netatmo.Client) *Collector {
	varLabels := []string{
		"home_id",
		"home_name",
		"home_country",
		"home_altitude",
		"home_lat",
		"home_long",
		"room_id",
		"bridge",
		"module",
		"type",
	}
	constLabels := prometheus.Labels{}

	return &Collector{
		client: client,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Status of netatmo exporter",
		}),
		fwRevision: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "firmware_revision"),
			"Firmware revision of module",
			varLabels,
			constLabels,
		),
		reachable: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "reachable"),
			"Tells if the module is currently reachable",
			varLabels,
			constLabels,
		),
		boilerStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "boiler_status"),
			"Status of the boiler",
			varLabels,
			constLabels,
		),
		temperature: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "temperature"),
			"Measured Temperature",
			varLabels,
			constLabels,
		),
		spTemperature: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "sp_temperature"),
			"Set Point Temperature",
			varLabels,
			constLabels,
		),
		wifiStrength: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "wifi_strength"),
			"WiFi signal strength",
			varLabels,
			constLabels,
		),
		rfStrength: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "rf_strength"),
			"Radio signal strength",
			varLabels,
			constLabels,
		),
		batteryLevel: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "battery_level"),
			"Level of the battery",
			varLabels,
			constLabels,
		),
		sumBoilerOn: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "sum_boiler_on"),
			"Summery of boiler being on over time",
			varLabels,
			constLabels,
		),
		sumBoilerOff: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "sum_boiler_off"),
			"Summery of boiler being off over time",
			varLabels,
			constLabels,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up.Desc()
	ch <- c.temperature
	ch <- c.spTemperature
	ch <- c.boilerStatus
	ch <- c.fwRevision
	ch <- c.rfStrength
	ch <- c.wifiStrength
	ch <- c.batteryLevel
	ch <- c.reachable
	ch <- c.sumBoilerOn
	ch <- c.sumBoilerOff
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {

	now := time.Now()
	if c.lastMeasure == nil {
		d := time.Duration(-5) * time.Minute
		from := now.Add(d)
		c.lastMeasure = &from
	}

	metrics, err := collectModulesWithMeasures(c.client, *c.lastMeasure, now)
	if err != nil {
		log.Println(err)
		c.up.Set(0)
		ch <- c.up
		return
	}
	c.lastMeasure = &now

	c.up.Set(1)
	ch <- c.up

	for _, m := range metrics {
		labels := []string{
			m.home.Id,
			m.home.Name,
			m.home.Country,
			strconv.FormatUint(uint64(m.home.Altitude), 10),
			strconv.FormatFloat(m.home.Coordinates[0], 'f', 8, 64),
			strconv.FormatFloat(m.home.Coordinates[1], 'f', 8, 64),
			m.module.RoomId,
			m.module.Bridge,
			m.module.Id,
			m.module.Type,
		}

		ch <- prometheus.MustNewConstMetric(
			c.batteryLevel,
			prometheus.GaugeValue,
			m.module.BatteryLevel,
			labels...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.wifiStrength,
			prometheus.GaugeValue,
			m.module.WifiStrength,
			labels...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.rfStrength,
			prometheus.GaugeValue,
			m.module.RfStrength,
			labels...,
		)

		var boilerStatus float64 = 0
		if m.module.BoilerStatus {
			boilerStatus = 1
		}

		ch <- prometheus.MustNewConstMetric(
			c.boilerStatus,
			prometheus.GaugeValue,
			boilerStatus,
			labels...,

		)

		var reachable float64 = 0
		if m.module.Reachable {
			reachable = 1
		}

		ch <- prometheus.MustNewConstMetric(
			c.reachable,
			prometheus.GaugeValue,
			reachable,
			labels...,
		)

		if m.measures == nil || len(m.measures.Measures) == 0 {
			continue
		}

		mp := m.measures.Measures[len(m.measures.Measures)-1]

		temp := prometheus.MustNewConstMetric(
			c.temperature,
			prometheus.GaugeValue,
			mp.MeasuredTemperature,
			labels...,
		)

		spTemp := prometheus.MustNewConstMetric(
			c.spTemperature,
			prometheus.GaugeValue,
			mp.SetPointTemperature,
			labels...,
		)

		bon := prometheus.MustNewConstMetric(
			c.sumBoilerOn,
			prometheus.GaugeValue,
			float64(mp.SumBoilerOn),
			labels...,
		)

		boff := prometheus.MustNewConstMetric(
			c.sumBoilerOff,
			prometheus.GaugeValue,
			float64(mp.SumBoilerOff),
			labels...,
		)
		t := time.Unix(mp.Time, 0)
		ch <- prometheus.NewMetricWithTimestamp(t, temp)
		ch <- prometheus.NewMetricWithTimestamp(t, spTemp)
		ch <- prometheus.NewMetricWithTimestamp(t, bon)
		ch <- prometheus.NewMetricWithTimestamp(t, boff)
	}
}

func collectModulesWithMeasures(client *netatmo.Client, from time.Time, until time.Time) ([]*metric, error) {
	var metrics []*metric
	homes, err := client.GetHomes()
	if err != nil {
		return nil, err
	}
	for _, h := range homes.Homes {
		for _, m := range h.Modules {
			measure, err := client.GetMeasure(m, from, until)
			if err != nil {
				log.Printf("Error getting info for module %v (%v): %v", m.Id, m.Type, err)
			}
			metrics = append(metrics, &metric{home: h, module: m, measures: measure})
		}
	}
	return metrics, nil
}

func init() {
	prometheus.MustRegister(version.NewCollector("netatmo_exporter"))
}

func main() {
	cnf := &netatmo.Config{
		ClientID:     "",
		ClientSecret: "",
		Username:     "",
		Password:     "",
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
		Addr:    ":2112",
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
