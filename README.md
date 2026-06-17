# edgeclaw

> The **DureClaw-native edge agent** — one small, static Go binary that turns
> **any** machine into a member of a collaborating AI fleet.
> Windows · macOS · Linux · Raspberry Pi · riscv64 · loongarch · mips … one codebase, every CPU.

Built on **[DureClaw](https://github.com/DureClaw/dureclaw)** (the Phoenix-Channel collaboration bus).

```
┌─ any box (win/mac/linux/pi/riscv) ─┐        ┌─ master ─┐
│  edgeclaw  ──task.assign──────────────▶  DureClaw bus  │
│   • local hands: shell · sensor      │        │  brain  │  (keyless LLM)
│   • LLM: keyless → master  | ollama  ◀──task.result────┘
└────────────────────────────────────┘
```

## Why Go

edgeclaw's goal is **support every OS and every CPU**. Go is the sweet spot:
**single static binary, no runtime, trivial cross-compile, broad arch coverage** —
proven here building for `linux/arm64`, `linux/armv6` (Pi Zero), `linux/riscv64`,
`windows/amd64`, `darwin/*` from one `make all`. One binary, every box — no runtime, no fallback needed.

## What it does

- **Joins the bus** (Phoenix Channel, `vsn=2.0.0`) with presence + heartbeat.
- **Receives `task.assign`**, runs the work, returns `task.result`.
- **Local hands:** `[SHELL] <cmd>` runs on the device (sensor/GPIO/scripts).
- **LLM, keyless by default:** delegates inference to the **master brain**
  (`BRAIN_URL`/brain/exec) so the edge holds **no API key, no cost**.
  Or use a **local** provider (`OLLAMA_URL`), or wrap **any CLI** (`AGENT_CMD`).
- **Physical-edge mode** (set any of `EDGE_LED_PIN` / `EDGE_BUZZER_PIN` / `EDGE_AUDIO`):
  drives a real LED + buzzer + voice on a Raspberry Pi — **"approval = physical result"**.
  GPIO is pure-Go over the gpiochip char device (no sysfs, no libgpiod, no CGo); off-Pi
  it degrades to a console mock so the same binary verifies on a laptop.

## Physical-edge mode (Raspberry Pi)

One static binary replaces the old python `pi-agent` (no `pip`, `gpiozero`, or
`websocket-client`). The guidance WAVs are **embedded in the binary** — nothing else to copy.

```bash
EDGE_LED_PIN=17 EDGE_BUZZER_PIN=18 EDGE_AUDIO=1 \
  STATE_SERVER=<bus-host:4000> OAH_SECRET=<token> WORK_KEY=<WK> \
  AGENT_NAME=executor@pi-cam ./edgeclaw
```

| Bus event | Physical reaction |
|-----------|-------------------|
| join ack | 🟢 `online` voice — line watch started |
| `task.assign` | 🟡 `suspect` LED blink + voice, returns a defect-suspect `task.result` |
| `task.result {approved, quarantine}` (broadcast) | 🔴 LED on + 🔔 buzzer + 🔊 `approved` voice |
| `state.update {decision_<lot>: quarantine}` | same quarantine fire (fallback) |

Verify on a laptop (mock GPIO, silent player): add `EDGE_PLAY_CMD=true`.

## Install (prebuilt — no toolchain needed)

One line, auto-detects your OS/CPU and downloads the matching binary:

```sh
curl -fsSL https://github.com/DureClaw/edgeclaw/releases/latest/download/install.sh | sh
```

Or grab a binary directly from **[Releases](https://github.com/DureClaw/edgeclaw/releases/latest)** —
win/mac/linux × `amd64` · `arm64` · `armv6`(Pi Zero) · `armv7` · `riscv64` · `loong64` · `mips64le`
(SHA256SUMS attached). Build from source: `go build -o edgeclaw .` or `make build`.

## Quick start

```bash
STATE_SERVER=<bus-host:4000> OAH_SECRET=<token> WORK_KEY=WK-demo \
  AGENT_NAME=edgeclaw@$(hostname) \
  BRAIN_URL=http://<master>:4111 BRAIN_TOKEN=<tok> \
  ./edgeclaw
```

Then a master can fan-out a task to it (and to many others) and fan-in the results.

### Cross-compile for the whole fleet

```bash
make all     # → dist/edgeclaw-{linux-amd64,linux-arm64,linux-armv6,linux-riscv64,
             #     linux-loong64,linux-mips64le,darwin-arm64,windows-amd64.exe, …}
```

## Configuration (env)

| Env | Meaning |
|-----|---------|
| `STATE_SERVER` | DureClaw bus `host:port` |
| `OAH_SECRET` | bus bearer token |
| `WORK_KEY` | collaboration session key |
| `AGENT_NAME` / `AGENT_ROLE` / `CAPABILITIES` | fleet identity |
| `BRAIN_URL` / `BRAIN_TOKEN` | master brain endpoint — **keyless LLM** (default path) |
| `OLLAMA_URL` / `OLLAMA_MODEL` | local LLM instead of the master |
| `AGENT_CMD` | wrap any external CLI; `{}` ← instruction |
| `EDGE_LED_PIN` / `EDGE_BUZZER_PIN` | BCM pins → physical-edge mode (gpiochip cdev) |
| `EDGE_GPIO_CHIP` | gpiochip device (default `gpiochip0`) |
| `EDGE_AUDIO` / `EDGE_AUDIO_DIR` / `EDGE_PLAY_CMD` | voice guidance (embedded WAVs; override dir/player) |

## Status

Verified end-to-end against a live DureClaw bus: keyless LLM delegation
(`backend=brain-remote`) **and** local shell hands, on macOS arm64. Cross-builds
clean for win/mac/linux × amd64/arm64/armv6/riscv64.

> Data at the edge · brains distributed · learning in a closed loop · humans decide.

---

_Companion adapters (bring an existing tool to a DureClaw fleet):
[picoclaw](https://github.com/DureClaw/picoclaw) ·
[nanobot](https://github.com/DureClaw/nanobot) ·
[zeroclaw](https://github.com/DureClaw/zeroclaw) ·
[nullclaw](https://github.com/DureClaw/nullclaw) — each ships a `dureclaw/` bridge.
edgeclaw is the **native** node, designed bus-first._
