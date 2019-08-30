# Backoff
Backoff package is used to generate durations required for retries.
It randomises seed and uses jitter to prevent call clustering.
Durations are generated as channels that are closed after underlying
duration time passes or as context is closed.

Things to be configured:
- base duration - used for base calculation for retry number 1
- jitter - jitter value as factor in 0-1 range. Duration value
will be in range of `baseDuration * (1 +- jitter)`
- max duration - if generated duration exceeds jittered max duration,
jittered max duration will be used (jittered max duration have 50%
chance of exceeding max duration!) 

Example:
```go
ctx := context.Background()
bo := backoff.New().
    SetBaseDuration(100). // in mills
    SetJitter(0.2). // 0-1 range
    SetMaxDelay(100e3) // in mills
    
// 3rd retry -> duration * 2^3 * (1 +- jitter)
<- bo.Wait(ctx, 3) 
```