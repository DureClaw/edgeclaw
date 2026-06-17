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
`windows/amd64`, `darwin/*` from one `make all`. A pure-Python stdlib bridge is
bundled (`bridge/`) as the ultra-portable fallback for boxes without the binary.

## What it does

- **Joins the bus** (Phoenix Channel, `vsn=2.0.0`) with presence + heartbeat.
- **Receives `task.assign`**, runs the work, returns `task.result`.
- **Local hands:** `[SHELL] <cmd>` runs on the device (sensor/GPIO/scripts).
- **LLM, keyless by default:** delegates inference to the **master brain**
  (`BRAIN_URL`/brain/exec) so the edge holds **no API key, no cost**.
  Or use a **local** provider (`OLLAMA_URL`), or wrap **any CLI** (`AGENT_CMD`).

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
