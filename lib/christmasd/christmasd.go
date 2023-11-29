package christmasd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/gobwas/ws"
	"golang.org/x/sync/errgroup"
	"gopkg.in/typ.v4/sync2"
)

// Config is the configuration for handling.
type Config struct {
	// Secret is the secret to use for the server.
	// The secret is used to authenticate the client.
	Secret string
}

// ServerOpts are options for a server.
type ServerOpts struct {
	// Logger is the logger to use for the server.
	Logger *slog.Logger
	// HTTPUpgrader is the HTTP-to-Websocket upgrader to use for the server.
	HTTPUpgrader ws.HTTPUpgrader
}

// Server handles all HTTP requests for the server.
type Server struct {
	opts        ServerOpts
	cfg         atomic.Pointer[Config]
	connections sync2.Map[*Session, sessionControl]
}

type sessionControl struct {
	cancel context.CancelCauseFunc
}

// NewServer creates a new server.
func NewServer(cfg Config, opts ServerOpts) *Server {
	s := &Server{
		opts: opts,
	}
	s.cfg.Store(&cfg)
	return s
}

// KickAllConnections kicks all connections from the server.
// Optionally, a reason can be provided.
func (s *Server) KickAllConnections(reason string) {
	var err error
	if reason != "" {
		err = fmt.Errorf("kicked: %s", reason)
	}

	s.connections.Range(func(s *Session, ctrl sessionControl) bool {
		ctrl.cancel(err)
		return true
	})
}

// SetConfig sets the configuration for the server. All future connections will
// use the new configuration. Existing connections will continue to use the old
// configuration, unless they are kicked out.
func (s *Server) SetConfig(cfg Config) {
	s.cfg.Store(&cfg)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, err := s.upgrade(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancelCause(r.Context())
	s.connections.Store(session, sessionControl{cancel: cancel})

	if err := session.Start(ctx); err != nil {
		s.connections.Delete(session)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) upgrade(w http.ResponseWriter, r *http.Request) (*Session, error) {
	wsconn, _, _, err := s.opts.HTTPUpgrader.Upgrade(r, w)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade HTTP: %w", err)
	}

	logger := s.opts.Logger.With(
		"local_addr", wsconn.LocalAddr(),
		"remote_addr", wsconn.RemoteAddr())

	return &Session{
		ws:     newWebsocketServer(wsconn, logger),
		logger: logger,
		cfg:    *s.cfg.Load(),
	}, nil
}

// Session is a websocket session. It implements handling of messages from a
// single client.
type Session struct {
	ws     *websocketServer
	logger *slog.Logger

	cfg Config
}

// Start starts the server.
func (s *Session) Start(ctx context.Context) error {
	errg, ctx := errgroup.WithContext(ctx)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errg.Go(func() error {
		return s.ws.Start(ctx)
	})

	errg.Go(func() error {
		// Treat main loop errors as fatal and kill the connection,
		// but don't return it because it's not the caller's fault.
		if err := s.mainLoop(ctx); err != nil {
			return s.ws.SendError(ctx, err)
		}
		return nil
	})

	return errg.Wait()
}

var (
	errNotAuthenticated = fmt.Errorf("not authenticated")
	errInvalidSecret    = fmt.Errorf("invalid secret")
)

func (s *Session) mainLoop(ctx context.Context) error {
	var authenticated bool

	for {
		select {
		case <-ctx.Done():
			return nil

		case msg := <-s.ws.Messages:
			// Assert that the client is authenticated.
			// Kick the client if not.
			if !authenticated {
				auth := msg.GetAuthenticate()
				if auth == nil {
					return errNotAuthenticated
				}
				if auth.Secret != s.cfg.Secret {
					return errInvalidSecret
				}
				authenticated = true

				s.logger.DebugContext(ctx,
					"new client authenticated")
			}

			// switch msg := msg.GetMessage().(type) {
			// case *christmaspb.LEDClientMessage_GetLedCanvasInfo:
			// case *christmaspb.LEDClientMessage_SetLedCanvas:
			// case *christmaspb.LEDClientMessage_GetLeds:
			// case *christmaspb.LEDClientMessage_SetLeds:
			// }
		}
	}
}
