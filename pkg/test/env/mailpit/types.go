package mailpit

import "time"

type ListMessagesResponse struct {
	Messages []MessageSummary `json:"Messages"`
}

type Account struct {
	Address string `json:"Address"`
	Name    string `json:"Name"`
}

type Message struct {
	Attachments []Attachment `json:"Attachments"`
	Bcc         []Account    `json:"Bcc"`
	Cc          []Account    `json:"Cc"`
	Date        string       `json:"Date"`
	From        Account      `json:"From"`
	HTML        string       `json:"HTML"`
	ID          string       `json:"ID"`
	Inline      []Attachment `json:"Inline"`
	MessageID   string       `json:"MessageID"`
	ReplyTo     []Account    `json:"ReplyTo"`
	ReturnPath  string       `json:"ReturnPath"`
	Size        int          `json:"Size"`
	Subject     string       `json:"Subject"`
	Tags        []string     `json:"Tags"`
	Text        string       `json:"Text"`
	To          []Account    `json:"To"`
}

type Attachment struct {
	ContentID   string `json:"ContentID"`
	ContentType string `json:"ContentType"`
	FileName    string `json:"FileName"`
	PartID      string `json:"PartID"`
	Size        int    `json:"Size"`
}

type MessageSummary struct {
	Attachments int       `json:"Attachments"`
	Bcc         []Account `json:"Bcc"`
	Cc          []Account `json:"Cc"`
	Created     time.Time `json:"Created"`
	From        Account   `json:"From"`
	ID          string    `json:"ID"`
	MessageID   string    `json:"MessageID"`
	Read        bool      `json:"Read"`
	ReplyTo     []Account `json:"ReplyTo"`
	Size        int       `json:"Size"`
	Snippet     string    `json:"Snippet"`
	Subject     string    `json:"Subject"`
	Tags        []string  `json:"Tags"`
	To          []Account `json:"To"`
}
