package solutions

import "strconv"

// ---- Exercise 1.1: constants ------------------------------------------------------------------

// Each constant is built from the previous one. All of them are computed at compile time.
const (
	SecondsPerMinute = 60
	SecondsPerHour   = 60 * SecondsPerMinute
	SecondsPerDay    = 24 * SecondsPerHour
)

// SplitDuration splits a number of seconds into whole days, hours, minutes and the seconds
// that remain.
func SplitDuration(total int) (days, hours, minutes, seconds int) {
	days = total / SecondsPerDay
	total %= SecondsPerDay // what is left after the whole days
	hours = total / SecondsPerHour
	total %= SecondsPerHour
	minutes = total / SecondsPerMinute
	seconds = total % SecondsPerMinute
	return days, hours, minutes, seconds
}

// ---- Exercise 1.2: an enum with iota -----------------------------------------------------------

// TrafficLight is an enum: a defined type plus a block of iota constants.
type TrafficLight int

const (
	Red    TrafficLight = iota // 0
	Yellow                     // 1
	Green                      // 2

	numLights // 3 - not a light: the number of lights, handy for wrapping around
)

// NextLight returns the light after l in the cycle Red -> Yellow -> Green -> Red.
func NextLight(l TrafficLight) TrafficLight {
	return (l + 1) % numLights // the constants are just numbers, so arithmetic works
}

// LightName returns "red", "yellow" or "green", and "unknown" for any other value.
func LightName(l TrafficLight) string {
	if l == Red {
		return "red"
	}
	if l == Yellow {
		return "yellow"
	}
	if l == Green {
		return "green"
	}
	return "unknown" // TrafficLight(7) compiles, so the function must handle it
}

// ---- Exercise 1.3: fix the shadowing bug -------------------------------------------------------

// SumStrings adds up numbers given as strings. On the first value that is not a number it
// returns 0 and that error.
//
// The buggy version declared "var err error" before the loop and then wrote
// "n, err := strconv.Atoi(v)" inside it. That := created a NEW err that lived only inside the
// loop, so the outer err stayed nil and the function reported success. The fix is to return
// the error right where it happens, so no outer variable is needed at all.
func SumStrings(values []string) (int, error) {
	total := 0
	for _, v := range values {
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}
