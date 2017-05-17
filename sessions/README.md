# Sessions

This package is a wrapper of [gorilla/sessions](https://www.github.com/gorilla/sessions) to fix the potential [memory leaking problem](https://qortex.com/theplant#groups/560b63da8d93e34b8500da28/entry/58a297e98d93e316d10328f3).

The method is, save the **original request pointer** and doing all later session operations by this **original request pointer**. So even application uses `WithContext` between session calls, the data saved in [gorilla/context](https://www.github.com/gorilla/context) won't be lost anymore.

## Usage

First, setup `SessionConfig` with `GenerateSession` middleware to your application like this:

```go
    func Handler(logger log.Logger, mux *http.ServeMux) http.Handler {
        sessionConf := &sessions.SessionConfig{
            Name:   "session name",
            Key:    "session key",
            Secure: true,
            MaxAge: 0,
        }

        middleware := server.Compose(
            sessions.GenerateSession(sessionConf),
        )

        return middleware(mux)
    }
```

Then, You can fetch session from current request's context. The wrapper provides 3 functions, `Get`, `Put`, `Del`.

Here's an example:

```go
    session := sessions.GetSession(c.Writer, c.Request)

    session.Put("uid", 123)

    key, ok := session.Get("uid")
    // => 123, true

    session.Del("uid")

    key, ok := session.Get("uid")
    // => "", false
```

## The reason of the memory leak problem

The leak is in the [gorilla/context](https://www.github.com/gorilla/context), it uses `*http.Request` as the key for its internal map, but between Get and Clear (via context.ClearHandler) the pointer is changed.

```go
    func (r *Request) WithContext(ctx context.Context) *Request {
            if ctx == nil {
                panic("nil context")
            }
            r2 := new(Request) // original r is replaced by r2, but in the gorilla/context, it still using the r as key
            *r2 = *r
            r2.ctx = ctx
            return r2
    }
```

It is actually a reported issue: https://github.com/gorilla/context/issues/32 that Gorilla contexts play badly with `http.Request.WithContext`.
