// edgeclaw — the DureClaw-native edge agent.
//
// One small, static Go binary that turns ANY machine (Windows · macOS · Linux ·
// Raspberry Pi · riscv64 · …) into a member of a DureClaw collaboration fleet.
//
//	"Local hands, remote brain": edgeclaw runs on-device work (shell / sensor)
//	locally, and delegates LLM inference to the master brain (keyless) — or to a
//	local provider (Ollama) — then returns results over the Phoenix-Channel bus.
//
// Build once, cross-compile everywhere (see Makefile). Pure-Go deps only.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// ── config (env) ────────────────────────────────────────────────────────────
var (
	stateServer = env("STATE_SERVER", "127.0.0.1:4000") // bus host:port
	secret      = env("OAH_SECRET", "")                 // bus bearer token
	workKey     = env("WORK_KEY", "WK-demo")            // collaboration session
	agentName   = env("AGENT_NAME", "edgeclaw@"+hostname())
	agentRole   = env("AGENT_ROLE", "executor")
	machine     = env("AGENT_MACHINE", hostname())
	brainURL    = env("BRAIN_URL", "")    // master brain /brain/exec (keyless LLM)
	brainToken  = env("BRAIN_TOKEN", "")  //
	ollamaURL   = env("OLLAMA_URL", "")   // local LLM fallback, e.g. http://127.0.0.1:11434
	ollamaModel = env("OLLAMA_MODEL", "") //
	agentCmd    = env("AGENT_CMD", "")    // wrap any external CLI; {} ← instruction
	caps        = splitCSV(env("CAPABILITIES", "edge,shell,agent"))
)

var refCounter int64
var joinRefGlobal atomic.Value // string — current connection's join_ref (for pushes on the work topic)

func nextRef() string { return fmt.Sprintf("%d", atomic.AddInt64(&refCounter, 1)) }

func joinRef() any {
	if v, ok := joinRefGlobal.Load().(string); ok {
		return v
	}
	return nil
}

func main() {
	log.SetFlags(log.Ltime)
	log.Printf("[edgeclaw] %s (role=%s, machine=%s) → bus %s work:%s", agentName, agentRole, machine, stateServer, workKey)
	for {
		if err := run(); err != nil {
			log.Printf("[edgeclaw] %v — reconnecting in 3s", err)
			time.Sleep(3 * time.Second)
		}
	}
}

func run() error {
	url := fmt.Sprintf("ws://%s/socket/websocket?vsn=2.0.0", stateServer)
	if secret != "" {
		url += "&token=" + secret
	}
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer c.Close()
	log.Printf("[edgeclaw] connected")

	jr := nextRef()
	joinRefGlobal.Store(jr)
	send(c, []any{jr, jr, "work:" + workKey, "phx_join", map[string]any{
		"agent_name": agentName, "role": agentRole, "machine": machine,
		"capabilities": caps, "preferred_model": backendLabel(), "version": "edgeclaw/0.1",
	}})
	log.Printf("[edgeclaw] joined work:%s as %s", workKey, agentName)

	// heartbeat
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				send(c, []any{nil, nextRef(), "phoenix", "heartbeat", map[string]any{}})
			}
		}
	}()

	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		if os.Getenv("DEBUG") != "" {
			log.Printf("[edgeclaw][rx] %.200s", string(raw))
		}
		var frame []json.RawMessage
		if json.Unmarshal(raw, &frame) != nil || len(frame) != 5 {
			continue
		}
		var event string
		json.Unmarshal(frame[3], &event)
		if event != "task.assign" {
			continue
		}
		var p taskPayload
		if json.Unmarshal(frame[4], &p) != nil {
			continue
		}
		if p.To != "" && p.To != agentName && p.To != "broadcast" {
			continue
		}
		go handle(c, p)
	}
}

type taskPayload struct {
	TaskID       string `json:"task_id"`
	To           string `json:"to"`
	From         string `json:"from"`
	Role         string `json:"role"`
	Instructions string `json:"instructions"`
}

func handle(c *websocket.Conn, p taskPayload) {
	instr := strings.TrimSpace(p.Instructions)
	if instr == "" {
		return
	}
	from := p.From
	if from == "" {
		from = "http@controller"
	}
	log.Printf("[edgeclaw] task %s: %.70s", p.TaskID, instr)
	out, code := runTask(instr)
	status := "done"
	if code != 0 {
		status = "blocked"
	}
	send(c, []any{joinRef(), nextRef(), "work:" + workKey, "task.result", map[string]any{
		"task_id": p.TaskID, "to": from, "from": agentName,
		"status": status, "output": clip(out, 1800), "exit_code": code, "backend": backendLabel(),
	}})
	log.Printf("[edgeclaw] result %s: %s (%d chars)", p.TaskID, status, len(out))
}

// ── task execution: local hands or LLM ──────────────────────────────────────
func runTask(instr string) (string, int) {
	switch {
	case strings.HasPrefix(strings.ToUpper(instr), "[SHELL]"):
		cmd := strings.TrimSpace(instr[len("[SHELL]"):])
		return runShell(cmd)
	case brainURL != "": // keyless: delegate LLM to the master brain
		out, err := brainExec(instr)
		return orErr(out, err)
	case ollamaURL != "": // local LLM
		out, err := ollamaGen(instr)
		return orErr(out, err)
	case agentCmd != "": // wrap any external CLI
		return runCmdTemplate(agentCmd, instr)
	default:
		return "[edgeclaw] no LLM configured (set BRAIN_URL / OLLAMA_URL / AGENT_CMD). echo: " + instr, 0
	}
}

func runShell(cmd string) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return string(out) + "\n" + err.Error(), 1
	}
	return string(out), 0
}

func runCmdTemplate(tmpl, instr string) (string, int) {
	fields := strings.Fields(tmpl)
	args := make([]string, 0, len(fields))
	replaced := false
	for _, f := range fields {
		if f == "{}" || f == "{prompt}" {
			args = append(args, instr)
			replaced = true
		} else {
			args = append(args, f)
		}
	}
	if !replaced {
		args = append(args, instr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return string(out) + "\n" + err.Error(), 1
	}
	return string(out), 0
}

func brainExec(prompt string) (string, error) {
	body, _ := json.Marshal(map[string]string{"prompt": prompt})
	req, _ := http.NewRequest("POST", strings.TrimRight(brainURL, "/")+"/brain/exec", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if brainToken != "" {
		req.Header.Set("Authorization", "Bearer "+brainToken)
	}
	return postJSON(req, "output")
}

func ollamaGen(prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{"model": ollamaModel, "prompt": prompt, "stream": false})
	req, _ := http.NewRequest("POST", strings.TrimRight(ollamaURL, "/")+"/api/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return postJSON(req, "response")
}

func postJSON(req *http.Request, field string) (string, error) {
	cl := &http.Client{Timeout: 180 * time.Second}
	resp, err := cl.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", err
	}
	if s, ok := m[field].(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", m), nil
}

// ── helpers ─────────────────────────────────────────────────────────────────
func send(c *websocket.Conn, v []any) {
	b, _ := json.Marshal(v)
	c.WriteMessage(websocket.TextMessage, b)
}

func backendLabel() string {
	switch {
	case brainURL != "":
		return "brain-remote"
	case ollamaURL != "":
		return "ollama:" + ollamaModel
	case agentCmd != "":
		return "cli"
	default:
		return "edge"
	}
}

func orErr(out string, err error) (string, int) {
	if err != nil {
		return err.Error(), 1
	}
	return out, 0
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func hostname() string { h, _ := os.Hostname(); return h }

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
