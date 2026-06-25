package main
import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)
type mpvPlayer struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	conn     net.Conn
	sockPath string
	playing  bool
	position float64
	duration float64
	onUpdate func()
	onEnded  func()
}
func newMpvPlayer(onUpdate, onEnded func()) *mpvPlayer {
	return &mpvPlayer{onUpdate: onUpdate, onEnded: onEnded}
}
func (p *mpvPlayer) Play(path string) error {
	p.Stop()
	sock := filepath.Join(os.TempDir(), fmt.Sprintf("plumx-mpv-%d-%d.sock", os.Getpid(), time.Now().UnixNano()))
	cmd := exec.Command("mpv", "--no-video", "--no-terminal", "--idle=yes",
		"--input-ipc-server="+sock, path)
	if err := cmd.Start(); err != nil {
		return err
	}
	var conn net.Conn
	var err error
	for i := 0; i < 50; i++ {
		conn, err = net.Dial("unix", sock)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		cmd.Process.Kill()
		os.Remove(sock)
		return fmt.Errorf("mpv ipc connect: %w", err)
	}
	p.mu.Lock()
	p.cmd = cmd
	p.conn = conn
	p.sockPath = sock
	p.playing = true
	p.position = 0
	p.duration = 0
	p.mu.Unlock()
	go p.readLoop(conn)
	p.send([]interface{}{"observe_property", 1, "time-pos"})
	p.send([]interface{}{"observe_property", 2, "duration"})
	p.send([]interface{}{"observe_property", 3, "pause"})
	return nil
}
func (p *mpvPlayer) TogglePause() {
	p.mu.Lock()
	pause := p.playing
	p.mu.Unlock()
	p.send([]interface{}{"set_property", "pause", pause})
}
func (p *mpvPlayer) SeekTo(seconds float64) {
	p.send([]interface{}{"seek", seconds, "absolute"})
}
func (p *mpvPlayer) Stop() {
	p.mu.Lock()
	cmd := p.cmd
	conn := p.conn
	sock := p.sockPath
	p.cmd = nil
	p.conn = nil
	p.sockPath = ""
	p.playing = false
	p.position = 0
	p.duration = 0
	p.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}
	if sock != "" {
		os.Remove(sock)
	}
}
func (p *mpvPlayer) State() (playing bool, position, duration float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing, p.position, p.duration
}
func (p *mpvPlayer) send(cmd []interface{}) {
	p.mu.Lock()
	conn := p.conn
	p.mu.Unlock()
	if conn == nil {
		return
	}
	payload, err := json.Marshal(map[string]interface{}{"command": cmd})
	if err != nil {
		return
	}
	conn.Write(append(payload, '\n'))
}
func (p *mpvPlayer) readLoop(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var msg map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		event, _ := msg["event"].(string)
		switch event {
		case "property-change":
			name, _ := msg["name"].(string)
			p.mu.Lock()
			switch name {
			case "time-pos":
				if v, ok := msg["data"].(float64); ok {
					p.position = v
				}
			case "duration":
				if v, ok := msg["data"].(float64); ok {
					p.duration = v
				}
			case "pause":
				if v, ok := msg["data"].(bool); ok {
					p.playing = !v
				}
			}
			p.mu.Unlock()
			if p.onUpdate != nil {
				p.onUpdate()
			}
		case "end-file":
			p.mu.Lock()
			p.playing = false
			p.mu.Unlock()
			if p.onEnded != nil {
				p.onEnded()
			}
		}
	}
}
