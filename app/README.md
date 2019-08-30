# App
App package is responsible for initialising most basic parts of the app.
- creates zerolog logger instance using __common/logger__ package
- creates context.Context with cancel
- adds logger to context
- creates handler for termination signals - `SIGINT`, `SIGTERM`, `SIGKILL` - that closes ctx.Done channel
- returns context and cancel method



