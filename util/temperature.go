package util

import "errors"

// Temperature holds temperature data from various sensors
type Temperature struct {
	Outdoor     float64
	Outdoorroom float64
	Shower      float64
	Hall        float64
	Kitchen     float64
	Loft        float64
	Lowe        float64
	Bedroom     float64
}

func NewTemperature() *Temperature {
	ret := &Temperature{}
	ret.Outdoor = 99
	ret.Outdoorroom = 99
	ret.Shower = 99
	ret.Hall = 99
	ret.Kitchen = 99
	ret.Loft = 99
	ret.Lowe = 99
	ret.Bedroom = 99

	return ret
}

// Set the temperature value
func (t *Temperature) Set(sensor string, value float64) (err error) {
	switch sensor {
	case "outdoor":
		t.Outdoor = value
	case "outdoorroom":
		t.Outdoorroom = value
	case "shower":
		t.Shower = value
	case "hall":
		t.Hall = value
	case "kitchen":
		t.Kitchen = value
	case "loft":
		t.Loft = value
	case "lowe":
		t.Lowe = value
	case "bedroom":
		t.Bedroom = value
	default:
		return errors.New("No sensor named: " + sensor)
	}

	return nil
}
