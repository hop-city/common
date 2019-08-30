# Readiness
Readiness package is used as centralized key value store for ready
state of multiple parts of the system. Each part of code can register
its status with
```go
func Set(k string, v bool)
````
where `k` is the name of the part of the system and `v` stands
for current readiness state.

One can check resulting app readiness state with
```go
func IsReady() bool
````

Each change in state and current status on IsReady are logged using
zerolog instance in info level.

## Endpoints
There are two ways of using readiness package to server `/liveness`
and `/readiness` endpoints:
- one can use `.Middleware` - it is good choice when one want to attach
it before authorization middleware
- `.Attach` function can be used to attach it to chi.Router

Also for projects that doesn't use http server, readiness package
comes with it's own simple server that uses Attach internally. It can
be run with `.StartServer`


