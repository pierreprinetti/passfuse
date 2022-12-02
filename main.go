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

var passCmd []string

func usage() {
	fmt.Fprintf(os.Stderr, "passfuse mounts a passwordstore secret into a file.\n")
	fmt.Fprintf(os.Stderr, "  github.com/pierreprinetti/passfuse\n")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT pass-name\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Requirements:\n")
	fmt.Fprintf(os.Stderr, "  * pass\n")
	fmt.Fprintf(os.Stderr, "  * pass-otp\n")
}

func main() {
	flag.Usage = usage
	layout := flag.String("layout", "%p", "Layout specifier. %p for password, %o for otp")
	passCmdStr := flag.String("pass-cmd", "pass", "Pass command")
	flag.Parse()

	// TODO: properly split respecting quotes
	passCmd = strings.Split(*passCmdStr, " ")

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

	commands := append(passCmd, "show", name)
	cmd := exec.CommandContext(ctx, commands[0], commands[1:]...)
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

	commands := append(passCmd, "otp", "show", name)
	cmd := exec.CommandContext(ctx, commands[0], commands[1:]...)
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
			log.Print(err)
			return nil, err
		}
		b, err := io.ReadAll(p)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		password = string(b)
	}

	if strings.Index(f.layout, "%o") >= 0 {
		o, err := getOTP(ctx, f.passName)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		b, err := io.ReadAll(o)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		otp = string(b)
	}

	rendered := f.layout

	rendered = strings.Replace(rendered, "%p", password, -1)
	rendered = strings.Replace(rendered, "%o", otp, -1)

	return []byte(rendered), nil
}
