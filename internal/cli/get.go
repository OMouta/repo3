package cli

import (
	"fmt"
	"io"
	"os"

	"repo3/internal/client"
)

func (c command) runGet(args []string) error {
	fs, opts, err := c.parseClientFlags("get", args)
	if err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: repo3 get s3://bucket/key <local>")
	}

	s3Path, err := client.ParseS3Path(fs.Arg(0))
	if err != nil {
		return err
	}
	if s3Path.Key == "" {
		return fmt.Errorf("source must include an object key")
	}

	resp, err := newClient(opts).GetObject(background(), s3Path.Bucket, s3Path.Key)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(fs.Arg(1))
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	if !opts.json {
		fmt.Fprintf(c.stdout, "downloaded %s to %s\n", fs.Arg(0), fs.Arg(1))
	}
	return nil
}
