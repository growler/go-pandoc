package pandoc

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A configuration for running pandoc executable.
type Conf struct {
	Pandoc string   // Path to pandoc executable
	Dir    string   // Working directory
	Format string   // Format to load or store.
	Ext    []string // List of format extensions, each must start with '+' or '-'
	Opts   []string // Additional options
}

var DefaultFormat = Conf{
	Format: "markdown",
}

// Makes a new Conf for format f.
func Format(f string) Conf {
	return Conf{Format: f}
}

// Returns a Conf with a specified path to pandoc executable.
func (c Conf) WithPandoc(path string) Conf {
	c.Pandoc = path
	return c
}

func (c Conf) WithDir(dir string) Conf {
	c.Dir = dir
	return c
}

func (c Conf) WithExt(ext string) Conf {
	for i := range c.Ext {
		if c.Ext[i] == "-"+ext {
			c.Ext = append(append(c.Ext[:i], "+"+ext), c.Ext[i+1:]...)
			return c
		} else if c.Ext[i] == "+"+ext {
			return c
		}
	}
	c.Ext = append(c.Ext, "+"+ext)
	return c
}

func (c Conf) WithoutExt(ext string) Conf {
	for i := range c.Ext {
		if c.Ext[i] == "-"+ext {
			return c
		} else if c.Ext[i] == "+"+ext {
			c.Ext = append(append(c.Ext[:i], "-"+ext), c.Ext[i+1:]...)
			return c
		}
	}
	c.Ext = append(c.Ext, "-"+ext)
	return c
}

// Add an option to the configuration. Accepts:
//   - single-letter option, e.g. "s"
//   - single-letter option with value, e.g. "s", "foo"
//   - long option, e.g. "smart"
//   - long option with value, e.g. "smart", "foo"
func (c Conf) WithOpt(opt string, val ...string) Conf {
	if opt == "" {
		return c
	}
	if len(opt) == 1 {
		c.Opts = append(c.Opts, "-"+opt)
		if len(val) == 1 {
			c.Opts = append(c.Opts, val[0])
		} else if len(val) > 1 {
			c.Opts = append(c.Opts, val[0]+"="+val[1])
		}
	} else if len(val) == 0 {
		c.Opts = append(c.Opts, "--"+opt)
	} else if len(val) == 1 {
		c.Opts = append(c.Opts, "--"+opt+"="+val[0])
	} else if len(val) > 1 {
		c.Opts = append(c.Opts, "--"+opt+"="+val[0]+":"+val[1])
	}
	return c
}

func (c *Conf) pandocExecutable() (string, error) {
	if c.Pandoc != "" {
		return c.Pandoc, nil
	}
	if this, err := os.Executable(); err == nil {
		pandoc, err := exec.LookPath(filepath.Join(filepath.Dir(this), "pandoc"))
		if err == nil || errors.Is(err, exec.ErrDot) {
			return pandoc, nil
		}
	}
	if pandoc, err := exec.LookPath("pandoc"); err == nil {
		return pandoc, nil
	} else {
		return "", fmt.Errorf("pandoc executable is not found: %w", err)
	}
}

func (c *Conf) loadCmd() (*exec.Cmd, error) {
	pandoc, err := c.pandocExecutable()
	if err != nil {
		return nil, err
	}
	return &exec.Cmd{
		Path: pandoc,
		Dir:  c.Dir,
		Args: append([]string{
			"pandoc",
			"-tjson",
			strings.Join(append([]string{"-f", c.Format}, c.Ext...), ""),
		}, c.Opts...),
	}, nil
}

func (c *Conf) storeCmd() (*exec.Cmd, error) {
	pandoc, err := c.pandocExecutable()
	if err != nil {
		return nil, err
	}
	return &exec.Cmd{
		Path: pandoc,
		Dir:  c.Dir,
		Args: append([]string{
			"pandoc",
			"-fjson",
			strings.Join(append([]string{"-t", c.Format}, c.Ext...), ""),
		}, c.Opts...),
	}, nil
}

func LoadFrom(r io.Reader, conf Conf) (*Pandoc, error) {
	cmd, err := conf.loadCmd()
	if err != nil {
		return nil, err
	}
	ip, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	op, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	var readErr = make(chan error, 1)
	go func() {
		_, err := io.Copy(ip, r)
		_ = ip.Close()
		readErr <- err
	}()
	p, err := ReadFrom(op)
	if err != nil {
		_, _ = io.Copy(io.Discard, op)
		_ = cmd.Wait()
		return nil, err
	}
	if err = cmd.Wait(); err != nil {
		return nil, err
	}
	return p, nil
}

func LoadFile(f string, conf Conf) (*Pandoc, error) {
	cmd, err := conf.loadCmd()
	if err != nil {
		return nil, err
	}
	cmd.Args = append(cmd.Args, f)
	op, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p, err := ReadFrom(op)
	if err != nil {
		_, _ = io.Copy(io.Discard, op)
		_ = cmd.Wait()
		return nil, err
	}
	if err = cmd.Wait(); err != nil {
		return nil, err
	}
	return p, nil
}

func LoadFiles(f []string, conf Conf) (*Pandoc, error) {
	cmd, err := conf.loadCmd()
	if err != nil {
		return nil, err
	}
	cmd.Args = append(cmd.Args, f...)
	op, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p, err := ReadFrom(op)
	if err != nil {
		_, _ = io.Copy(io.Discard, op)
		_ = cmd.Wait()
		return nil, err
	}
	if err = cmd.Wait(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Pandoc) StoreTo(w io.Writer, conf Conf) error {
	cmd, err := conf.storeCmd()
	if err != nil {
		return err
	}
	ip, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := p.write(ip); err != nil {
		_ = ip.Close()
		_ = cmd.Wait()
		return err
	}
	if err = ip.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func (p *Pandoc) StoreFile(f string, conf Conf) error {
	conf = conf.WithOpt("o", f)
	cmd, err := conf.storeCmd()
	if err != nil {
		return err
	}
	ip, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := p.write(ip); err != nil {
		_ = ip.Close()
		_ = cmd.Wait()
		return err
	}
	if err = ip.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func StoreTo(w io.Writer, conf Conf, meta Meta, docs ...*Pandoc) error {
	cmd, err := conf.storeCmd()
	if err != nil {
		return err
	}
	ip, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := writeMany(ip, meta, docs...); err != nil {
		_ = ip.Close()
		_ = cmd.Wait()
		return err
	}
	if err = ip.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func StoreFile(f string, conf Conf, meta Meta, docs ...*Pandoc) error {
	conf = conf.WithOpt("o", f)
	cmd, err := conf.storeCmd()
	if err != nil {
		return err
	}
	ip, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := writeMany(ip, meta, docs...); err != nil {
		_ = ip.Close()
		_ = cmd.Wait()
		return err
	}
	if err = ip.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}
