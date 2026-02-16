package discord

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// Discord IPC opcodes.
const (
	opHandshake = 0
	opFrame     = 1
	opClose     = 2
)

// Activity types sent via Discord Rich Presence.
type Activity struct {
	Type       int          `json:"type,omitempty"`
	Name       string       `json:"name,omitempty"`
	Details    string       `json:"details,omitempty"`
	State      string       `json:"state,omitempty"`
	Timestamps *Timestamps  `json:"timestamps,omitempty"`
	Assets     *Assets      `json:"assets,omitempty"`
	Instance   bool         `json:"instance"`
}

type Timestamps struct {
	Start *int64 `json:"start,omitempty"`
	End   *int64 `json:"end,omitempty"`
}

type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

type ipcClient struct {
	conn net.Conn
}

func ipcConnect(appID string) (*ipcClient, error) {
	conn, err := dialSocket()
	if err != nil {
		return nil, fmt.Errorf("dial discord socket: %w", err)
	}
	c := &ipcClient{conn: conn}

	handshake, _ := json.Marshal(map[string]any{
		"v":         1,
		"client_id": appID,
	})
	if err := c.writeFrame(opHandshake, handshake); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake write: %w", err)
	}

	// Read handshake response.
	if _, _, err := c.readFrame(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake read: %w", err)
	}
	return c, nil
}

func dialSocket() (net.Conn, error) {
	base := os.TempDir()
	var lastErr error
	for i := 0; i <= 9; i++ {
		path := fmt.Sprintf("%s/discord-ipc-%d", base, i)
		conn, err := net.DialTimeout("unix", path, 5*time.Second)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("no discord socket found: %w", lastErr)
}

func (c *ipcClient) SetActivity(a Activity) error {
	payload, _ := json.Marshal(map[string]any{
		"cmd": "SET_ACTIVITY",
		"args": map[string]any{
			"pid":      os.Getpid(),
			"activity": a,
		},
		"nonce": nonce(),
	})
	if err := c.writeFrame(opFrame, payload); err != nil {
		return err
	}

	_, data, err := c.readFrame()
	if err != nil {
		return err
	}

	var resp struct {
		Evt  string `json:"evt"`
		Data struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if resp.Evt == "ERROR" {
		return fmt.Errorf("discord error %d: %s", resp.Data.Code, resp.Data.Message)
	}
	return nil
}

func (c *ipcClient) Close() {
	c.writeFrame(opClose, []byte("{}"))
	c.conn.Close()
}

// writeFrame sends a Discord IPC frame: [opcode LE u32][length LE u32][payload].
func (c *ipcClient) writeFrame(opcode uint32, payload []byte) error {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header[0:4], opcode)
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))
	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(payload)
	return err
}

// readFrame reads a Discord IPC frame, allocating a buffer of the exact
// size declared in the header. This avoids the fixed-buffer truncation
// bug in the old library.
func (c *ipcClient) readFrame() (uint32, []byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		return 0, nil, err
	}
	opcode := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.conn, payload); err != nil {
		return 0, nil, err
	}
	return opcode, payload, nil
}

func nonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
