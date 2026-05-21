package cli

import (
	"bufio"
	"flag"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"repo3/internal/client"
)

type createMode int

const (
	createAsk createMode = iota
	createAlways
	createNever
)

func (c command) runPut(args []string) error {
	fs := flag.NewFlagSet("put", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	opts := defaultGlobalOptions()
	c.addClientFlags(fs, &opts)
	var meta metadataFlags
	contentType := fs.String("content-type", "", "object content type")
	fs.Var(&meta, "meta", "custom metadata as key=value; can be repeated")
	createBucket := fs.Bool("create-bucket", false, "create bucket automatically if missing")
	noCreateBucket := fs.Bool("no-create-bucket", false, "fail if bucket is missing")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("usage: repo3 put <local> s3://bucket/key")
	}
	if *createBucket && *noCreateBucket {
		return fmt.Errorf("--create-bucket and --no-create-bucket are mutually exclusive")
	}

	mode := createAsk
	if *createBucket {
		mode = createAlways
	}
	if *noCreateBucket {
		mode = createNever
	}

	localPath := fs.Arg(0)
	s3Path, err := client.ParseS3Path(fs.Arg(1))
	if err != nil {
		return err
	}
	if s3Path.Key == "" {
		return fmt.Errorf("destination must include an object key")
	}

	metadata, err := meta.Map()
	if err != nil {
		return err
	}
	cli := newClient(opts)
	putOpts := client.PutOptions{ContentType: *contentType, UserMetadata: metadata}
	err = c.uploadFile(cli, localPath, s3Path, putOpts)
	if !client.IsCode(err, "NoSuchBucket") {
		if err != nil {
			return err
		}
		if !opts.json {
			_, err := fmt.Fprintf(c.stdout, "uploaded %s to %s\n", localPath, fs.Arg(1))
			return err
		}
		return nil
	}

	shouldCreate, err := c.shouldCreateBucket(mode, s3Path.Bucket)
	if err != nil {
		return err
	}
	if !shouldCreate {
		return err
	}
	if err := cli.CreateBucket(background(), s3Path.Bucket); err != nil {
		return err
	}
	if err := c.uploadFile(cli, localPath, s3Path, putOpts); err != nil {
		return err
	}
	if !opts.json {
		_, err := fmt.Fprintf(c.stdout, "uploaded %s to %s\n", localPath, fs.Arg(1))
		return err
	}
	return nil
}

func (c command) uploadFile(cli *client.Client, localPath string, s3Path client.S3Path, opts client.PutOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer closeIgnore(file)

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if opts.ContentType == "" {
		opts.ContentType = mime.TypeByExtension(filepath.Ext(localPath))
	}
	return cli.PutObject(background(), s3Path.Bucket, s3Path.Key, file, info.Size(), opts)
}

func (c command) shouldCreateBucket(mode createMode, bucket string) (bool, error) {
	switch mode {
	case createAlways:
		return true, nil
	case createNever:
		return false, nil
	default:
		if !isTerminal() {
			return false, nil
		}
		fmt.Fprintf(c.stderr, "Bucket '%s' does not exist.\nCreate GitHub repository '%s'? [y/N] ", bucket, bucket)
		answer, err := bufio.NewReader(c.stdin).ReadString('\n')
		if err != nil {
			return false, err
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		return answer == "y" || answer == "yes", nil
	}
}

func isTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

type metadataFlags []string

func (m *metadataFlags) String() string {
	return strings.Join(*m, ",")
}

func (m *metadataFlags) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func (m metadataFlags) Map() (map[string]string, error) {
	if len(m) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(m))
	for _, raw := range m {
		key, value, ok := strings.Cut(raw, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --meta %q: expected key=value", raw)
		}
		key = strings.TrimSpace(strings.ToLower(key))
		key = strings.TrimPrefix(key, "x-amz-meta-")
		if strings.ContainsAny(key, " \t\r\n:") {
			return nil, fmt.Errorf("invalid --meta key %q", key)
		}
		out[key] = strings.TrimSpace(value)
	}
	return out, nil
}
