# Gosoline middleware

We saw in the [Kernel](kernel.md) section, that the _kernel.Kernel_ interface offers a method to add middleware to a kernel:

[embedmd]:# (../../pkg/kernel/kernel.go /AddMiddleware\(/ /\)/)
```go
AddMiddleware(middleware Middleware, position Position)
```

The position of a middleware is one of two values:

[embedmd]:# (../../pkg/kernel/middleware.go /type Position/ /\)/)
```go
type Position string

const (
	PositionBeginning Position = "beginning"
	PositionEnd       Position = "end"
)
```

Lastly, a _Middleware_ is a function that takes in the following parameters:

[embedmd]:# (../../pkg/kernel/middleware.go /type \(/ /\n\)/)
```go
type (
	Middleware func(ctx context.Context, config cfg.Config, logger log.Logger, next Handler) Handler
	Handler    func()
)
```

With these notions, we are ready for an example. Note that the example is just to illustrate middleware behaviour, and that Gosoline already provides several useful out-of-the box middlewares, like `aws.AttemptLoggerInitMiddleware`, `aws.AttemptLoggerRetryMiddleware` ,`db_repo.KernelMiddlewareChangeHistory`, etc.

## Usage example

[embedmd]:# (../../examples/getting_started/middleware/main.go)
```go
package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	app := application.New()
	app.Add("hello", moduleFactory)
	app.AddMiddleware(two, kernel.PositionBeginning)
	app.AddMiddleware(one, kernel.PositionBeginning)
	app.Run()
}

func moduleFactory(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return &helloWorldModule{}, nil
}

type helloWorldModule struct{}

func (h *helloWorldModule) Run(ctx context.Context) error {
	fmt.Println("Hello World")

	return nil
}

func one(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
	return func() {
		fmt.Println("Beginning of one")

		next()

		fmt.Println("End of one")
	}
}

func two(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
	return func() {
		fmt.Println("Beginning of two")

		next()

		fmt.Println("End of two")
	}
}
```

The _main_ function will create a new kernel, add module _"hello"_ to it, add two middleware functions, and start running the kernel. The two middleware functions are very similar, let us take a look at `one`:

[embedmd]:# (../../examples/getting_started/middleware/main.go /func one/ /\n}/)
```go
func one(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
	return func() {
		fmt.Println("Beginning of one")

		next()

		fmt.Println("End of one")
	}
}
```

`one` returns a _kernel.Handler_ object, namely a function without parameters nor return values:

[embedmd]:# (../../pkg/kernel/middleware.go /Handler    func/ /\)/)
```go
Handler    func()
```

The _Handler_ returned by `one` will fist execute some code, in this case print _"Beginning of one"_, then invoke the next handler, and lastly, execute another block of code, this time printing _"End of one"_.

Look again at the order in which we added our middleware functions:

[embedmd]:# (../../examples/getting_started/middleware/main.go /func main/ /\n}/)
```go
func main() {
	app := application.New()
	app.Add("hello", moduleFactory)
	app.AddMiddleware(two, kernel.PositionBeginning)
	app.AddMiddleware(one, kernel.PositionBeginning)
	app.Run()
}
```

When creating the kernel, its middleware chain is empty, and we add `two` at the beginning of it. Then we add `one` with position `kernel.PositionBeginning`, which now makes the middleware chain look like {`one`, `two`}. Running the code confirms this:

```
$ go run main.go 
Beginning of one
Beginning of two
Hello World
End of two
End of one
```

At runtime, the middleware chain for the _"hello"_ module will look like this: `one` -> `two` -> _"hello"_.

The first function in the middleware chain, in this case `one`, starts running and prints `Beginning of one`. Then it invokes `next()` and the next function in the middleware chain, that is `two`, starts running, and prints `Beginning of two`. It invokes `next()` and now module _"hello"_ gets to run, printing `Hello World`. As module _"hello"_ returns, function `two` continues by printing `"End of two"` and lastly function `one` will resume and print `"End of one"`.

## Wrapping it up

Gosoline's middleware offers a convenient way to run extra code before and after each module. Having an understanding of Gosoline kernels, modules and middleware, we are now ready for the next step: [How to build an API](how_to_build_an_api.md)
