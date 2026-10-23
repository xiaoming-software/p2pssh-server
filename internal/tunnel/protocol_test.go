package tunnel

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	in := Frame{Op: OpOpenAck, Channel: 7, Seq: 0, Payload: []byte("ok")}
	raw := Encode(in)
	out, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Op != in.Op || out.Channel != in.Channel || out.Seq != in.Seq || string(out.Payload) != "ok" {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
	if raw[3] != MagicByte {
		t.Fatalf("magic byte want %d got %d", MagicByte, raw[3])
	}
}

func TestShellOpenPayload(t *testing.T) {
	if !IsShellOpen([]byte("shell")) {
		t.Fatal("shell payload")
	}
	if IsShellOpen(nil) || IsShellOpen([]byte("tcp")) {
		t.Fatal("tcp/empty must not be shell")
	}
}
