# Sessions

This package is a wrapper of [gorilla/sessions](https://www.github.com/gorilla/sessions) to fix the potential [memory leaking problem](https://qortex.com/theplant#groups/560b63da8d93e34b8500da28/entry/58a297e98d93e316d10328f3).

The method is, save the **original request pointer** and doing all later session operations by this **original request pointer**. So even application uses `WithContext` between session calls, the data saved in [gorilla/context](https://www.github.com/gorilla/context) won't be lost anymore.

## Usage

First, setup `Config` with `WithSession` middleware to your application like this:

```go
    func Handler(logger log.Logger, mux *http.ServeMux) http.Handler {
        sessionConf := &sessions.Config{
            Name:   "session name",
            Key:    "session key",
            Secure: true,
            MaxAge: 0,
        }

        middleware := server.Compose(
            sessions.WithSession(sessionConf),
        )

        return middleware(mux)
    }
```

Then, You can fetch session from current request's context. The wrapper provides 3 functions, `Get`, `Put`, `Del`.

Here's an example:

```go
    sessions.Put(c.Request.Context(), "uid", 123)

    key, err := sessions.Get(c.Request.Context(), "uid")
    // => 123, nil

    session.Del(c.Request.Context(), "uid")

    key, err := session.Get(c.Request.Context(), "uid")
    // => "", "Cannot find value for: 'uid'"
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

## General cookie store using [`gorilla/sessions.CookieStore`](https://github.com/gorilla/sessions/blob/7910f5bb5ac86ab08f97d8bda39b476fc117b684/store.go#L66-L70)

The `sessions.Config` is compatible with [`jinzhu/configor`](https://github.com/jinzhu/configor) with `HttpOnly` and `Secure` enabled by default.

It's easy to `sessions.NewCookieStore` to new a `gorilla/sessions.CookieStore` using `sessions.Config`.

```go
    config := sessions.Config{}

    if err := configor.Load(&config); err != nil {
        panic(err)
    }

    cookieStore := sessions.NewCookieStore(config)
    // using cookieStore bla bla bla...
```