package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ffx64/discord-rpc-go/internal/codec"
	"github.com/ffx64/discord-rpc-go/transport/ipc"
	"github.com/google/uuid"
)

type Client struct {
	AppID     string
	transport *ipc.Conn
	mu        sync.Mutex

	verbose bool

	// event callbacks (unexported fields)
	onReady               func(map[string]any)
	onError               func(error)
	onClose               func()
	onActivityJoin        func(string)
	onActivitySpectate    func(string)
	onActivityJoinRequest func(map[string]any)

	pendingActivity *Activity
	ready           bool
	activity        Activity
	reconnect       bool
	closed          bool
}

func NewClient(appID string) *Client {
	return &Client{AppID: appID, reconnect: true}
}

func (c *Client) SetVerbose(v bool) {
	c.verbose = v
}

func (c *Client) logf(format string, a ...any) {
	if c.verbose {
		log.Printf(format, a...)
	}
}

func (c *Client) OnReady(fn func(info map[string]any))                  { c.onReady = fn }
func (c *Client) OnError(fn func(err error))                            { c.onError = fn }
func (c *Client) OnClose(fn func())                                     { c.onClose = fn }
func (c *Client) OnActivityJoin(fn func(secret string))                 { c.onActivityJoin = fn }
func (c *Client) OnActivitySpectate(fn func(secret string))             { c.onActivitySpectate = fn }
func (c *Client) OnActivityJoinRequest(fn func(payload map[string]any)) { c.onActivityJoinRequest = fn }

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.transport != nil {
		return errors.New("already connected")
	}

	conn, err := ipc.NewConn(c.AppID)
	if err != nil {
		return fmt.Errorf("dial ipc: %w", err)
	}
	c.transport = conn
	c.closed = false
	c.logf("[debug] connected to discord ipc")

	hs := map[string]any{"v": 1, "client_id": c.AppID}
	if err := c.transport.SendOp(ipc.OpHandshake, hs); err != nil {
		_ = c.transport.Close()
		c.transport = nil
		return fmt.Errorf("handshake send: %w", err)
	}

	go c.readLoop()

	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reconnect = false
	c.closed = true
	if c.transport != nil {
		err := c.transport.Close()
		c.transport = nil
		c.logf("[debug] connection closed")
		return err
	}
	return nil
}

func (c *Client) Login() error {
	return c.Connect()
}

func (c *Client) Logout() error {
	return c.Close()
}

func (c *Client) SetActivity(act Activity) error {
	c.mu.Lock()
	c.activity = act
	if !c.ready {
		c.pendingActivity = &act
		c.logf("[info] activity queued until READY")
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	return c.sendSetActivity(act)
}

func (c *Client) sendSetActivity(act Activity) error {
	if c.transport == nil {
		return errors.New("not connected")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if act.Party != nil && len(act.Party.Size) == 2 && act.Party.ID == "" {
		act.Party.ID = uuid.NewString()
	}

	validButtons := []Button{}
	for _, b := range act.Buttons {
		label := strings.TrimSpace(b.Label)
		url := strings.TrimSpace(b.Url)
		if label == "" || url == "" || !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
			continue
		}
		validButtons = append(validButtons, Button{Label: label, Url: url})
		if len(validButtons) == 2 {
			break
		}
	}
	act.Buttons = validButtons

	payloadAct := map[string]any{}

	payloadAct["type"] = int(act.Type) // Playing=0, Streaming=1, etc

	if act.State != "" {
		payloadAct["state"] = act.State
	}
	if act.Details != "" {
		payloadAct["details"] = act.Details
	}
	if act.Timestamps != nil && (act.Timestamps.Start != 0 || act.Timestamps.End != 0) {
		payloadAct["timestamps"] = act.Timestamps
	}
	if act.Party != nil && len(act.Party.Size) == 2 {
		payloadAct["party"] = act.Party
	}
	if len(act.Secrets) > 0 {
		payloadAct["secrets"] = act.Secrets
	}
	if len(act.Buttons) > 0 {
		payloadAct["buttons"] = act.Buttons
	}

	if act.Assets != nil {
		assets := map[string]any{}
		if act.Assets.LargeImage != "" {
			assets["large_image"] = act.Assets.LargeImage
			if act.Assets.LargeText == "" {
				assets["large_text"] = act.State
			} else {
				assets["large_text"] = act.Assets.LargeText
			}
		}
		if act.Assets.SmallImage != "" {
			assets["small_image"] = act.Assets.SmallImage
			if act.Assets.SmallText == "" {
				assets["small_text"] = act.State
			} else {
				assets["small_text"] = act.Assets.SmallText
			}
		}
		payloadAct["assets"] = assets
	}

	args := map[string]any{
		"pid":      os.Getpid(),
		"activity": payloadAct,
	}

	payload := map[string]any{
		"cmd":   "SET_ACTIVITY",
		"args":  args,
		"nonce": uuid.NewString(),
	}

	if c.verbose {
		if b, err := json.MarshalIndent(payload, "", "  "); err == nil {
			c.logf("[debug] outgoing SET_ACTIVITY payload:\n%s", string(b))
		}
	}

	return c.transport.SendOp(ipc.OpFrame, payload)
}

func (c *Client) readLoop() {
	for {
		if c.transport == nil {
			c.logf("[debug] transport nil, exiting readLoop")
			return
		}
		op, raw, err := c.transport.Receive()
		if err != nil {
			if c.onError != nil {
				c.onError(err)
			}
			c.logf("[warn] transport receive error: %v", err)
			if c.reconnect && !c.closed {
				c.tryReconnect()
			}
			return
		}

		var doc map[string]any
		if err := codec.Unmarshal(raw, &doc); err != nil {
			c.logf("[warn] failed decode payload: %v", err)
			continue
		}

		if c.verbose {
			if b, err := json.MarshalIndent(doc, "", "  "); err == nil {
				c.logf("[debug] incoming frame payload:\n%s", string(b))
			}
		}

		c.handleIncoming(op, doc)
	}
}

func (c *Client) handleIncoming(op ipc.OpCode, doc map[string]any) {
	if evt, ok := doc["evt"].(string); ok && evt == "READY" {
		c.logf("[debug] event READY")
		c.mu.Lock()
		c.ready = true
		var d map[string]any
		if tmp, ok := doc["data"].(map[string]any); ok {
			d = tmp
		}
		c.mu.Unlock()
		if c.onReady != nil {
			c.onReady(d)
		}
		c.mu.Lock()
		pa := c.pendingActivity
		c.pendingActivity = nil
		c.mu.Unlock()
		if pa != nil {
			if err := c.sendSetActivity(*pa); err != nil {
				c.logf("[error] failed sending pending activity: %v", err)
			} else {
				c.logf("[info] sent pending activity after READY")
			}
		}
	}
}

func codecToString(m map[string]any) string {
	for _, v := range m {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c *Client) tryReconnect() {
	backoff := time.Second
	for i := 0; i < 5; i++ {
		c.logf("[debug] reconnect attempt %d", i+1)
		time.Sleep(backoff)
		if err := c.Connect(); err == nil {
			c.logf("[info] reconnected successfully")
			if !c.activity.IsEmpty() {
				_ = c.SetActivity(c.activity)
			}
			return
		}
		backoff *= 2
	}
	c.logf("[error] failed to reconnect after 5 attempts")
}
