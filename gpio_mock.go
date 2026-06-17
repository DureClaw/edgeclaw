// Console-mock GPIO line — shared by the non-Linux backend and as the Linux fallback
// when a real chip/line can't be opened. Lets the physical-edge logic run anywhere.
package main

import "log"

type mockLine struct {
	label string
	pin   int
}

func (m *mockLine) on()        { log.Printf("[edgeclaw][mock] %s(pin %d) ON", m.label, m.pin) }
func (m *mockLine) off()       { log.Printf("[edgeclaw][mock] %s(pin %d) off", m.label, m.pin) }
func (m *mockLine) blink(n int) { log.Printf("[edgeclaw][mock] %s(pin %d) blink x%d", m.label, m.pin, n) }
func (m *mockLine) beep(n int)  { log.Printf("[edgeclaw][mock] %s(pin %d) beep x%d", m.label, m.pin, n) }
func (m *mockLine) close()      {}
