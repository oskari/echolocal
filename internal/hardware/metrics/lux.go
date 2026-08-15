package metrics

import "os"

// LuxPath is the file the light sensor's reading comes from, empty when the board has none.
//
// Biscuit device trees declare more than one ALS and bind whichever chip is fitted. The reading
// shows up in different sysfs shapes depending on that binding:
//
//   - IIO channel files under iio:deviceN (e.g. tsl258x → illuminance0_input)
//   - Vendor attributes on the i2c device itself (e.g. tsl2540 → als_lux)
//
// Probe once at start-up for a readable lux file rather than assuming a fixed path.
func (r Reader) LuxPath() string {
	if at := r.firstLux(r.iioLuxFiles()); at != "" {
		return at
	}
	return r.firstLux(r.i2cLuxFiles())
}

// iioLuxFiles are standard IIO illuminance channel names, in preference order, on every present
// iio device. Newer kernels use in_illuminance_*; older Amazon drivers use illuminance0_input.
func (r Reader) iioLuxFiles() []string {
	return r.deviceAttrFiles("sys/bus/iio/devices", []string{
		"illuminance0_input",
		"in_illuminance_input",
	})
}

// i2cLuxFiles are vendor ALS attributes on i2c children. Prefer the calibrated reading when the
// driver publishes both.
func (r Reader) i2cLuxFiles() []string {
	return r.deviceAttrFiles("sys/bus/i2c/devices", []string{
		"als_calibrated_lux",
		"als_lux",
	})
}

// deviceAttrFiles joins each child of dir with each attr name, preserving attr preference order
// across devices (try every device's best attr before falling to the next attr).
func (r Reader) deviceAttrFiles(dir string, attrs []string) []string {
	entries, err := os.ReadDir(r.path(dir))
	if err != nil {
		return nil
	}
	var out []string
	for _, attr := range attrs {
		for _, entry := range entries {
			out = append(out, r.path(dir+"/"+entry.Name()+"/"+attr))
		}
	}
	return out
}

func (r Reader) firstLux(candidates []string) string {
	for _, at := range candidates {
		if _, err := number(at); err == nil {
			return at
		}
	}
	return ""
}

// Lux is how bright the room is, read from the file LuxPath found.
//
// The driver holds a converted value and hands back the last one, so a read costs about ten
// milliseconds rather than the integration time. There is no buffer, trigger or event to wait on, so
// asking is the only way to have the number.
func (r Reader) Lux(path string) Reading {
	if path == "" {
		return Reading{}
	}

	lux, err := number(path)
	if err != nil {
		return Reading{}
	}
	return known(lux)
}
