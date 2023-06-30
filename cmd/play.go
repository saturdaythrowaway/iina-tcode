package main

import "time"

func play(dir string) error {
	scripts := Scripts{
		preferedModifier: ScriptModSoft,
	}
	err := scripts.Load(dir)
	if err != nil {
		return err
	}

	tcode, err := scripts.TCode(Params{
		Min: 0.05,
		Max: 0.95,
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
