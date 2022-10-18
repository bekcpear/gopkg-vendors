package utp

import (
	"math"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"
)

// All this rubbish came from missinggo but it's easier just to have them
// here because then they stand a chance of compiling to WASM, unlike
// various things in missinggo itself.

func WaitEvents(l sync.Locker, evs ...*Event) {
	cases := make([]reflect.SelectCase, 0, len(evs))
	for _, ev := range evs {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ev.C()),
		})
	}
	l.Unlock()
	reflect.Select(cases)
	l.Lock()
}

// Monotonic time represents time since an arbitrary point in the past, where
// the concept of now is only ever moving in a positive direction.
type MonotonicTime struct {
	skewedStdTime time.Time
}

func (me MonotonicTime) Sub(other MonotonicTime) time.Duration {
	return me.skewedStdTime.Sub(other.skewedStdTime)
}

var (
	stdNowFunc    = time.Now
	monotonicMu   sync.Mutex
	lastStdNow    time.Time
	monotonicSkew time.Duration
)

func skewedStdNow() time.Time {
	monotonicMu.Lock()
	defer monotonicMu.Unlock()
	stdNow := stdNowFunc()
	if !lastStdNow.IsZero() && stdNow.Before(lastStdNow) {
		monotonicSkew += lastStdNow.Sub(stdNow)
	}
	lastStdNow = stdNow
	return stdNow.Add(monotonicSkew)
}

// Consecutive calls always produce the same or greater time than previous
// calls.
func MonotonicNow() MonotonicTime {
	return MonotonicTime{skewedStdNow()}
}

func MonotonicSince(since MonotonicTime) (ret time.Duration) {
	return skewedStdNow().Sub(since.skewedStdTime)
}

// Events are boolean flags that provide a channel that's closed when true.
// This could go in the sync package, but that's more of a debug wrapper on
// the standard library sync.
type Event struct {
	ch     chan struct{}
	closed bool
}

func (me *Event) LockedChan(lock sync.Locker) <-chan struct{} {
	lock.Lock()
	ch := me.C()
	lock.Unlock()
	return ch
}

// Returns a chan that is closed when the event is true.
func (me *Event) C() <-chan struct{} {
	if me.ch == nil {
		me.ch = make(chan struct{})
	}
	return me.ch
}

// TODO: Merge into Set.
func (me *Event) Clear() {
	if me.closed {
		me.ch = nil
		me.closed = false
	}
}

// Set the event to true/on.
func (me *Event) Set() (first bool) {
	if me.closed {
		return false
	}
	if me.ch == nil {
		me.ch = make(chan struct{})
	}
	close(me.ch)
	me.closed = true
	return true
}

// TODO: Change to Get.
func (me *Event) IsSet() bool {
	return me.closed
}

func (me *Event) Wait() {
	<-me.C()
}

// TODO: Merge into Set.
func (me *Event) SetBool(b bool) {
	if b {
		me.Set()
	} else {
		me.Clear()
	}
}

// Extracts the port as an integer from an address string.
func AddrPort(addr net.Addr) int {
	switch raw := addr.(type) {
	case *net.UDPAddr:
		return raw.Port
	case *net.TCPAddr:
		return raw.Port
	default:
		_, port, err := net.SplitHostPort(addr.String())
		if err != nil {
			panic(err)
		}
		i64, err := strconv.ParseInt(port, 0, 0)
		if err != nil {
			panic(err)
		}
		return int(i64)
	}
}

// Returns random duration in the range [average-plusMinus,
// average+plusMinus]. Negative plusMinus will likely panic. Be aware that if
// plusMinus >= average, you may get a zero or negative Duration. The
// distribution function is unspecified, in case I find a more appropriate one
// in the future.
func JitterDuration(average, plusMinus time.Duration) (ret time.Duration) {
	ret = average - plusMinus
	ret += time.Duration(rand.Int63n(2*int64(plusMinus) + 1))
	return
}

// Returns a time.Timer that calls f. The timer is initially stopped.
func StoppedFuncTimer(f func()) (t *time.Timer) {
	t = time.AfterFunc(math.MaxInt64, f)
	if !t.Stop() {
		panic("timer already fired")
	}
	return
}
