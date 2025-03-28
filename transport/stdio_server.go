package transport

import (
	"bufio"
	"context"
	"io"
	"os"

	"go-mcp/pkg"
)

const stdioSessionID = "stdio"

type stdioServerTransport struct {
	receiver ServerReceiver
	reader   io.Reader
	writer   io.Writer

	logger pkg.Logger

	cancel          context.CancelFunc
	receiveShutDone chan struct{}
}

func NewStdioServerTransport() ServerTransport {
	return &stdioServerTransport{
		reader: os.Stdin,
		writer: os.Stdout,
		logger: pkg.DefaultLogger,

		receiveShutDone: make(chan struct{}),
	}
}

func (t *stdioServerTransport) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	go func() {
		defer pkg.Recover()

		t.receive(ctx)
		close(t.receiveShutDone)
	}()
	return nil
}

func (t *stdioServerTransport) Send(ctx context.Context, sessionID string, msg Message) error {
	_, err := t.writer.Write(append(msg, mcpMessageDelimiter))
	return err
}

func (t *stdioServerTransport) SetReceiver(receiver ServerReceiver) {
	t.receiver = receiver
}

func (t *stdioServerTransport) Shutdown(userCtx context.Context, serverCtx context.Context) error {
	t.cancel()

	<-t.receiveShutDone

	select {
	case <-serverCtx.Done():
		return nil
	case <-userCtx.Done():
		return userCtx.Err()
	}
}

func (t *stdioServerTransport) receive(ctx context.Context) {
	s := bufio.NewScanner(t.reader)

	for s.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			if err := t.receiver.Receive(ctx, stdioSessionID, s.Bytes()); err != nil {
				t.logger.Errorf("receiver failed: %v", err)
				return
			}
		}
	}

	if err := s.Err(); err != nil {
		t.logger.Errorf("server server unexpected error reading input: %v", err)
		return
	}
}
