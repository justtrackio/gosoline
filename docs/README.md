# Website

This website is built using [Docusaurus 2](https://docusaurus.io/), a modern static website generator.

##  Installation

```
$ yarn
```

##  Local Development

```
$ yarn start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

##  Build

```
$ yarn build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

##  Deployment

Using SSH:

```
$ USE_SSH=true yarn deploy
```

Not using SSH:

```
$ GIT_USER=<Your GitHub username> yarn deploy
```

If you are using GitHub pages for hosting, this command is a convenient way to build the website and push to the `gh-pages` branch.

## Custom Components

### CodeBlock

To improve the reusability and maintainability of our code blocks, we've implemented a wrapper around Docusaurus's classic CodeBlock component that you can use to show snippets from a code file.

To use this component, first add snippet comments to your code. For example:

Yaml:

```yaml
port: 80
host: example.com

# snippet-start: example
struct_example:
  port: 80
  host: example.com
  timeout: 5s
  threshold: 100
# snippet-end: example
```

Go:

```go
package main

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
)

// snippet-start: main
func main() {
	config := cfg.New()
	
	// snippet-start: configure options
	options := []cfg.Option{
		cfg.WithConfigSetting("host", "localhost"),
		cfg.WithConfigSetting("port", "80"),
	}
	// snippet-end: configure options
    
	if err := config.Option(options...); err != nil {
		panic(err)
	}
}
// snippet-end: main
```

Note that:

1. Every snippet must have a `snippet-start` and `snippet-end` comment.
2. The name of the snippet must be the same.
3. Snippets can be nested.
4. Two types of comments are currently supported (`#` and `//`)
5. The snippet comment belongs on its own line.

To use these snippets in your mdx file, you first need to import the custom `CodeBlock` component:

```mdx
import { CodeBlock } from '../components.jsx';
```

Then, you'll use it the same way you'd use the Docusaurus `CodeBlock` component, but you'll include the snippet name:

```mdx
import Main from "!!raw-loader!./src/test/test.go";

<CodeBlock showLineNumbers language="go" snippet="main">{Main}</CodeBlock>
<CodeBlock showLineNumbers language="go" snippet="configure options">{Main}</CodeBlock>
```

In the output, you'll get just the snippet from the code file. If you have highlight comments, those will still work in the snippets. If you have nested snippet comments, they won't appear in the output.
