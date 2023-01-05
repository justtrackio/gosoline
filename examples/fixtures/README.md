# Fixture loader examples
Make sure you set up a database, dynamodb and/or a redis when running the example application and also provide the credentials in the `config.dist.yml` file.

# Notes
Fixtures are loaded within an application context to ensure that app settings properly apply, e.g. clock settings.

# Utility
* Named Fixture Sets allow you to retrieve fixtures by name with simple adjustments of your Fixture Sets
* AutoNumbered provides (locally scoped) monotonically increasing generated ids
