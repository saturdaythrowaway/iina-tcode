package main

import (
	"time"
)

func play(filename string) error {
	scripts := Scripts{
		preferedModifier: ScriptModSoft,
	}

	err := scripts.Load(filename)
	if err != nil {
		return err
	}

	tcode, err := scripts.TCode(Params{
		Min: 0.15,
		Max: 0.75,
	})
	if err != nil {
		return err
	}

	tcode.Seek(time.Duration(0))

	for msg := range tcode.Tick() {
		err = sendTCode(msg)
		if err != nil {
			return err
		}
	}

	return nil
}
