package main

import "time"

type Params struct {
	Min, Max float64

	Offset time.Duration

	PreferSoft bool
	PreferHard bool
	PreferAlt  bool
}

var params = Params{
	Min: 0.15,
	Max: 0.75,

	Offset: time.Duration(0),

	PreferSoft: false,
	PreferHard: false,
	PreferAlt:  false,
}
