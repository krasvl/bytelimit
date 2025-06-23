package bytelimit

// Unit is a unit of measurement for the rate limiter.
type Unit int64

const (
	KiB Unit = 1024
	MiB Unit = 1024 * KiB
	GiB Unit = 1024 * MiB
)
