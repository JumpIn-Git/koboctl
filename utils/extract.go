package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"github.com/schollz/progressbar/v3"
)

// zip also uses ReadAt as it doesn't read from start to end, this should work with any format
type ExtractFormat struct {
	archives.Extractor
}

type progressFile struct {
	*os.File
	bar *progressbar.ProgressBar
}

func (pf *progressFile) Read(p []byte) (int, error) {
	n, err := pf.File.Read(p)
	if n > 0 {
		_ = pf.bar.Add(n)
	}
	return n, err
}
func (pf *progressFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := pf.File.ReadAt(p, off)
	if n > 0 {
		_ = pf.bar.Add(n)
	}
	return n, err
}

func (ep *ExtractFormat) InitProgress(stat os.FileInfo, f *os.File, name string) *progressFile {
	bar := progressbar.DefaultBytes(stat.Size(), fmt.Sprintf("unpacking %s", name))
	return &progressFile{File: f, bar: bar}
}

var (
	Tgz = ExtractFormat{archives.CompressedArchive{Compression: archives.Gz{}, Extraction: archives.Tar{}}}
	Zip = ExtractFormat{archives.Zip{}}
)

func extractFormat(ctx context.Context, format ExtractFormat, f *os.File, stat os.FileInfo, name string, getOut func(f archives.FileInfo) string) error {
	bar := format.InitProgress(stat, f, name)
	err := format.Extract(ctx, bar, func(ctx context.Context, f archives.FileInfo) error {
		out := getOut(f)
		if out == "" {
			return nil
		}
		if f.IsDir() {
			return os.MkdirAll(out, f.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return err
		}

		stream, err := f.Open()
		if err != nil {
			return err
		}
		defer stream.Close()

		outFile, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, stream)
		return err
	})
	if err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}
	return nil
}

func Extract(ctx context.Context, format ExtractFormat, f *os.File, stat os.FileInfo, dst, name string) error {
	return extractFormat(ctx, format, f, stat, name, func(f archives.FileInfo) string {
		return filepath.Join(dst, f.NameInArchive)
	})
}

type Prefixes map[string]string

func ExtractPrefix(ctx context.Context, format ExtractFormat, f *os.File, stat os.FileInfo, prefixes Prefixes, name string) error {
	return extractFormat(ctx, format, f, stat, name, func(f archives.FileInfo) string {
		var out string
		for prefix, folder := range prefixes {
			if !strings.HasPrefix(f.NameInArchive, prefix) {
				continue
			}
			if strings.HasSuffix(prefix, "/") {
				out = filepath.Join(folder, strings.TrimPrefix(f.NameInArchive, prefix))
			} else if f.NameInArchive == prefix {
				out = folder
			}
			break
		}
		return out
	})
}

func OptWrite(path, content string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0666)
	if errors.Is(err, fs.ErrExist) {
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
