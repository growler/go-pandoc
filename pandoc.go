package pandoc

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Pandoc's run configuration.
type Pandoc struct {
	Path   string   // Path to pandoc executable
	Format string   // Format
	Ext    []string // List of format extensions, each must start with '+' or '-'
	Opts   []string // Additional options
}

func (c *Pandoc) WithExt(ext string) *Pandoc {
	for i := range c.Ext {
		if c.Ext[i] == "-"+ext {
			c.Ext[i] = "+" + ext
			break
		} else if c.Ext[i] == "+"+ext {
			break
		}
	}
	return c
}

func (c *Pandoc) WithoutExt(ext string) *Pandoc {
	for i := range c.Ext {
		if c.Ext[i] == "-"+ext {
			break
		} else if c.Ext[i] == "+"+ext {
			c.Ext[i] = "-" + ext
			break
		}
	}
	return c
}

// Add an option to the configuration. Accepts:
//   - single-letter option, e.g. "s"
func (c *Pandoc) WithOpt(opt string, val ...string) *Pandoc {
	if opt == "" {
		return c
	}
	if len(opt) == 1 {
		c.Opts = append(c.Opts, "-"+opt)
		if len(val) > 0 {
			c.Opts = append(c.Opts, val[0])
		}
	} else if len(val) > 0 {
		c.Opts = append(c.Opts, "--"+opt+"="+strings.Join(val, ","))
	} else {
		c.Opts = append(c.Opts, "--"+opt)
	}
	return c
}

func (c *Pandoc) findPandoc() (string, error) {
	if c.Path != "" {
		return c.Path, nil
	}
	if this, err := os.Executable(); err == nil {
		pandoc, err := exec.LookPath(filepath.Join(filepath.Dir(this), "pandoc"))
		if err == nil || errors.Is(err, exec.ErrDot) {
			c.Path = pandoc
			return c.Path, nil
		}
	}
	if pandoc, err := exec.LookPath("pandoc"); err == nil {
		c.Path = pandoc
		return c.Path, nil
	}
	return "", exec.ErrNotFound
}

func (c *Pandoc) loadPandocDocument(r io.Reader) (*Doc, error) {
	pandoc, err := c.findPandoc()
	if err != nil {
		return nil, err
	}
	var cmd = &exec.Cmd{
		Path: pandoc,
		Args: append([]string{
			"pandoc",
			"-tjson",
			strings.Join(append([]string{"-f", c.Format}, c.Ext...), ""),
		}, c.Opts...),
		Stdin:  r,
		Stderr: os.Stderr,
	}
	w, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p, err := Read(w)
	if err != nil {
		_, _ = io.Copy(io.Discard, w)
		_ = cmd.Wait()
		return nil, err
	}
	if err = cmd.Wait(); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *Pandoc) storePandocDocument(w io.Writer, p *Doc) error {
	pandoc, err := c.findPandoc()
	if err != nil {
		return err
	}
	var cmd = &exec.Cmd{
		Path: pandoc,
		Args: append([]string{
			"pandoc",
			"-fjson",
			strings.Join(append([]string{"-t", c.Format}, c.Ext...), ""),
		}, c.Opts...),
		Stdout: w,
		Stderr: os.Stderr,
	}
	r, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := Write(r, p); err != nil {
		_ = r.Close()
		_ = cmd.Wait()
		return err
	}
	if err = r.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func (c *Pandoc) ReadFile(f string) (*Doc, error) {
	r, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return c.loadPandocDocument(r)
}

func (c *Pandoc) Read(r io.Reader) (*Doc, error) {
	return c.loadPandocDocument(r)
}

func (c *Pandoc) WriteFile(f string, p *Doc) error {
	w, err := os.Create(f)
	if err != nil {
		return err
	}
	defer w.Close()
	return c.storePandocDocument(w, p)
}

func (c *Pandoc) Write(w io.Writer, p *Doc) error {
	return c.storePandocDocument(w, p)
}
