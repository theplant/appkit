package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-logfmt/logfmt"
)

type kv struct {
	key string
	val string
}

type record map[string][]string

func main() {
	core := group(read())

	if len(os.Args) > 1 {
		for _, v := range os.Args[1:] {
			d := strings.Split(v, "=")
			if len(d) > 1 {
				core = drop(d[0], d[1])(core)
			}
		}
	}

	ch := print(os.Stdout, core)

	<-ch
}

func read() <-chan []kv {
	ch := make(chan []kv, 1)

	d := logfmt.NewDecoder(os.Stdin)

	go func(ch chan<- []kv) {
		defer close(ch)

		for d.ScanRecord() {
			r := []kv{}
			for d.ScanKeyval() {
				r = append(r, kv{string(d.Key()), string(d.Value())})
			}
			ch <- r
		}
	}(ch)

	return ch
}

func group(in <-chan []kv) <-chan record {
	ch := make(chan record, 1)

	go func() {
		defer close(ch)

		for {
			data, ok := <-in
			if !ok {
				return
			}

			m := record{}
			for _, k := range data {
				arr, ok := m[k.key]
				if !ok {
					arr = []string{}
				}
				m[k.key] = append(arr, k.val)
			}
			ch <- m
		}
	}()

	return ch
}

func drop(key, val string) func(<-chan record) <-chan record {
	return func(in <-chan record) <-chan record {
		ch := make(chan record, 1)

		go func() {
			defer close(ch)

			for {
				data, ok := <-in
				if !ok {
					return
				}

				for _, v := range data[key] {
					if v == val {
						goto skip
					}
				}
				ch <- data
			skip:
			}
		}()

		return ch

	}
}

func print(w io.Writer, in <-chan record) <-chan record {
	ch := make(chan record, 1)

	go func() {
		defer close(ch)

		for {
			data, ok := <-in
			if !ok {
				return
			}

			printRecord(data, w)
		}
	}()

	return ch
}

func ex(r record, f string) (string, bool) {
	v, ok := r[f]
	delete(r, f)
	if ok && len(v) > 0 {
		return strings.Join(v, "."), true
	}
	return "", ok

}

func printRecord(r record, w io.Writer) {

	level, ok := ex(r, "level")
	if !ok {
		level = "<none>"
	}

	ts, ok := ex(r, "ts")
	var tsStr string
	if ok {
		tsStr = ts
	} else {
		tsStr = strings.Repeat(" ", len(time.StampMilli))
	}

	msg, ok := ex(r, "msg")
	if len(msg) == 0 {
		msg = "<no msg>"
	}

	colour := "37"
	switch level {
	case "crit":
		colour = "35"
	case "error":
		colour = "31"
	case "warn":
		colour = "33"
	case "info":
		colour = "32"
	case "debug":
		colour = "30"
	}

	output := []string{fmt.Sprintf("\033[%s;1m%s: %s\033[0m (%s)", colour, tsStr, msg, level)}

	for k, v := range r {
		output = append(output, fmt.Sprintf("  %s=%v", k, strings.Join(v, ".")))
	}

	output = append(output, "", "")

	if _, err := w.Write([]byte(strings.Join(output, "\n"))); err != nil {
		panic(err)
	}
}
