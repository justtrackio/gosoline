package currency

type Currency string

type Rate struct {
	Currency string  `xml:"currency,attr"`
	Rate     float64 `xml:"rate,attr"`
}

type Content struct {
	Time  string `xml:"time,attr"`
	Rates []Rate `xml:"Cube"`
}

type Body struct {
	Content Content `xml:"Cube"`
}

type Sender struct {
	Name string `xml:"name"`
}

type ExchangeResponse struct {
	Subject string `xml:"subject"`
	Sender  Sender `xml:"Sender"`
	Body    Body   `xml:"Cube"`
}
