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
		err = fs.Serve(c, File{uid: uid, gid: gid, passName: passName})
		if err != nil {
			log.Fatal(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)

	<-sigs
	log.Print("Signal received, closing...")
	c.Close()
	log.Print("Connection closed.")
}

type File struct {
	uid, gid uint32
	passName string
}

func (f File) Root() (fs.Node, error) {
	return f, nil
}

func pass(ctx context.Context, name string) (*bytes.Buffer, error) {
	outBuffer := new(bytes.Buffer)

	{
		cmd := exec.CommandContext(ctx, "pass", "show", name)
		cmd.Stdout = FirstLineWriter(outBuffer)
		if err := cmd.Run(); err != nil {
			log.Print(err)
			return outBuffer, err
		}
	}

	{
		cmd := exec.CommandContext(ctx, "pass", "otp", "show", name)
		cmd.Stdout = FirstLineWriter(outBuffer)
		if err := cmd.Run(); err != nil {
			log.Print(err)
			return outBuffer, err
		}
	}

	return outBuffer, nil
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Uid = f.uid
	a.Gid = f.gid
	a.Mode = 0o400
	a.Size = defaultSize
	return nil
}

func (f File) ReadAll(ctx context.Context) ([]byte, error) {
	r, err := pass(ctx, f.passName)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}
