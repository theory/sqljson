package types

import (
	"context"
	"time"
)

// offsetLocationFor returns an offset-only time.Location with the offset of
// the location of t.
func offsetLocationFor(t time.Time) *time.Location {
	if name, off := t.Zone(); name != "" {
		return time.FixedZone("", off)
	}
	return t.Location()
}

// offsetOnlyTimeFor returns t if its time zone is offset-only or a new
// offset-only time.Time with the offset of t's zone.
func offsetOnlyTimeFor(t time.Time) time.Time {
	if name, off := t.Zone(); name != "" {
		return t.In(time.FixedZone("", off))
	}
	return t
}

// key is an unexported type for keys defined in this package. This prevents
// collisions with keys defined in other packages.
type key int

//nolint:gochecknoglobals
var (
	// offsetZero represents time zone offset zero.
	offsetZero = time.FixedZone("", 0)

	// tzKey is the key for time.Location values in Contexts. It is unexported;
	// clients use ContextWithTZ and TZFromContext instead of using this key
	// directly.
	tzKey key
)

// ContextWithTZ returns a new Context that carries value tz.
func ContextWithTZ(ctx context.Context, tz *time.Location) context.Context {
	if tz == nil {
		return ctx
	}
	return context.WithValue(ctx, tzKey, tz)
}

// TZFromContext returns the time.Location value stored in ctx or time.UTC.
func TZFromContext(ctx context.Context) *time.Location {
	tz, ok := ctx.Value(tzKey).(*time.Location)
	if ok {
		return tz
	}
	return time.UTC
}
