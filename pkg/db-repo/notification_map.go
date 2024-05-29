package db_repo

var NotificationTypes = []string{Create, Update, Delete}

type (
	NotificationMap map[string][]Notifier
)

func (m NotificationMap) AddNotifierAll(c Notifier) {
	for _, t := range NotificationTypes {
		m.AddNotifier(t, c)
	}
}

func (m NotificationMap) AddNotifier(t string, c Notifier) {
	if _, ok := m[t]; !ok {
		m[t] = make([]Notifier, 0)
	}

	m[t] = append(m[t], c)
}
