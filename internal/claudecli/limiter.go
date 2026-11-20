package claudecli

import "github.com/forgen-team/punisher/internal/clicore"

// The lane limiter lives in internal/clicore (shared by every AI CLI client);
// these aliases keep the claudecli names.
type (
	Lane          = clicore.Lane
	LimiterConfig = clicore.LimiterConfig
	Limiter       = clicore.Limiter
	LaneStats     = clicore.LaneStats
)

const (
	LaneShort       = clicore.LaneShort
	LaneInteractive = clicore.LaneInteractive
	LaneBackground  = clicore.LaneBackground
)

var (
	// ErrQueueFull is returned when a lane's wait queue is full.
	ErrQueueFull = clicore.ErrQueueFull
	NewLimiter   = clicore.NewLimiter
	ParseLane    = clicore.ParseLane
)
