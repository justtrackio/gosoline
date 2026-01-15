package cfg

// AppTags is a dynamic map of application tags.
// Common tags include "project", "family", and "group", but any key-value
// pairs can be defined for custom use cases like logging, metrics labels, etc.
//
// Example configuration:
//
//	app:
//	  env: production
//	  name: myapp
//	  tags:
//	    project: myproject
//	    family: myfamily
//	    group: mygroup
//	    custom_tag: custom_value
type AppTags map[string]string

// Get returns the value for a tag key, or empty string if not present.
func (t AppTags) Get(key string) string {
	if t == nil {
		return ""
	}

	return t[key]
}
