package cli

import "io"

func closeIgnore(closer io.Closer) {
	_ = closer.Close()
}
