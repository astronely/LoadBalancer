package rateLimiter

type Info struct {
	ID         string
	Capacity   float64
	RefillRate float64
	Interval   float64
}

type InfoFromBody struct {
	ID         string  `json:"id"`
	Capacity   float64 `json:"capacity"`
	RefillRate float64 `json:"refill_rate"`
	Interval   float64 `json:"interval"`
}
