package netatmo_api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"
)

var (
	homesData, _   = url.Parse("https://api.netatmo.com/api/homesdata")
	homeStatus, _  = url.Parse("https://api.netatmo.com/api/homestatus")
	roomMeasure, _ = url.Parse("https://api.netatmo.com/api/getroommeasure")
	measure, _     = url.Parse("https://api.netatmo.com/api/getmeasure")
)

const (
	ReadThermostat = "read_thermostat"
	ReadStation    = "read_station"
)

func (c *Client) GetHomesData() (*HomesData, error) {
	var v HomesData
	if err := c.get(homesData, &v); err != nil {
		return nil, fmt.Errorf("could not get data: %w", err)
	}
	return &v, nil
}

func (c *Client) GetHomes() (*Homes, error) {
	homesData, err := c.GetHomesData()
	if err != nil {
		return nil, fmt.Errorf("could not get homes data: %w", err)
	}

	var homes []Home
	for _, home := range homesData.Homes {
		if homesStatus, err1 := c.GetHomeStatus(home.Id); err1 == nil {
			home.Merge(&homesStatus.Home)
		} else {
			log.Printf("Error during get home status: %v\n", err1)
			continue
		}
		homes = append(homes, home)
	}

	return &Homes{Homes: homes}, nil
}

func (c *Client) GetHomeStatus(home string) (*HomeStatus, error) {
	homeStatusUrl, err := url.Parse(homeStatus.String())
	if err != nil {
		return nil, err
	}
	q := homeStatusUrl.Query()
	q.Add("home_id", home)
	homeStatusUrl.RawQuery = q.Encode()

	var v HomeStatus
	if err := c.get(homeStatusUrl, &v); err != nil {
		return nil, fmt.Errorf("could not get data: %w", err)
	}

	return &v, nil
}

func (c *Client) getRoomMeasure(home string, room string) (interface{}, error) {
	roomMeasureUrl, err := url.Parse(roomMeasure.String())
	if err != nil {
		return nil, err
	}

	q := roomMeasureUrl.Query()
	q.Add("home_id", home)
	q.Add("room_id", room)
	q.Add("type", "heating_power_request,temperature,sp_temperature")
	q.Add("scale", "5min")
	q.Add("real_time", "true")
	now := time.Now()
	d := time.Duration(-8) * time.Hour
	q.Add("date_end", strconv.FormatInt(now.Unix(), 10))
	q.Add("date_begin", strconv.FormatInt(now.Add(d).Unix(), 10))
	roomMeasureUrl.RawQuery = q.Encode()

	var v interface{}
	if err := c.get(roomMeasureUrl, &v); err != nil {
		return nil, fmt.Errorf("could not get room measure data: %w", err)
	}
	return v, nil
}

func (c *Client) GetMeasure(bridge string, module string) (*ModuleMeasures, error) {
	measureUrl, err := url.Parse(measure.String())
	if err != nil {
		return nil, err
	}

	q := measureUrl.Query()
	q.Add("device_id", bridge)
	q.Add("module_id", module)
	q.Add("type", "sum_boiler_on,sum_boiler_off,temperature,sp_temperature")
	q.Add("scale", "5min")
	q.Add("real_time", "true")
	now := time.Now()
	d := time.Duration(-8) * time.Hour
	q.Add("date_end", strconv.FormatInt(now.Unix(), 10))
	q.Add("date_begin", strconv.FormatInt(now.Add(d).Unix(), 10))
	measureUrl.RawQuery = q.Encode()

	var objmap []map[string]*json.RawMessage
	if err := c.get(measureUrl, &objmap); err != nil {
		return nil, fmt.Errorf("could not get measure data: %w", err)
	}

	mps := parseModuleMeasurePoints(objmap)

	return &ModuleMeasures{Measures: mps}, nil
}

func parseModuleMeasurePoints(objmap []map[string]*json.RawMessage) []ModuleMeasurePoint {
	var mps []ModuleMeasurePoint
	for _, p := range objmap {
		var bt uint64
		if err := json.Unmarshal(*p["beg_time"], &bt); err != nil {
			log.Printf("Error during unmarshal of beg_time: %v\n", err)
			continue
		}
		var step uint32
		if err := json.Unmarshal(*p["step_time"], &step); err != nil {
			log.Printf("Error during unmarshal of step_time: %v\n", err)
			continue
		}
		if vr, ok := p["value"]; ok {
			var values [][]*json.RawMessage
			if err := json.Unmarshal(*vr, &values); err == nil {
				for i, value := range values {
					var bon uint16
					var boff uint16
					var t float64
					var spt float64
					if err := json.Unmarshal(*value[0], &bon); err != nil {
						log.Printf("Error during unmarshal first value: %v\n", err)
					}
					if err := json.Unmarshal(*value[1], &boff); err != nil {
						log.Printf("Error during unmarshal second value: %v\n", err)
					}
					if err := json.Unmarshal(*value[2], &t); err != nil {
						log.Printf("Error during unmarshal third value: %v\n", err)
					}
					if err := json.Unmarshal(*value[2], &spt); err != nil {
						log.Printf("Error during unmarshal fourth value: %v\n", err)
					}
					pt := bt + (uint64(step) * uint64(i))
					mp := ModuleMeasurePoint{
						Time:                pt,
						SumBoilerOn:         bon,
						SumBoilerOff:        boff,
						MeasuredTemperature: t,
						SetPointTemperature: spt,
					}
					mps = append(mps, mp)
				}
			} else {
				log.Printf("Error during unmarshal: %v\n", err)
			}
		}
	}
	return mps
}
