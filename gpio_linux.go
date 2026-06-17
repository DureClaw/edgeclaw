//go:build linux && !loong64 && !s390x

// Linux GPIO backend — pure-Go via the gpiochip character device (works on current
// Raspberry Pi OS, no sysfs, no CGo, no libgpiod install). Falls back to the console
// mock if the chip/line can't be opened (e.g. not a Pi, or insufficient permissions).
package main

import (
	"log"
	"time"

	gpiocdev "github.com/warthog618/go-gpiocdev"
)

type cdevLine struct {
	line  *gpiocdev.Line
	label string
}

func openGPIO(chip string, pin int, label string) gpioLine {
	l, err := gpiocdev.RequestLine(chip, pin,
		gpiocdev.AsOutput(0),
		gpiocdev.WithConsumer("edgeclaw"))
	if err != nil {
		log.Printf("[edgeclaw] gpio %s pin %d on %s unavailable → mock (%v)", label, pin, chip, err)
		return &mockLine{label: label, pin: pin}
	}
	log.Printf("[edgeclaw] gpio %s on %s pin %d", label, chip, pin)
	return &cdevLine{line: l, label: label}
}

func (c *cdevLine) on()  { _ = c.line.SetValue(1) }
func (c *cdevLine) off() { _ = c.line.SetValue(0) }

func (c *cdevLine) blink(n int) {
	go func() {
		for i := 0; i < n; i++ {
			_ = c.line.SetValue(1)
			time.Sleep(200 * time.Millisecond)
			_ = c.line.SetValue(0)
			time.Sleep(200 * time.Millisecond)
		}
	}()
}

func (c *cdevLine) beep(n int) {
	go func() {
		for i := 0; i < n; i++ {
			_ = c.line.SetValue(1)
			time.Sleep(150 * time.Millisecond)
			_ = c.line.SetValue(0)
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

func (c *cdevLine) close() {
	if c.line != nil {
		_ = c.line.SetValue(0)
		_ = c.line.Close()
	}
}
