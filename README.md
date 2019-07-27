1. Logging
2. Readiness
3. Server
4. Middlewares (header forwarding)
5. Authorisation (oAuth)
6. Network calls (header forwarding)

WRAPPERS

Libraries don't log, unless in debug mode
They can use anything for Logging - don't have to keep the agreed logging format
All libraries take:
1. ctx
2. options object for configuration

1. Logging - wrapper - not obligatory - can use something different
Should come with a middleware that adds request id to the logger

2. Readiness
Able to attach it to a router on /liveness and /readiness endpoints
Method that can be used by code to specify that this module is ready or not
- use from web or write own
- Adrian please share

3. Server
Build in shutdown based on ctx.Done()
Able to configure basic parameters - timeouts, etc.
Functionality:
1. Library creates and returns router
outside programm can add middlewares, routes, subrouters, etc..
2. Router is passed to start server function
Use Chi?
Add basic readiness and liveness and ping
-> take a look at rdlabs-feed

4. Middlewares
- Rate limitting - Chi middleware
- CORS - chi Library
- request id generation -> write (uid4)
- adding logger to context (?do we do it like this - for me it was super helpful)
/ add readines and ping to router, not as middlewares, but they can be part of the package

5. oAuth Authentication
- there can be serveral different users and scopes needed in one microservice
- also internal and external authorisation
- can be used as authorization module when one creates a network client
- takse ctx and credentials as parameters
- automatically reconnects if receives info that token is no longer valid or expired
- have a waiting mechanism so other functions can wait for reauthorisation

6. Network requests
- create client
- forward headers (read in middleware)











