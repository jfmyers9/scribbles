package discord

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	c := &ipcClient{conn: client}

	// Write a frame from the client side.
	payload := `{"cmd":"SET_ACTIVITY","nonce":"abc123"}`
	go func() {
		if err := c.writeFrame(opFrame, []byte(payload)); err != nil {
			t.Errorf("writeFrame: %v", err)
		}
	}()

	// Read raw bytes from the server side and verify framing.
	header := make([]byte, 8)
	if _, err := io.ReadFull(server, header); err != nil {
		t.Fatalf("read header: %v", err)
	}
	opcode := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])

	if opcode != opFrame {
		t.Errorf("opcode = %d, want %d", opcode, opFrame)
	}
	if int(length) != len(payload) {
		t.Errorf("length = %d, want %d", length, len(payload))
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(server, body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != payload {
		t.Errorf("body = %q, want %q", body, payload)
	}
}

func TestReadFrameLargePayload(t *testing.T) {
	// Verify readFrame handles payloads larger than 512 bytes,
	// which was the bug in the old library.
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	c := &ipcClient{conn: server}

	// Build a payload >512 bytes.
	large := make([]byte, 2048)
	for i := range large {
		large[i] = 'x'
	}

	// Write raw frame from client side simulating Discord.
	go func() {
		header := make([]byte, 8)
		binary.LittleEndian.PutUint32(header[0:4], opFrame)
		binary.LittleEndian.PutUint32(header[4:8], uint32(len(large)))
		_, _ = client.Write(header)
		_, _ = client.Write(large)
	}()

	opcode, payload, err := c.readFrame()
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if opcode != opFrame {
		t.Errorf("opcode = %d, want %d", opcode, opFrame)
	}
	if len(payload) != len(large) {
		t.Errorf("payload length = %d, want %d", len(payload), len(large))
	}
}

func TestReadFrameHandshake(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	c := &ipcClient{conn: server}

	payload := `{"cmd":"DISPATCH","evt":"READY"}`
	go func() {
		header := make([]byte, 8)
		binary.LittleEndian.PutUint32(header[0:4], opHandshake)
		binary.LittleEndian.PutUint32(header[4:8], uint32(len(payload)))
		_, _ = client.Write(header)
		_, _ = client.Write([]byte(payload))
	}()

	opcode, data, err := c.readFrame()
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if opcode != opHandshake {
		t.Errorf("opcode = %d, want %d", opcode, opHandshake)
	}
	if string(data) != payload {
		t.Errorf("data = %q, want %q", data, payload)
	}
}
