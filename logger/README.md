# Logger
Common logger is a thin wrapper on zerolog:  
https://github.com/rs/zerolog

### Instantiate logger
```go
func New() *zerolog.Logger
```
Logger is also added to ctx of __app package__ (`ctx, cancel := app.Scaffold()`).

### Control output with env variables
By default logger logs in JSON format.
- LOG_LEVEL - default `info`
- LOG_PRETTY - pretty log instead of JSON - default `false`
- LOG_CALLER - will log caller file and line number - default `false`
- LOG_REVISION - specified revision value will be logged with each entry - default empty

### Middleware
```Go
func Middleware(next http.Handler) http.Handler
```  
Logger comes with middleware that does two things:
1. Adds logger to context. It can be later extracted from context with `zerolog.ctx(context.Context)`
2. If context has `requestId` set with __chi__ middleware, it will be added to zerolog instance as `requestId` field in log.

