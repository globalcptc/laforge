package agentproto

import (
	"bytes"
	"encoding/binary"
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
