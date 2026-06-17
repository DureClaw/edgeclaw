//go:build !linux || loong64 || s390x

// Mock GPIO backend — console mock for non-Linux (macOS/Windows) and for Linux arches
// the gpiochip cdev uapi doesn't cover (loong64, s390x). Real GPIO is gpio_linux.go.
package main

func openGPIO(chip string, pin int, label string) gpioLine {
	return &mockLine{label: label, pin: pin}
}
