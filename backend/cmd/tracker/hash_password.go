package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"music-tracker/backend/internal/adapter/crypto"
)

// hashPassword reads a password (hidden input on a terminal, first line otherwise)
// and prints its argon2id hash for APP_PASSWORD_HASH.
func hashPassword(in *os.File, out io.Writer) error {
	var password string
	if term.IsTerminal(int(in.Fd())) {
		fmt.Fprint(os.Stderr, "Password: ")
		b, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		password = string(b)
	} else {
		line, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read password: %w", err)
		}
		password = strings.TrimRight(line, "\r\n")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hash, err := crypto.NewArgon2Hasher(crypto.DefaultArgon2Params).Hash(password)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, hash)
	return err
}
