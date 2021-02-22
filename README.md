![Gosoline Logo](http://cdn.applike-services.info/public/2019/10/23/gosoline.svg)
------------------
![Gosoline](https://github.com/applike/gosoline/workflows/Gosoline/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/applike/gosoline)](https://goreportcard.com/report/github.com/applike/gosoline)
------------------

Gosoline is our framework which fuels all of our Golang applications. Internally,
we're using a lot of established Go libraries like Viper, Gin, Gorm, etc. and 
put it together into a framework to build wep apis and microservice based 
backend applications. Despite the fact that we already use it in production, 
the current state should be considered as an early alpha. Main things to 
come in the next weeks:


* more tests
* GoDoc
* overall documentation

## Roadmap
* cfg: change callbacks
* cfg: proper default value merging
* test: gateway case
* cloud: bundle aws config
* add error returns to constructors
