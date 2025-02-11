// Copyright (c) 2015 Uber Technologies, Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package testutils

import (
	"flag"
	"net"
	"testing"
	"time"

	"github.com/temporalio/tchannel-go"
	"github.com/temporalio/tchannel-go/tos"

	"go.uber.org/atomic"
	"golang.org/x/net/context"
)

var connectionLog = flag.Bool("connectionLog", false, "Enables connection logging in tests")

// Default service names for the test channels.
const (
	DefaultServerName = "testService"
	DefaultClientName = "testService-client"
)

// ChannelOpts contains options to create a test channel using WithServer
type ChannelOpts struct {
	tchannel.ChannelOptions

	// ServiceName defaults to DefaultServerName or DefaultClientName.
	ServiceName string

	// LogVerification contains options for controlling the log verification.
	LogVerification LogVerification

	// DisableRelay disables the relay interposed between clients/servers.
	// By default, all tests are run with a relay interposed.
	DisableRelay bool

	// DisableServer disables creation of the TChannel server.
	// This is typically only used in relay tests when a custom server is required.
	DisableServer bool

	// OnlyRelay instructs TestServer the test must only be run with a relay.
	OnlyRelay bool

	// RunCount is the number of times the test should be run. Zero or
	// negative values are treated as a single run.
	RunCount int

	// postFns is a list of functions that are run after the test.
	// They are run even if the test fails.
	postFns []func()
}

// LogVerification contains options to control the log verification.
type LogVerification struct {
	Disabled bool

	Filters []LogFilter
}

// LogFilter is a single substring match that can be ignored.
type LogFilter struct {
	// Filter specifies the substring match to search
	// for in the log message to skip raising an error.
	Filter string

	// Count is the maximum number of allowed warn+ logs matching
	// Filter before errors are raised.
	Count uint

	// FieldFilters specifies expected substring matches for fields.
	FieldFilters map[string]string
}

// Copy copies the channel options (so that they can be safely modified).
func (o *ChannelOpts) Copy() *ChannelOpts {
	if o == nil {
		return NewOpts()
	}
	copiedOpts := *o
	return &copiedOpts
}

// SetServiceName sets ServiceName.
func (o *ChannelOpts) SetServiceName(svcName string) *ChannelOpts {
	o.ServiceName = svcName
	return o
}

// SetProcessName sets the ProcessName in ChannelOptions.
func (o *ChannelOpts) SetProcessName(processName string) *ChannelOpts {
	o.ProcessName = processName
	return o
}

// SetStatsReporter sets StatsReporter in ChannelOptions.
func (o *ChannelOpts) SetStatsReporter(statsReporter tchannel.StatsReporter) *ChannelOpts {
	o.StatsReporter = statsReporter
	return o
}

// SetFramePool sets FramePool in DefaultConnectionOptions.
func (o *ChannelOpts) SetFramePool(framePool tchannel.FramePool) *ChannelOpts {
	o.DefaultConnectionOptions.FramePool = framePool
	return o
}

// SetHealthChecks sets HealthChecks in DefaultConnectionOptions.
func (o *ChannelOpts) SetHealthChecks(healthChecks tchannel.HealthCheckOptions) *ChannelOpts {
	o.DefaultConnectionOptions.HealthChecks = healthChecks
	return o
}

// SetSendBufferSize sets the SendBufferSize in DefaultConnectionOptions.
func (o *ChannelOpts) SetSendBufferSize(bufSize int) *ChannelOpts {
	o.DefaultConnectionOptions.SendBufferSize = bufSize
	return o
}

// SetSendBufferSizeOverrides sets the SendBufferOverrides in DefaultConnectionOptions.
func (o *ChannelOpts) SetSendBufferSizeOverrides(overrides []tchannel.SendBufferSizeOverride) *ChannelOpts {
	o.DefaultConnectionOptions.SendBufferSizeOverrides = overrides
	return o
}

// SetTosPriority set TosPriority in DefaultConnectionOptions.
func (o *ChannelOpts) SetTosPriority(tosPriority tos.ToS) *ChannelOpts {
	o.DefaultConnectionOptions.TosPriority = tosPriority
	return o
}

// SetChecksumType sets the ChecksumType in DefaultConnectionOptions.
func (o *ChannelOpts) SetChecksumType(checksumType tchannel.ChecksumType) *ChannelOpts {
	o.DefaultConnectionOptions.ChecksumType = checksumType
	return o
}

// SetTimeNow sets TimeNow in ChannelOptions.
func (o *ChannelOpts) SetTimeNow(timeNow func() time.Time) *ChannelOpts {
	o.TimeNow = timeNow
	return o
}

// SetTimeTicker sets TimeTicker in ChannelOptions.
func (o *ChannelOpts) SetTimeTicker(timeTicker func(d time.Duration) *time.Ticker) *ChannelOpts {
	o.TimeTicker = timeTicker
	return o
}

// DisableLogVerification disables log verification for this channel.
func (o *ChannelOpts) DisableLogVerification() *ChannelOpts {
	o.LogVerification.Disabled = true
	return o
}

// NoRelay disables running the test with a relay interposed.
func (o *ChannelOpts) NoRelay() *ChannelOpts {
	o.DisableRelay = true
	return o
}

// SetRelayOnly instructs TestServer to only run with a relay in front of this channel.
func (o *ChannelOpts) SetRelayOnly() *ChannelOpts {
	o.OnlyRelay = true
	return o
}

// SetDisableServer disables creation of the TChannel server.
// This is typically only used in relay tests when a custom server is required.
func (o *ChannelOpts) SetDisableServer() *ChannelOpts {
	o.DisableServer = true
	return o
}

// SetRunCount sets the number of times run the test.
func (o *ChannelOpts) SetRunCount(n int) *ChannelOpts {
	o.RunCount = n
	return o
}

// AddLogFilter sets an allowed filter for warning/error logs and sets
// the maximum number of times that log can occur.
func (o *ChannelOpts) AddLogFilter(filter string, maxCount uint, fields ...string) *ChannelOpts {
	fieldFilters := make(map[string]string)
	for i := 0; i < len(fields); i += 2 {
		fieldFilters[fields[i]] = fields[i+1]
	}

	o.LogVerification.Filters = append(o.LogVerification.Filters, LogFilter{
		Filter:       filter,
		Count:        maxCount,
		FieldFilters: fieldFilters,
	})
	return o
}

func (o *ChannelOpts) addPostFn(f func()) {
	o.postFns = append(o.postFns, f)
}

// SetRelayHost sets the channel's RelayHost, which enables relaying.
func (o *ChannelOpts) SetRelayHost(rh tchannel.RelayHost) *ChannelOpts {
	o.ChannelOptions.RelayHost = rh
	return o
}

// SetRelayLocal sets the channel's relay local handlers for service names
// that should be handled by the relay channel itself.
func (o *ChannelOpts) SetRelayLocal(relayLocal ...string) *ChannelOpts {
	o.ChannelOptions.RelayLocalHandlers = relayLocal
	return o
}

// SetRelayMaxTimeout sets the maximum allowable timeout for relayed calls.
func (o *ChannelOpts) SetRelayMaxTimeout(d time.Duration) *ChannelOpts {
	o.ChannelOptions.RelayMaxTimeout = d
	return o
}

// SetRelayMaxConnectionTimeout sets the maximum timeout for connection attempts.
func (o *ChannelOpts) SetRelayMaxConnectionTimeout(d time.Duration) *ChannelOpts {
	o.ChannelOptions.RelayMaxConnectionTimeout = d
	return o
}

// SetRelayMaxTombs sets the maximum number of tombs tracked in the relayer.
func (o *ChannelOpts) SetRelayMaxTombs(maxTombs uint64) *ChannelOpts {
	o.ChannelOptions.RelayMaxTombs = maxTombs
	return o
}

// SetOnPeerStatusChanged sets the callback for channel status change
// noficiations.
func (o *ChannelOpts) SetOnPeerStatusChanged(f func(*tchannel.Peer)) *ChannelOpts {
	o.ChannelOptions.OnPeerStatusChanged = f
	return o
}

// SetMaxIdleTime sets a threshold after which idle connections will
// automatically get dropped. See idle_sweep.go for more details.
func (o *ChannelOpts) SetMaxIdleTime(d time.Duration) *ChannelOpts {
	o.ChannelOptions.MaxIdleTime = d
	return o
}

// SetIdleCheckInterval sets the frequency of the periodic poller that removes
// stale connections from the channel.
func (o *ChannelOpts) SetIdleCheckInterval(d time.Duration) *ChannelOpts {
	o.ChannelOptions.IdleCheckInterval = d
	return o
}

// SetDialer sets the dialer used for outbound connections
func (o *ChannelOpts) SetDialer(f func(context.Context, string, string) (net.Conn, error)) *ChannelOpts {
	o.ChannelOptions.Dialer = f
	return o
}

// SetConnContext sets the connection's ConnContext function
func (o *ChannelOpts) SetConnContext(f func(context.Context, net.Conn) context.Context) *ChannelOpts {
	o.ConnContext = f
	return o
}

func defaultString(v string, defaultValue string) string {
	if v == "" {
		return defaultValue
	}
	return v
}

// NewOpts returns a new ChannelOpts that can be used in a chained fashion.
func NewOpts() *ChannelOpts { return &ChannelOpts{} }

// DefaultOpts will return opts if opts is non-nil, NewOpts otherwise.
func DefaultOpts(opts *ChannelOpts) *ChannelOpts {
	if opts == nil {
		return NewOpts()
	}
	return opts
}

// WrapLogger wraps the given logger with extra verification.
func (v *LogVerification) WrapLogger(t testing.TB, l tchannel.Logger) tchannel.Logger {
	return errorLogger{l, t, v, &errorLoggerState{
		matchCount: make([]atomic.Uint32, len(v.Filters)),
	}}
}
