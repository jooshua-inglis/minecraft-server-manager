package rcon

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
)

// fakeServer is a minimal Source RCON server good enough to test the
// client's wire format without needing a real Minecraft server.
func fakeServer(t *testing.T, password string) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			id, ptype, payload, err := readPacket(conn)
			if err != nil {
				return
			}
			switch ptype {
			case packetAuth:
				if payload == password {
					writePacket(conn, id, 2, "")
				} else {
					writePacket(conn, authFailedID, 2, "")
				}
			case packetExecCommand:
				writePacket(conn, id, 0, "ok: "+payload)
			}
		}
	}()

	return ln.Addr().String()
}

func readPacket(conn net.Conn) (id, ptype int32, payload string, err error) {
	var length int32
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return 0, 0, "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return 0, 0, "", err
	}
	id = int32(binary.LittleEndian.Uint32(buf[0:4]))
	ptype = int32(binary.LittleEndian.Uint32(buf[4:8]))
	payload = string(buf[8 : len(buf)-2])
	return id, ptype, payload, nil
}

func writePacket(conn net.Conn, id, ptype int32, payload string) error {
	body := new(bytes.Buffer)
	binary.Write(body, binary.LittleEndian, id)
	binary.Write(body, binary.LittleEndian, ptype)
	body.WriteString(payload)
	body.Write([]byte{0, 0})

	if err := binary.Write(conn, binary.LittleEndian, int32(body.Len())); err != nil {
		return err
	}
	_, err := conn.Write(body.Bytes())
	return err
}

func TestDialAndExecute(t *testing.T) {
	addr := fakeServer(t, "secret")

	client, err := Dial(addr, "secret")
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	got, err := client.Execute("say hello")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "ok: say hello"; got != want {
		t.Errorf("Execute = %q, want %q", got, want)
	}
}

func TestDialWrongPassword(t *testing.T) {
	addr := fakeServer(t, "secret")

	if _, err := Dial(addr, "wrong"); err != ErrAuthFailed {
		t.Fatalf("Dial(wrong password) = %v, want ErrAuthFailed", err)
	}
}
