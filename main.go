package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

const defaultSize uint64 = 1 << 10

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT pass-name\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	layout := flag.String("layout", "%p", "Layout specifier. %p for password, %o for otp")
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	passName := flag.Arg(1)

	uid, gid, err := CurrentUser()
	if err != nil {
		log.Fatal(err)
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("passfile"),
		fuse.Subtype("passfile"),
	)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err = fs.Serve(c, File{uid: uid, gid: gid, passName: passName, layout: *layout})
		if err != nil {
			log.Println("ohu")
			log.Fatal(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs
	if err := fuse.Unmount(mountpoint); err != nil {
		log.Fatal(err)
	}
	if err := c.Close(); err != nil {
		log.Fatal(err)
	}
}

type File struct {
	uid, gid uint32
	passName string
	layout   string
}

func (f File) Root() (fs.Node, error) {
	return f, nil
}

func getPassword(ctx context.Context, name string) (*bytes.Buffer, error) {
	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)

	cmd := exec.CommandContext(ctx, "pass", "show", name)
	cmd.Stdout = FirstLineWriter(outBuffer)
	cmd.Stderr = errBuffer

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("%w: %s", err, errBuffer)
	}

	return outBuffer, err
}

func getOTP(ctx context.Context, name string) (*bytes.Buffer, error) {
	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)

	cmd := exec.CommandContext(ctx, "pass", "otp", "show", name)
	cmd.Stdout = FirstLineWriter(outBuffer)
	cmd.Stderr = errBuffer

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("%w: %s", err, errBuffer)
	}

	return outBuffer, err
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Uid = f.uid
	a.Gid = f.gid
	a.Mode = 0o400
	a.Size = defaultSize
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	var password, otp string

	if strings.Index(f.layout, "%p") >= 0 {
		p, err := getPassword(ctx, f.passName)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(p)
		if err != nil {
			return nil, err
		}
		password = string(b)
	}

	if strings.Index(f.layout, "%o") >= 0 {
		o, err := getOTP(ctx, f.passName)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(o)
		if err != nil {
			return nil, err
		}
		otp = string(b)
	}

	rendered := f.layout

	rendered = strings.Replace(rendered, "%p", password, -1)
	rendered = strings.Replace(rendered, "%o", otp, -1)

	return []byte(rendered), nil
}
