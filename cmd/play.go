package main

import (
	"fmt"
	"time"
)

func play(filename string) error {
	scripts := Scripts{
		preferedModifier: ScriptModSoft,
	}

	err := scripts.Load(filename)
	if err != nil {
		return fmt.Errorf("%s: %w", "scripts.Load", err)
	}

	tcode, err := scripts.TCode()
	if err != nil {
		return fmt.Errorf("%s: %w", "scripts.TCode", err)
	}

	tcode.Seek(time.Duration(0))

	for msg := range tcode.Tick() {
		err = sendTCode(msg)
		if err != nil {
			return fmt.Errorf("%s: %w", "sendTCode", err)
		}
	}

	return nil
}
