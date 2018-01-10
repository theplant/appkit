package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/kr/logfmt"
	"github.com/theplant/appkit/log"
)

type kv struct {
	key string
	val interface{}
}

type kvs []*kv

func (k *kvs) HandleLogfmt(key, val []byte) error {
	kv := &kv{
		key: string(key),
		val: string(val),
	}
	*k = append(*k, kv)
	return nil
}

func main() {
	buf := bufio.NewReader(os.Stdin)

	var data kvs

	for {
		line, err := buf.ReadBytes('\n')

		data = make(kvs, 0)
		valLen := 0
		if err := logfmt.Unmarshal(line, &data); err == nil {
			r := []interface{}{}
			for _, d := range data {
				r = append(r, d.key, d.val)
				valLen += len(d.val.(string))
			}

			if valLen > 0 {
				fmt.Print(log.PrettyFormat(r...))
			} else {
				fmt.Print(string(line))
			}
		} else {
			fmt.Println(string(line))
			fmt.Println("error parsing log output", err)
		}
		// break after, to not miss the last line before EOF
		if err != nil {
			break
		}
	}
}
