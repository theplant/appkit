// Lifted from Gin

// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package server

import (
	"net/http"
)

func Recovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var statusCode int

		defer func() {
			if statusCode != 0 {
				rw.WriteHeader(statusCode)
			}
		}()

		defer RecoverAndSetStatusCode(&statusCode)

		h.ServeHTTP(rw, r)
	})
}

func RecoverAndSetStatusCode(statusCode *int) {
	if err := recover(); err != nil {
		*statusCode = http.StatusInternalServerError
		panic(err)
	}
}
