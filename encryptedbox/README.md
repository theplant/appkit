# Encrypted Box

## Secret Box

Secret Box provides a simple interface for encryption of data for storage at rest.

It is implemented as a simple wrapper around `golang.org/x/crypto/nacl/secretbox` that takes care of handling the nonce by:

1. Generating a random nonce for each message (192 bit nonce means
   vanishingly small probability of nonce reuse).

2. Encrypting the message with `secretBox.Seal`.

3. Prefixing the nonce onto the encrypted result.

As a wrapper of `secretbox`, the same caveats apply:

> The length of messages is not hidden.

The key must be 32 bytes long (as a requirement from `secretbox`).

Usage:

```
box, err := NewSecretBox(key)

if err != nil {
	panic(err)
}

var message string

// Seal/Encrypt
cipher, err := box.SealString(message)
if err != nil {
	panic(err)
}

// Do something with cipher...

// Open/Decrypt
plaintext, err := box.Open(cipher)
if err != nil {
	panic(err)
}

// Do something with plaintext...
```

## Box

Not implemented yet, but would be good to provide a simplified wrapper around `golang.org/x/crypto/nacl/box`.
