# Logger
Common logger is a thin wrapper on zerolog:  
https://github.com/rs/zerolog

By default logger logs in JSON format.
### Env
- LOG_LEVEL, default `info`
- LOG_PRETTY, pretty log instead of JSON, default false
- LOG_CALLER, should caller and line be logged? default false
- LOG_REVISION, id of revision that will be logged with each entry, default none

### Middleware
Logger comes with middleware that does 2 things:
1. Adds logger to context. It can be later extracted from context with zerolog.ctx()
2. If context has requestId set with chi middleware, it will be added to zerolog instance as `requestId` field.

