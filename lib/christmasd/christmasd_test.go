package christmasd

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/gobwas/ws/wsutil"
	"github.com/google/go-cmp/cmp"
	"github.com/neilotoole/slogt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"libdb.so/acm-christmas/lib/christmas/go/christmaspb"
)

func TestSession(t *testing.T) {
	conn := startTestSession(t, Config{Secret: "test"})

	writeClientMessage(t, conn, &christmaspb.LEDClientMessage{
		Message: &christmaspb.LEDClientMessage_Authenticate{
			Authenticate: &christmaspb.AuthenticateRequest{
				Secret: "bruh moment",
			},
		},
	})
	assertEq(t,
		&christmaspb.LEDServerMessage{Error: proto.String("invalid secret")},
		readServerMessage(t, conn))
	expectCloseFrame(t, conn)
}

func writeClientMessage(t *testing.T, conn combinedPipe, msg *christmaspb.LEDClientMessage) {
	t.Helper()

	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal("invalid client proto message:", err)
	}
	if err := wsutil.WriteClientBinary(conn, b); err != nil {
		t.Fatal("error writing client message:", err)
	}
}

func readServerMessage(t *testing.T, conn combinedPipe) *christmaspb.LEDServerMessage {
	t.Helper()

	b, err := wsutil.ReadServerBinary(conn)
	if err != nil {
		t.Fatal("error reading server message:", err)
	}

	msg := &christmaspb.LEDServerMessage{}
	if err := proto.Unmarshal(b, msg); err != nil {
		t.Fatal("invalid server proto message:", err)
	}

	return msg
}

func assertEq[T any](t *testing.T, expected, actual T, opts ...cmp.Option) {
	t.Helper()

	opts = append(opts, protocmp.Transform())
	if diff := cmp.Diff(expected, actual, opts...); diff != "" {
		t.Errorf("unexpected diff (-want +got):\n%s", diff)
	}
}

func expectCloseFrame(t *testing.T, conn combinedPipe) {
	t.Helper()
	var closedErr wsutil.ClosedError

	_, op, err := wsutil.ReadServerData(conn)
	if err == nil {
		t.Fatal("no close frame received, got op", op)
	}
	if !errors.As(err, &closedErr) {
		t.Fatal("unexpected non-ClosedError while reading server data:", err)
	}

	// Responding close frame is automatically handled by gobwas/ws/wsutil.
	// See wsutil/handler.go @ ControlHandler.HandleClose.
}

func startTestSession(t *testing.T, cfg Config) combinedPipe {
	t.Helper()

	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := combinedPipe{r1, w2}
	conn2 := combinedPipe{r2, w1}

	t.Cleanup(func() {
		t.Log("closing test session pipes")
		conn1.wp.Close()
		conn2.wp.Close()
	})

	logger := slogt.New(t)

	session := &Session{
		ws:     newWebsocketServer(conn1, logger),
		logger: logger,
		cfg:    cfg,
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	t.Cleanup(func() {
		cancel()
		if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) {
			t.Error("server session error:", err)
		}
	})

	go func() {
		errCh <- session.Start(ctx)
	}()

	return conn2
}

type combinedPipe struct {
	rp *io.PipeReader
	wp *io.PipeWriter
}

func (p combinedPipe) Read(b []byte) (int, error) {
	return p.rp.Read(b)
}

func (p combinedPipe) Write(b []byte) (int, error) {
	return p.wp.Write(b)
}

func (p combinedPipe) Close() error {
	p.wp.Close()
	return p.rp.Close()
}
