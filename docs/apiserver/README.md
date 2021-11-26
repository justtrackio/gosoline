# apiserver package

### Purpose

Package apiserver offers the backbone of building a server app which exposes an api. 

Its most important parts are detailed below:

### Implementation
* create a new apiserver:

[embedmd]:# (../../pkg/apiserver/server.go /func NewWithInterfaces/ /\n}/)
```go
func NewWithInterfaces(logger log.Logger, router *gin.Engine, tracer tracing.Tracer, s *Settings) (*ApiServer, error) {
	server := &http.Server{
		Addr:         ":" + s.Port,
		Handler:      tracer.HttpHandler(router),
		ReadTimeout:  s.Timeout.Read,
		WriteTimeout: s.Timeout.Write,
		IdleTimeout:  s.Timeout.Idle,
	}

	var err error
	var listener net.Listener
	address := server.Addr

	if address == "" {
		address = ":http"
	}

	// open a port for the server already in this step so we can already start accepting connections
	// when this module is later run (see also issue #201)
	if listener, err = net.Listen("tcp", address); err != nil {
		return nil, err
	}

	logger.Info("serving api requests on address %s", listener.Addr().String())

	apiServer := &ApiServer{
		logger:   logger,
		server:   server,
		listener: listener,
	}

	return apiServer, nil
}
```

* the configuration for an apiserver object looks like this:

[structmd]:# (pkg/apiserver/server.go Settings)
##### Struct **Settings**

Settings stores the settings for an apiserver.

| field       | type     | default     | description     |
| :------------- | :----------: | :----------: | -----------: |
| Port | string | 8080 | Port stores the port where this app will listen on. |
| Mode | string | release |  |
| Compression | CompressionSettings |  |  |
| Timeout | TimeoutSettings |  |  |

[structmd end]:#

* the Run method of an apiserver object:

[embedmd]:# (../../pkg/apiserver/server.go /func \(a \*ApiServer\) Run/ /\n}/)
```go
func (a *ApiServer) Run(ctx context.Context) error {
	go a.waitForStop(ctx)

	err := a.server.Serve(a.listener)

	if err != http.ErrServerClosed {
		a.logger.Error("Server closed unexpected: %w", err)

		return err
	}

	return nil
}
```

### Usage example

[embedmd]:# (../../examples/apiserver/simple-handlers/main.go /func main/ /\n}/)
```go
func main() {
	app := application.New(application.WithConfigFile("config.dist.yml", "yml"))
	app.Add("api", apiserver.New(apiDefiner))
	app.Run()
}
```
