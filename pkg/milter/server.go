package milter

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/d--j/go-milter"
	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/filter"
)

// Server represents the ZPO milter server
type Server struct {
	config     *config.Config
	spamFilter *filter.SpamFilter
	milterSrv  *milter.Server
}

// NewServer creates a new milter server with the given configuration
func NewServer(cfg *config.Config) (*Server, error) {
	if !cfg.Milter.Enabled {
		return nil, fmt.Errorf("milter is not enabled in configuration")
	}

	// Create spam filter instance
	spamFilter := filter.NewSpamFilterWithConfig(cfg)

	// Configure milter options based on config
	var milterOpts []milter.Option

	// Configure protocol options (what events to skip)
	var skipProtocols milter.OptProtocol
	if cfg.Milter.SkipConnect {
		skipProtocols |= milter.OptNoConnect
	}
	if cfg.Milter.SkipHelo {
		skipProtocols |= milter.OptNoHelo
	}
	if cfg.Milter.SkipMail {
		skipProtocols |= milter.OptNoMailFrom
	}
	if cfg.Milter.SkipRcpt {
		skipProtocols |= milter.OptNoRcptTo
	}
	if cfg.Milter.SkipHeaders {
		skipProtocols |= milter.OptNoHeaders
	}
	if cfg.Milter.SkipBody {
		skipProtocols |= milter.OptNoBody
	}
	if cfg.Milter.SkipEOH {
		skipProtocols |= milter.OptNoEOH
	}
	if cfg.Milter.SkipData {
		skipProtocols |= milter.OptNoData
	}

	if skipProtocols != 0 {
		milterOpts = append(milterOpts, milter.WithProtocol(skipProtocols))
	}

	// Configure action capabilities
	var actions milter.OptAction
	if cfg.Milter.CanAddHeaders {
		actions |= milter.OptAddHeader
	}
	if cfg.Milter.CanChangeHeaders {
		actions |= milter.OptChangeHeader
	}
	if cfg.Milter.CanAddRecipients {
		actions |= milter.OptAddRcpt | milter.OptAddRcptWithArgs
	}
	if cfg.Milter.CanRemoveRecipients {
		actions |= milter.OptRemoveRcpt
	}
	if cfg.Milter.CanChangeBody {
		actions |= milter.OptChangeBody
	}
	if cfg.Milter.CanQuarantine {
		actions |= milter.OptQuarantine
	}
	if cfg.Milter.CanChangeFrom {
		actions |= milter.OptChangeFrom
	}

	if actions != 0 {
		milterOpts = append(milterOpts, milter.WithAction(actions))
	}

	// Configure timeouts
	if cfg.Milter.ReadTimeoutMs > 0 {
		milterOpts = append(milterOpts, milter.WithReadTimeout(
			time.Duration(cfg.Milter.ReadTimeoutMs)*time.Millisecond))
	}
	if cfg.Milter.WriteTimeoutMs > 0 {
		milterOpts = append(milterOpts, milter.WithWriteTimeout(
			time.Duration(cfg.Milter.WriteTimeoutMs)*time.Millisecond))
	}

	// Configure milter factory function
	milterOpts = append(milterOpts, milter.WithMilter(func() milter.Milter {
		return NewHandler(cfg, spamFilter)
	}))

	// Create milter server
	milterSrv := milter.NewServer(milterOpts...)

	return &Server{
		config:     cfg,
		spamFilter: spamFilter,
		milterSrv:  milterSrv,
	}, nil
}

// Serve starts the milter server and listens for connections
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	// Start the milter server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.milterSrv.Serve(listener)
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(s.config.Milter.GracefulShutdownTimeout)*time.Millisecond,
		)
		defer cancel()

		// Shutdown the milter server
		if err := s.milterSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shutdown milter server: %v", err)
		}

		return ctx.Err()

	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("milter server error: %v", err)
		}
		return nil
	}
}

// Close closes the milter server
func (s *Server) Close() error {
	return s.milterSrv.Close()
}

// Stats returns server statistics
func (s *Server) Stats() ServerStats {
	return ServerStats{
		MilterCount: s.milterSrv.MilterCount(),
	}
}

// ServerStats contains server statistics
type ServerStats struct {
	MilterCount uint64 // Total number of milter instances created
} 