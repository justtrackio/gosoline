package exec

import (
	"fmt"
	"net/http"
	"time"
)

func NewTestHttpClient(timeout time.Duration, trips Trips) http.Client {
	return http.Client{
		Timeout:   timeout,
		Transport: NewTestRoundTripper(trips...),
	}
}

type Trips []Trip

type Trip struct {
	duration time.Duration
	err      error
}

func DoTrip(duration time.Duration, err error) Trip {
	return Trip{
		duration: duration,
		err:      err,
	}
}

type TestRoundTripper struct {
	trips   []Trip
	current int
}

func NewTestRoundTripper(trips ...Trip) *TestRoundTripper {
	return &TestRoundTripper{
		trips:   trips,
		current: 0,
	}
}

func (t *TestRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	defer func() {
		t.current++
	}()

	if t.current >= len(t.trips) {
		return nil, fmt.Errorf("out of trips")
	}

	trip := t.trips[t.current]
	time.Sleep(trip.duration)

	if trip.err != nil {
		return nil, trip.err
	}

	return &http.Response{}, nil
}
