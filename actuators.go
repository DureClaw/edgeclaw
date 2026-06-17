// Physical-edge actuators — the "approval = physical result" magic moment.
//
// edgeclaw turns a Raspberry Pi (or any box) into a real fleet node. When LED /
// buzzer / audio are configured, edgeclaw runs in *physical-edge mode*, a faithful
// port of the original pi-agent (executor@pi-cam):
//
//	join        → "online" voice (line watch started)
//	task.assign → "suspect" LED blink + voice, return a defect-suspect result
//	approved    → "quarantine" red LED + buzzer + voice  ← AI decision returns to the physical world
//
// GPIO is pure-Go via the gpiochip char device (Linux); elsewhere it degrades to a
// console mock so the same binary verifies on a laptop. Audio uses aplay/afplay and
// the guidance WAVs are embedded in the binary (//go:embed) — one file to install.
package main

import (
	"embed"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

//go:embed audio/online.wav audio/suspect.wav audio/approved.wav
var audioFS embed.FS

// gpioLine is the minimal LED/buzzer contract; backends are selected by build tag
// (gpio_linux.go uses the gpiochip cdev, gpio_other.go is a console mock).
type gpioLine interface {
	on()
	off()
	blink(n int)  // brief on/off pulses (suspect)
	beep(n int)   // short beeps (buzzer)
	close()
}

// Actuators bundles LED + buzzer + audio. The zero value (all nil) is a safe no-op.
type Actuators struct {
	led     gpioLine
	buzzer  gpioLine
	playCmd []string // e.g. ["aplay","-q"] or ["afplay"]; empty = silent
	dir     string   // extracted WAV dir
}

// actuatorsEnabled reports whether any physical output is configured (→ physical-edge mode).
func actuatorsEnabled() bool {
	return os.Getenv("EDGE_LED_PIN") != "" ||
		os.Getenv("EDGE_BUZZER_PIN") != "" ||
		os.Getenv("EDGE_AUDIO") != "" || os.Getenv("EDGE_AUDIO_DIR") != ""
}

func newActuators() *Actuators {
	a := &Actuators{}

	// ── GPIO (LED / buzzer) ──
	chip := env("EDGE_GPIO_CHIP", "gpiochip0")
	if pin, ok := pinEnv("EDGE_LED_PIN"); ok {
		a.led = openGPIO(chip, pin, "LED")
	}
	if pin, ok := pinEnv("EDGE_BUZZER_PIN"); ok && pin >= 0 {
		a.buzzer = openGPIO(chip, pin, "buzzer")
	}

	// ── audio ──
	if os.Getenv("EDGE_AUDIO") != "" || os.Getenv("EDGE_AUDIO_DIR") != "" {
		a.playCmd = resolvePlayCmd()
		a.dir = resolveAudioDir()
		if len(a.playCmd) == 0 {
			log.Printf("[edgeclaw] ⚠️  audio requested but no aplay/afplay found → silent")
		} else {
			log.Printf("[edgeclaw] 🔊 audio via %v (dir %s)", a.playCmd, a.dir)
		}
	}
	return a
}

func pinEnv(k string) (int, bool) {
	v := os.Getenv(k)
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("[edgeclaw] %s=%q invalid (want BCM pin number) — ignored", k, v)
		return 0, false
	}
	return n, true
}

// resolveAudioDir extracts the embedded WAVs to a temp dir unless EDGE_AUDIO_DIR
// points at an on-disk set (lets users swap the voice without rebuilding).
func resolveAudioDir() string {
	if d := os.Getenv("EDGE_AUDIO_DIR"); d != "" {
		return d
	}
	dir := filepath.Join(os.TempDir(), "edgeclaw-audio")
	_ = os.MkdirAll(dir, 0o755)
	for _, name := range []string{"online", "suspect", "approved"} {
		data, err := audioFS.ReadFile("audio/" + name + ".wav")
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(dir, name+".wav"), data, 0o644)
	}
	return dir
}

func resolvePlayCmd() []string {
	if c := os.Getenv("EDGE_PLAY_CMD"); c != "" {
		return splitFields(c)
	}
	if runtime.GOOS == "darwin" {
		if p, _ := exec.LookPath("afplay"); p != "" {
			return []string{"afplay"}
		}
	}
	if p, _ := exec.LookPath("aplay"); p != "" {
		return []string{"aplay", "-q"}
	}
	if p, _ := exec.LookPath("afplay"); p != "" {
		return []string{"afplay"}
	}
	return nil
}

func (a *Actuators) play(name string) {
	if a == nil || len(a.playCmd) == 0 || a.dir == "" {
		return
	}
	path := filepath.Join(a.dir, name+".wav")
	if _, err := os.Stat(path); err != nil {
		return
	}
	args := append(append([]string{}, a.playCmd[1:]...), path)
	cmd := exec.Command(a.playCmd[0], args...)
	cmd.Stdout, cmd.Stderr = nil, nil
	_ = cmd.Start() // non-blocking; we don't wait on the player
}

// ── named moments (mirror pi-agent) ──────────────────────────────────────────
func (a *Actuators) online() {
	log.Printf("[edgeclaw] 🟢 online — line watch started")
	a.play("online")
}

func (a *Actuators) suspectBlink() {
	log.Printf("[edgeclaw] 🟡 LED blink + 🔊 'defect suspect'")
	if a.led != nil {
		a.led.blink(6)
	}
	a.play("suspect")
}

func (a *Actuators) quarantineFire() {
	log.Printf("[edgeclaw] 🔴 LED on + 🔔 buzzer + 🔊 'quarantine APPROVED'")
	if a.led != nil {
		a.led.on()
	}
	if a.buzzer != nil {
		a.buzzer.beep(3)
	}
	a.play("approved")
}

func (a *Actuators) reset() {
	if a == nil {
		return
	}
	if a.led != nil {
		a.led.off()
		a.led.close()
	}
	if a.buzzer != nil {
		a.buzzer.close()
	}
}

func splitFields(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
