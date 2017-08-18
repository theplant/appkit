package encryptedbox

import (
	"fmt"
	"testing"
)

const key = "12345678901234567890123456789012" // 32 bytes
var box SecretBox

func init() {
	b, err := NewSecretBox(key)

	if err != nil {
		panic(err)
	}

	box = b
}

func TestInvalidSecretBoxKey(t *testing.T) {

	cases := []string{
		"",
		"1234567890123456789012345678901",   // 31 bytes
		"123456789012345678901234567890123", // 33 bytes
	}

	for _, key := range cases {
		box, err := NewSecretBox(key)

		if err == nil {
			t.Fatalf("no error creating box with %d byte key", len(key))
		}

		if box != nil {
			t.Fatalf("returned box with %d byte key", len(key))
		}
	}
}

func ExampleSecretBox_Bytes() {
	message := "hello world"

	fmt.Println(message)

	cipher, err := box.Seal([]byte(message))
	if err != nil {
		panic(err)
	}

	plaintext, err := box.Open(cipher)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(plaintext))

	// Output:
	// hello world
	// hello world
}

func ExampleSecretBox_String() {
	message := "hello world"

	fmt.Println(message)

	cipher, err := box.SealString(message)

	if err != nil {
		panic(err)
	}

	plaintext, err := box.OpenString(cipher)

	if err != nil {
		panic(err)
	}

	fmt.Println(plaintext)

	// Output:
	// hello world
	// hello world
}

func ExampleSecretBox_OpenString() {
	// "hello world" sealed with key
	message := "8cd92c57758b88a9208e30943b762b2f38f1719bbc75699b90578af008787cafcf72cdce1deef57880522bb84c6bec34e8cc6d"

	plaintext, err := box.OpenString(message)

	if err != nil {
		panic(err)
	}

	fmt.Println(plaintext)

	// Output:
	// hello world
}
