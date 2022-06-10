# How to build an API

The purpose of Gosoline is, among others, to help with API building. Much of its functionality is API related, and now we will cover the most important parts.

### Package apiserver

An API server is, in the context of Gosoline, a module that runs indefinitely, listens to a port for requests, and provides answers to those requests. Package apiserver provides a convenient way to create API servers:

[embedmd]:# (../../pkg/apiserver/server.go /func New\(/ /ModuleFactory /)
```go
func New(definer Definer) kernel.ModuleFactory 
```

_Definer_ is a method that returns a _*Definitions_:

[embedmd]:# (../../pkg/apiserver/definition.go /type Definer/ /\)\n/)
```go
type Definer func(ctx context.Context, config cfg.Config, logger log.Logger) (*Definitions, error)
```

The idiomatic way to create an apiserver is to instantiate a new _Definitions_ object, then use its many methods to add functionality to it, declare a _Definer_ function, which returns this object, and lastly call `apiserver.New`. Below are two of those methods:

[embedmd]:# (../../pkg/apiserver/definition.go /func \(d \*Definitions\) Handle/ /HandlerFunc\) /)
```go
func (d *Definitions) Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) 
```

This allows to define functionality that is to be run whenever a given HTTP call to a given path is received.

[embedmd]:# (../../pkg/apiserver/definition.go /func \(d \*Definitions\) POST/ /HandlerFunc\) /)
```go
func (d *Definitions) POST(relativePath string, handlers ...gin.HandlerFunc) 
```

Same as `Handle`, except that it only applies to HTTP POST calls. Similar methods exists for each type of HTTP REST call.

### Config structs

Some useful configuration structures for api servers:

[structmd]:# (pkg/apiserver/server.go Settings TimeoutSettings)
**Settings**

Settings structure for an API server.

| field       | type     | default     | description     |
| :------------- | :----------: | :----------: | -----------: |
| Port | string | 8080 | Port the API listens to. |
| Mode | string | release | Mode is either debug, release, test. |
| Compression | CompressionSettings |  | Compression settings. |
| Timeout | TimeoutSettings |  | Timeout settings. |

**TimeoutSettings**

TimeoutSettings configures IO timeouts.

| field       | type     | default     | description     |
| :------------- | :----------: | :----------: | -----------: |
| Read | time.Duration | 60s | Read timeout is the maximum duration for reading the entire request, including the body. |
| Write | time.Duration | 60s | Write timeout is the maximum duration before timing out writes of the response. |
| Idle | time.Duration | 60s | Idle timeout is the maximum amount of time to wait for the next request when keep-alives are enabled |

[structmd end]:#

[structmd]:# (pkg/apiserver/compression.go CompressionSettings)
**CompressionSettings**

CompressionSettings allow the enabling of gzip support for requests and responses. By default compressed requests are accepted, and compressed responses are returned (if suitable).

| field       | type     | default     | description     |
| :------------- | :----------: | :----------: | -----------: |
| Level | string | default |  |
| Decompression | bool | true |  |
| Exclude | CompressionExcludeSettings |  | Exclude files by path, extension, or regular expression from being considered for compression. Useful if you are serving a format unknown to Gosoline. |

[structmd end]:#

## Practical example: money-exchange application

In `examples/getting_started/api` we can see the _money-exchange_ application. To run it:

1. open a terminal, navigate to `examples/getting_started/api`, and type `$ go run main.go`
2. open a second terminal, and type `curl localhost:8080/euro/10/GBP` or `curl localhost:8080/euro-at-date/10/USD/2021-01-03T00:00:00Z`

To understand what _money-exchange_ does, let us start by looking at its _main_ function:

[embedmd]:# (../../examples/getting_started/api/main.go /func main/ /\n}/)
```go
func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithKernelSettingsFromConfig,
		application.WithLoggerHandlersFromConfig,

		application.WithModuleFactory("api", apiserver.New(definer.ApiDefiner)),
		application.WithModuleFactory("currency", currency.NewCurrencyModule()),
	)
}
```

This application creates a kernel, uses _config.dist.yml_ for its configuration, adds modules _"currency"_ and _"api_ to it, then starts the kernel.

These are the contents of _config.dist.yml_:
```yaml
env: dev

app_project: gosoline
app_family: example
app_name: money-exchange

api:
  port: 8080

kvstore:
  currency:
    type: chain
    in_memory:
      max_size: 500000
    application: money-exchange
    elements: [inMemory]
    ttl: 30m
```

An interesting part of this config file is the line that configures the API to expose port 8080. Other configuration values, for the API, can be added in a similar manner. 

Another interesting part is the currency key value store. This is defined as `inMemory` and serves as a local database.

The _"currency"_ module is already defined by Gosoline, and will use this kvstore to store the exchange rates for various currencies. Its main functionalities are:
- it makes an initial call to an external endpoint in order to get exchange rates, and stores them in a kvstore
- it occasionally makes more calls to obtain exchange rates, in order to keep the kvstore updated

The _"api"_ module is our API server, notice that it was created with a call to `apiserver.New`:

[embedmd]:# (../../examples/getting_started/api/main.go /application.WithModuleFactory\(\"api\"/ /\)/)
```go
application.WithModuleFactory("api", apiserver.New(definer.ApiDefiner)
```

The code for _ApiDefiner_ is as follows:

[embedmd]:# (../../examples/getting_started/api/definer/definer.go /func ApiDefiner/ /\n}/)
```go
func ApiDefiner(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
	definitions := &apiserver.Definitions{}

	euroHandler, err := handler.NewEuroHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create euroHandler: %w", err)
	}

	euroAtDateHandler, err := handler.NewEuroAtDateHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create euroAtDateHandler: %w", err)
	}

	definitions.GET("/euro/:amount/:currency", apiserver.CreateHandler(euroHandler))
	definitions.GET("/euro-at-date/:amount/:currency/:date", apiserver.CreateHandler(euroAtDateHandler))

	return definitions, nil
}
```

_ApiDefiner_ does three things:

1. It creates a new _EuroHandler_.
2. Creates a new _EuroAtDateHandler_.
3. It instantiates a new _api.Definitions_ objects to which it adds two routes served by the two handlers.

Notice that you need to define each route you want an apiserver to listen to, and that for each such route you must also specify the handler that is to handle it. 

Another thing to notice is the use of `definitions.GET`. This method configures a given route to be handled by a given handler, for (and only for) every incoming request of type HTTP GET.

Lastly, let us look at this route `"/euro-at-date/:amount/:currency/:date"`. Its prefix `/euro-at-date/` is static, but the three _:name_ constructs following it are path parameters. This means that the handler will be able to access and use each of the following three path parameters: `amount`, `currency`, and `date`. _euroAtDateHandler_ uses these values in the following manner:

[embedmd]:# (../../examples/getting_started/api/handler/handler.go /	currency := request/ /amountString := request\.Params\.ByName\(\"amount\"\)/)
```go
	currency := request.Params.ByName("currency")
	amountString := request.Params.ByName("amount")
```

Both _EuroHandler_ and _EuroAtDateHandler_ do similar things: the first takes in a money amount and a currency, and returns its value in euro, while the other takes in a date as well, and returns the euro value using the exchange rate for that given date.

[embedmd]:# (../../examples/getting_started/api/handler/handler.go /type euroHandler struct/ /\n}/)
```go
type euroHandler struct {
	logger          log.Logger
	currencyService currency.Service
}
```

The _euroHandler_ struct has a _logger_ and a _currencyService_ field. It is a private struct, but there is a convenient way to instantiate one:

[embedmd]:# (../../examples/getting_started/api/handler/handler.go /func NewEuroHandler/ /\)/)
```go
func NewEuroHandler(ctx context.Context, config cfg.Config, logger log.Logger)
```

The most important method of _euroHanlder_ is _Handle_:

[embedmd]:# (../../examples/getting_started/api/handler/handler.go /func \(h \*euroHandler\) Handle/ /\n}/)
```go
func (h *euroHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, error error) {
	currency := request.Params.ByName("currency")
	amountString := request.Params.ByName("amount")
	amount, err := strconv.ParseFloat(amountString, 64)
	if err != nil {
		h.logger.Error("cannot parse amount %s: %w", amountString, err)

		return apiserver.NewStatusResponse(http.StatusBadRequest), nil
	}

	result, err := h.currencyService.ToEur(requestContext, amount, currency)
	if err != nil {
		h.logger.Error("cannot convert amount %f with currency %s: %w", amount, currency, err)

		return apiserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	return apiserver.NewJsonResponse(result), nil
}
```

_Handle_ will be called to service each request to the route _euroHandler_ is registered for. 
- The first thing it does is parse its input URL parameters: _currency_ and _amount_.
- Secondly, it calls _currencyService_ to do the actual currency conversion.
- Lastly, it returns the result as a JSON response.

## Wrapping it up

Having seen a sample API server, we can look into more detailed functionality: [Integration tests for your API](integration_tests.md)
