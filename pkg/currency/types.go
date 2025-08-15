package currency

import "time"

type Rate struct {
	Currency string  `xml:"currency,attr"`
	Rate     float64 `xml:"rate,attr"`
}

type Content struct {
	Time  string `xml:"time,attr"`
	Rates []Rate `xml:"Cube"`
}

func (c Content) GetTime() (time.Time, error) {
	t, err := time.Parse(time.DateOnly, c.Time)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}
