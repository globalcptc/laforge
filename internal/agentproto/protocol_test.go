package agentproto

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, HeartbeatRequest, nil); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	payload := []byte(`{"hello":"world"}`)
	if err := WriteFrame(&buf, GetTaskResponse, payload); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	mt, body, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame (1st): %v", err)
	}
	if mt != HeartbeatRequest || len(body) != 0 {
		t.Fatalf("1st frame = (%v, %q), want (HeartbeatRequest, empty)", mt, body)
	}

	mt, body, err = ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame (2nd): %v", err)
	}
	if mt != GetTaskResponse || !bytes.Equal(body, payload) {
		t.Fatalf("2nd frame = (%v, %q), want (GetTaskResponse, %q)", mt, body, payload)
	}
}

// TestShellFramesRoundTrip covers the interactive-shell relay frames: a JSON
// attach, a RAW (non-JSON) data frame, a resize, and a close, all through the
// same WriteFrame/ReadFrame framing.
func TestShellFramesRoundTrip(t *testing.T) {
	var buf bytes.Buffer

	attach, _ := json.Marshal(ShellAttachPayload{SessionID: "s1", Role: ShellRoleClient, ObjectID: "obj1", Cols: 120, Rows: 40})
	if err := WriteFrame(&buf, ShellAttach, attach); err != nil {
		t.Fatalf("WriteFrame attach: %v", err)
	}
	raw := []byte{0x00, 0x1b, '[', 'A', 0xff} // raw bytes incl. non-UTF8 / control chars
	if err := WriteFrame(&buf, ShellData, raw); err != nil {
		t.Fatalf("WriteFrame data: %v", err)
	}
	resize, _ := json.Marshal(ShellResizePayload{Cols: 80, Rows: 24})
	if err := WriteFrame(&buf, ShellResize, resize); err != nil {
		t.Fatalf("WriteFrame resize: %v", err)
	}
	cls, _ := json.Marshal(ShellClosePayload{Reason: "exit"})
	if err := WriteFrame(&buf, ShellClose, cls); err != nil {
		t.Fatalf("WriteFrame close: %v", err)
	}

	mt, body, err := ReadFrame(&buf)
	if err != nil || mt != ShellAttach {
		t.Fatalf("attach frame = (%v, err=%v), want ShellAttach", mt, err)
	}
	var gotAttach ShellAttachPayload
	if err := json.Unmarshal(body, &gotAttach); err != nil || gotAttach.SessionID != "s1" || gotAttach.Role != ShellRoleClient || gotAttach.ObjectID != "obj1" || gotAttach.Cols != 120 || gotAttach.Rows != 40 {
		t.Fatalf("attach payload = %+v (err=%v), want {s1 client obj1 120 40}", gotAttach, err)
	}

	mt, body, err = ReadFrame(&buf)
	if err != nil || mt != ShellData || !bytes.Equal(body, raw) {
		t.Fatalf("data frame = (%v, %q, err=%v), want (ShellData, raw bytes preserved)", mt, body, err)
	}

	mt, body, err = ReadFrame(&buf)
	if err != nil || mt != ShellResize {
		t.Fatalf("resize frame = (%v, err=%v), want ShellResize", mt, err)
	}
	var gotResize ShellResizePayload
	if err := json.Unmarshal(body, &gotResize); err != nil || gotResize.Cols != 80 || gotResize.Rows != 24 {
		t.Fatalf("resize payload = %+v, want {80 24}", gotResize)
	}

	mt, _, err = ReadFrame(&buf)
	if err != nil || mt != ShellClose {
		t.Fatalf("close frame = (%v, err=%v), want ShellClose", mt, err)
	}
}

func TestReadFrameRejectsZeroLength(t *testing.T) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 0)
	_, _, err := ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatal("expected an error for a zero-length frame (no room for a message type byte)")
	}
}

func TestReadFrameRejectsOversized(t *testing.T) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, MaxFrameSize+1)
	_, _, err := ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatal("expected an error for a frame claiming to exceed MaxFrameSize")
	}
}

func TestReadFrameRejectsTruncatedStream(t *testing.T) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, 100) // claims 100 bytes follow
	buf = append(buf, []byte("short")...)
	_, _, err := ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatal("expected an error for a frame whose body is shorter than its own length prefix")
	}
	if err != io.ErrUnexpectedEOF && err != io.EOF {
		t.Logf("got error %v (any read error is acceptable, just not nil or a panic)", err)
	}
}

func TestWriteFrameRejectsOversizedPayload(t *testing.T) {
	huge := make([]byte, MaxFrameSize)
	var buf bytes.Buffer
	err := WriteFrame(&buf, HeartbeatRequest, huge)
	if err == nil {
		t.Fatal("expected an error writing a payload at MaxFrameSize (leaves no room for the type byte)")
	}
}

// FuzzReadFrame is the required fuzz test: ReadFrame is the one
// function in this whole system that parses bytes an attacker fully
// controls (anything reaching the gateway's TLS listener, before any
// application-level trust decision has been made) into structured data.
// The property under test is simply "never panics, on anything" -- a
// malformed length prefix, a truncated body, random noise. `go test
// -fuzz=FuzzReadFrame` runs this against a mutating corpus; the seed
// corpus below plus `go test -run FuzzReadFrame` (as `go test ./...`
// already does) replays every previously-found failing case as a normal
// test.
func FuzzReadFrame(f *testing.F) {
	var validFrame bytes.Buffer
	WriteFrame(&validFrame, GetTaskResponse, []byte(`{"task":null}`))
	f.Add(validFrame.Bytes())

	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{0, 0, 0, 1})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0, 0, 0, 5, 1, 2, 3})
	f.Add([]byte{0, 0, 0, 1, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ReadFrame panicked on input %x: %v", data, r)
			}
		}()
		ReadFrame(bytes.NewReader(data))
	})
}
