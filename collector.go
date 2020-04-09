package main

import (
	"github.com/prometheus/client_golang/prometheus"
	netatmo "github.com/tipok/netatmo_exporter/netatmo-api"
	"log"
	"strconv"
	"time"
)

const (
	namespace       = "netatmo"
	subsystemModule = "module"
	subsystemRoom   = "room"
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
	client          *netatmo.Client
	cache           *cache
	up              prometheus.Gauge
	fwRevision      *prometheus.Desc
	boilerStatus    *prometheus.Desc
	reachableModule *prometheus.Desc
	reachableRoom   *prometheus.Desc
	temperature     *prometheus.Desc
	spTemperature   *prometheus.Desc
	sumBoilerOn     *prometheus.Desc
	sumBoilerOff    *prometheus.Desc
	wifiStrength    *prometheus.Desc
	rfStrength      *prometheus.Desc
	batteryLevel    *prometheus.Desc
	openWindow      *prometheus.Desc
	lastMeasure     *time.Time
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
	}

	varModuleLabels := append(
		varLabels,
		"bridge",
		"module",
		"type",
	)

	constLabels := prometheus.Labels{}

	return &Collector{
		client: client,
		cache:  &cache{entries: make(map[string]*cacheEntry)},

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Status of netatmo exporter",
		}),

		fwRevision: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemModule, "firmware_revision"),
			"Firmware revision of module",
			varModuleLabels,
			constLabels,
		),

		boilerStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemModule, "boiler_status"),
			"Status of the boiler",
			varModuleLabels,
			constLabels,
		),

		wifiStrength: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemModule, "wifi_strength"),
			"WiFi signal strength",
			varModuleLabels,
			constLabels,
		),

		rfStrength: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemModule, "rf_strength"),
			"Radio signal strength",
			varModuleLabels,
			constLabels,
		),

		batteryLevel: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemModule, "battery_level"),
			"Level of the battery",
			varModuleLabels,
			constLabels,
		),

		reachableModule: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemModule, "reachable"),
			"Tells if the module is currently reachable",
			varModuleLabels,
			constLabels,
		),

		reachableRoom: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemRoom, "reachable"),
			"Tells if the room is currently reachable",
			varLabels,
			constLabels,
		),

		openWindow: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemRoom, "open_window"),
			"Tells if the window is open.",
			varLabels,
			constLabels,
		),

		temperature: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemRoom, "temperature"),
			"Measured Temperature in a room",
			varLabels,
			constLabels,
		),

		spTemperature: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystemRoom, "sp_temperature"),
			"Set Point Temperature of a room",
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
	ch <- c.reachableModule
	ch <- c.reachableRoom
	ch <- c.openWindow
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {

	now := time.Now()
	if c.lastMeasure == nil {
		d := time.Duration(-1) * time.Hour
		from := now.Add(d)
		c.lastMeasure = &from
	}

	homes, err := c.client.GetHomes()
	if err != nil {
		log.Println(err)
		c.up.Set(0)
		ch <- c.up
		return
	}

	c.up.Set(1)
	ch <- c.up

	for _, home := range homes.Homes {
		labelsHome := []string{
			home.Id,
			home.Name,
			home.Country,
			strconv.FormatUint(uint64(home.Altitude), 10),
			strconv.FormatFloat(home.Coordinates[0], 'f', 8, 64),
			strconv.FormatFloat(home.Coordinates[1], 'f', 8, 64),
		}

		for _, m := range home.Modules {
			labelsModule := append(
				labelsHome,
				m.RoomId,
				m.Bridge,
				m.Id,
				m.Type,
			)

			ch <- prometheus.MustNewConstMetric(
				c.batteryLevel,
				prometheus.GaugeValue,
				m.BatteryLevel,
				labelsModule...,
			)

			ch <- prometheus.MustNewConstMetric(
				c.wifiStrength,
				prometheus.GaugeValue,
				m.WifiStrength,
				labelsModule...,
			)

			ch <- prometheus.MustNewConstMetric(
				c.rfStrength,
				prometheus.GaugeValue,
				m.RfStrength,
				labelsModule...,
			)

			var boilerStatus float64 = 0
			if m.BoilerStatus {
				boilerStatus = 1
			}

			ch <- prometheus.MustNewConstMetric(
				c.boilerStatus,
				prometheus.GaugeValue,
				boilerStatus,
				labelsModule...,
			)

			var reachable float64 = 0
			if m.Reachable {
				reachable = 1
			}

			ch <- prometheus.MustNewConstMetric(
				c.reachableModule,
				prometheus.GaugeValue,
				reachable,
				labelsModule...,
			)
		}

		for _, room := range home.Rooms {
			labelsRoom := []string{
				home.Id,
				home.Name,
				home.Country,
				strconv.FormatUint(uint64(home.Altitude), 10),
				strconv.FormatFloat(home.Coordinates[0], 'f', 8, 64),
				strconv.FormatFloat(home.Coordinates[1], 'f', 8, 64),
				room.Id,
			}

			ch <- prometheus.MustNewConstMetric(
				c.temperature,
				prometheus.GaugeValue,
				room.MeasuredTemperature,
				labelsRoom...,
			)

			ch <- prometheus.MustNewConstMetric(
				c.spTemperature,
				prometheus.GaugeValue,
				room.SetPointTemperature,
				labelsRoom...,
			)

			var reachable float64 = 0
			if room.Reachable {
				reachable = 1
			}

			ch <- prometheus.MustNewConstMetric(
				c.reachableRoom,
				prometheus.GaugeValue,
				reachable,
				labelsRoom...,
			)

			var openWindow float64 = 0
			if room.OpenWindow {
				openWindow = 1
			}

			ch <- prometheus.MustNewConstMetric(
				c.openWindow,
				prometheus.GaugeValue,
				openWindow,
				labelsRoom...,
			)
		}
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
			metrics = append(metrics, &home{home: h, module: m, measures: nil})
		}
	}
	return metrics, nil
}
