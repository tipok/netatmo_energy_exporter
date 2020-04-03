package main

import (
	"github.com/prometheus/client_golang/prometheus"
	netatmo "github.com/tipok/netatmo_exporter/netatmo-api"
	"log"
	"strconv"
	"time"
)

const (
	namespace = "netatmo"
	subsystem = ""
)

type home struct {
	home     *netatmo.Home
	module   *netatmo.Module
	measures *netatmo.ModuleMeasures
}

type cache struct {
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	home    *netatmo.Home
	module  *netatmo.Module
	measure *netatmo.ModuleMeasurePoint
}

type Collector struct {
	client        *netatmo.Client
	cache         *cache
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
		cache: &cache{entries: make(map[string]*cacheEntry)},
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
		d := time.Duration(-1) * time.Hour
		from := now.Add(d)
		c.lastMeasure = &from
	}

	homes, err := collectModulesWithMeasures(c.client, *c.lastMeasure, now)
	if err != nil {
		log.Println(err)
		c.up.Set(0)
		ch <- c.up
		return
	}

	c.up.Set(1)
	ch <- c.up

	for _, h := range homes {
		var ce *cacheEntry
		if ce1, ok := c.cache.entries[h.home.Id]; ok {
			ce = ce1
		} else {
			ce = &cacheEntry{}
		}
		ce.home = h.home
		ce.module = h.module

		c.cache.entries[h.home.Id] = ce
		if h.measures == nil || len(h.measures.Measures) == 0 {
			continue
		}
		c.lastMeasure = &now
		c.cache.entries[h.home.Id].measure = h.measures.Measures[len(h.measures.Measures)-1]
	}


	for _, ce := range c.cache.entries {
		labels := []string{
			ce.home.Id,
			ce.home.Name,
			ce.home.Country,
			strconv.FormatUint(uint64(ce.home.Altitude), 10),
			strconv.FormatFloat(ce.home.Coordinates[0], 'f', 8, 64),
			strconv.FormatFloat(ce.home.Coordinates[1], 'f', 8, 64),
			ce.module.RoomId,
			ce.module.Bridge,
			ce.module.Id,
			ce.module.Type,
		}

		ch <- prometheus.MustNewConstMetric(
			c.batteryLevel,
			prometheus.GaugeValue,
			ce.module.BatteryLevel,
			labels...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.wifiStrength,
			prometheus.GaugeValue,
			ce.module.WifiStrength,
			labels...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.rfStrength,
			prometheus.GaugeValue,
			ce.module.RfStrength,
			labels...,
		)

		var boilerStatus float64 = 0
		if ce.module.BoilerStatus {
			boilerStatus = 1
		}

		ch <- prometheus.MustNewConstMetric(
			c.boilerStatus,
			prometheus.GaugeValue,
			boilerStatus,
			labels...,
		)

		var reachable float64 = 0
		if ce.module.Reachable {
			reachable = 1
		}

		ch <- prometheus.MustNewConstMetric(
			c.reachable,
			prometheus.GaugeValue,
			reachable,
			labels...,
		)

		temp := prometheus.MustNewConstMetric(
			c.temperature,
			prometheus.GaugeValue,
			ce.measure.MeasuredTemperature,
			labels...,
		)

		spTemp := prometheus.MustNewConstMetric(
			c.spTemperature,
			prometheus.GaugeValue,
			ce.measure.SetPointTemperature,
			labels...,
		)

		bon := prometheus.MustNewConstMetric(
			c.sumBoilerOn,
			prometheus.GaugeValue,
			float64(ce.measure.SumBoilerOn),
			labels...,
		)

		boff := prometheus.MustNewConstMetric(
			c.sumBoilerOff,
			prometheus.GaugeValue,
			float64(ce.measure.SumBoilerOff),
			labels...,
		)
		t := time.Unix(ce.measure.Time, 0)
		ch <- prometheus.NewMetricWithTimestamp(t, temp)
		ch <- prometheus.NewMetricWithTimestamp(t, spTemp)
		ch <- prometheus.NewMetricWithTimestamp(t, bon)
		ch <- prometheus.NewMetricWithTimestamp(t, boff)
	}
}

func collectModulesWithMeasures(client *netatmo.Client, from time.Time, until time.Time) ([]*home, error) {
	var metrics []*home
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
			metrics = append(metrics, &home{home: h, module: m, measures: measure})
		}
	}
	return metrics, nil
}