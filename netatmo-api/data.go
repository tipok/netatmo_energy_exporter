package netatmo_api

type HomesData struct {
	Homes []*Home `json:"homes"`
}

type Homes struct {
	Homes []*Home `json:"homes"`
}

type HomeStatus struct {
	Home *Home `json:"home"`
}

type Home struct {
	Altitude    uint32    `json:"altitude"`
	Country     string    `json:"country"`
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Coordinates []float64 `json:"coordinates"`
	Modules     []*Module `json:"modules"`
	Rooms       []*Room   `json:"rooms"`
}

type Module struct {
	Id               string  `json:"id"`
	Reachable        bool    `json:"reachable"`
	Type             string  `json:"type"`
	Bridge           string  `json:"bridge"`
	Anticipating     bool    `json:"anticipating"`
	FirmwareRevision float64 `json:"firmware_revision"`
	RfStrength       float64 `json:"rf_strength"`
	WifiStrength     float64 `json:"wifi_strength"`
	BatteryLevel     float64 `json:"battery_level"`
	BatteryState     string  `json:"battery_state"`
	BoilerStatus     bool    `json:"boiler_status"`
	RoomId           string  `json:"room_id"`
}

type Room struct {
	Reachable           bool    `json:"reachable"`
	Id                  string  `json:"id"`
	Name                string  `json:"name"`
	Anticipating        bool    `json:"anticipating"`
	OpenWindow          bool    `json:"open_window"`
	MeasuredTemperature float64 `json:"therm_measured_temperature"`
	SetPointTemperature float64 `json:"therm_setpoint_temperature"`
	SetPointStartTime   uint64  `json:"therm_setpoint_start_time"`
	SetPointEndTime     uint64  `json:"therm_setpoint_end_time"`
	SetPointMode        string  `json:"therm_setpoint_mode"`
}

type ModuleMeasures struct {
	Measures []*ModuleMeasurePoint `json:"measures"`
}

type ModuleMeasurePoint struct {
	Time                int64   `json:"time"`
	SumBoilerOn         uint16  `json:"sum_boiler_on"`
	SumBoilerOff        uint16  `json:"sum_boiler_off"`
	MeasuredTemperature float64 `json:"therm_measured_temperature"`
	SetPointTemperature float64 `json:"therm_setpoint_temperature"`
}

func (r *Room) Merge(r2 *Room) {
	if r.Reachable == false {
		r.Reachable = r2.Reachable
	}

	if r.Name == "" {
		r.Name = r2.Name
	}

	if r.Anticipating == false {
		r.Anticipating = r2.Anticipating
	}

	if r.OpenWindow == false {
		r.OpenWindow = r2.OpenWindow
	}

	if r.MeasuredTemperature == 0 {
		r.MeasuredTemperature = r2.MeasuredTemperature
	}

	if r.SetPointTemperature == 0 {
		r.SetPointTemperature = r2.SetPointTemperature
	}

	if r.SetPointStartTime == 0 {
		r.SetPointStartTime = r2.SetPointStartTime
	}

	if r.SetPointEndTime == 0 {
		r.SetPointEndTime = r2.SetPointEndTime
	}

	if r.SetPointMode == "" {
		r.SetPointMode = r2.SetPointMode
	}
}

func (m *Module) Merge(m2 *Module) {
	if m.Reachable == false {
		m.Reachable = m2.Reachable
	}

	if m.Anticipating == false {
		m.Anticipating = m2.Anticipating
	}

	if m.Type == "" {
		m.Type = m2.Type
	}

	if m.FirmwareRevision == 0 {
		m.FirmwareRevision = m2.FirmwareRevision
	}

	if m.RfStrength == 0 {
		m.RfStrength = m2.RfStrength
	}

	if m.BoilerStatus == false {
		m.BoilerStatus = m2.BoilerStatus
	}

	if m.WifiStrength == 0 {
		m.WifiStrength = m2.WifiStrength
	}

	if m.BatteryLevel == 0 {
		m.BatteryLevel = m2.BatteryLevel
	}

	if m.BatteryState == "" {
		m.BatteryState = m2.BatteryState
	}

	if m.RoomId == "" {
		m.RoomId = m2.RoomId
	}
}

func (h *Home) Merge(h2 *Home) {
	if h.Name == "" {
		h.Name = h2.Name
	}

	if h.Altitude == 0 {
		h.Altitude = h2.Altitude
	}

	if len(h.Coordinates) == 0 {
		h.Coordinates = h2.Coordinates
	}

	mergeRooms(h, h2)
	mergeModules(h, h2)
}

func mergeRooms(h *Home, h2 *Home) {
	if len(h.Rooms) == 0 {
		h.Rooms = h2.Rooms
		return
	}

	h2rm := make(map[string]*Room)
	for _, r := range h2.Rooms {
		h2rm[r.Id] = r
	}

	var rooms []*Room
	for _, r := range h.Rooms {
		if r2, ok := h2rm[r.Id]; ok {
			r.Merge(r2)
			delete(h2rm, r.Id)
		}
		rooms = append(rooms, r)
	}

	for _, r2 := range h2rm {
		rooms = append(rooms, r2)
	}

	h.Rooms = rooms
}

func mergeModules(h *Home, h2 *Home) {
	if len(h.Modules) == 0 {
		h.Modules = h2.Modules
		return
	}

	h2mm := make(map[string]*Module)
	for _, m := range h2.Modules {
		h2mm[m.Id] = m
	}

	var modules []*Module
	for _, m := range h.Modules {
		if m2, ok := h2mm[m.Id]; ok {
			m.Merge(m2)
			delete(h2mm, m.Id)
		}
		modules = append(modules, m)
	}

	for _, m2 := range h2mm {
		modules = append(modules, m2)
	}

	h.Modules = modules
}
