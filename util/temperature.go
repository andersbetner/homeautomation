package util

import "errors"

// Temperature holds temperature data from various sensors
type Temperature struct {
	Outdoor           float64
	BathroomUpstairs  float64
	HallwayDownstairs float64
	HallwayUpstairs   float64
	Loft              float64
	Bedroom           float64
	Laundry           float64
}

// NewTemperature yadayada
func NewTemperature() *Temperature {
	ret := &Temperature{}
	ret.Outdoor = 99
	ret.BathroomUpstairs = 99
	ret.HallwayDownstairs = 99
	ret.HallwayUpstairs = 99
	ret.Loft = 99
	ret.Bedroom = 99
	ret.Laundry = 99

	return ret
}

// Set the temperature value
func (t *Temperature) Set(sensor string, value float64) (err error) {
	switch sensor {
	case "outdoor":
		t.Outdoor = value
	case "bathroomupstairs", "motion_badrum_uppe_temperature":
		t.BathroomUpstairs = value
	case "hallwaydownstairs", "motion_hall_nere_temperature":
		t.HallwayDownstairs = value
	case "hallwayupstairs", "motion_hall_uppe_temperature":
		t.HallwayUpstairs = value
	case "loft", "motion_loft_temperature":
		t.Loft = value
	case "laundry", "motion_tvattstuga_temperature":
		t.Laundry = value
	case "bedroom", "motion_sovrum_temperature":
		t.Bedroom = value

	default:
		return errors.New("No sensor named: " + sensor)
	}

	return nil
}
