package phone

import (
	"fmt"
	"io"

	"rsc.io/qr"
)

const qrQuiet = 2 // white modules around the code so phones can find it

// writeQR prints text as a QR code using half blocks, forcing black on white
// so it scans on dark and light terminals alike.
func writeQR(w io.Writer, text string) error {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return err
	}
	black := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < code.Size && y < code.Size && code.Black(x, y)
	}
	for y := -qrQuiet; y < code.Size+qrQuiet; y += 2 {
		line := ""
		for x := -qrQuiet; x < code.Size+qrQuiet; x++ {
			switch top, bottom := black(x, y), black(x, y+1); {
			case top && bottom:
				line += "█"
			case top:
				line += "▀"
			case bottom:
				line += "▄"
			default:
				line += " "
			}
		}
		if _, err := fmt.Fprintf(w, "\x1b[30;47m%s\x1b[0m\n", line); err != nil {
			return err
		}
	}
	return nil
}
