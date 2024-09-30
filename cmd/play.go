package main

import (
	"fmt"
	"os"
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

	if os.Getenv("DEBUG") != "" {
		err = WriteImageFromTcode(tcode, "debug.png")
		if err != nil {
			return fmt.Errorf("%s: %w", "WriteImageFromTcode", err)
		}
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
