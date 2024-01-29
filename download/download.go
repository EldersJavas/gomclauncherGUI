package download

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
	"sync/atomic"

	"github.com/avast/retry-go/v4"
	"github.com/xmdhs/gomclauncher/lang"
	"github.com/xmdhs/gomclauncher/launcher"
	"golang.org/x/sync/errgroup"
)

func (l Libraries) Downassets(i int, c chan int) error {
	if len(l.assetIndex.Objects) == 0 {
		return nil
	}
	g, ectx := errgroup.WithContext(l.cxt)
	g.SetLimit(i)

	n := atomic.Uint64{}

	for _, v := range l.assetIndex.Objects {
		v := v
		g.Go(func() error {
			ok := ver(l.path+`/assets/objects/`+v.Hash[:2]+`/`+v.Hash, v.Hash)
			if !ok {
				d := downinfo{
					url:      `https://resources.download.minecraft.net/` + v.Hash[:2] + `/` + v.Hash,
					path:     l.path + `/assets/objects/` + v.Hash[:2] + `/` + v.Hash,
					Sha1:     v.Hash,
					randurls: l.randurls,
					print:    l.print,
				}
				if err := d.down(ectx); err != nil {
					return err
				}
				c <- len(l.assetIndex.Objects) - int(n.Add(1))
			} else {
				c <- len(l.assetIndex.Objects) - int(n.Add(1))
			}
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return fmt.Errorf("Downassets: %w", err)
	}
	return nil
}

func ver(path, ahash string) bool {
	if ahash != "" {
		var m hash.Hash
		switch len(ahash) {
		case 40:
			m = sha1.New()
		case 64:
			m = sha256.New()
		}
		file, err := os.Open(path)
		if err != nil {
			return false
		}
		defer file.Close()
		if _, err := io.Copy(m, file); err != nil {
			return false
		}
		h := hex.EncodeToString(m.Sum(nil))
		return strings.ToTitle(h) == strings.ToTitle(ahash)
	}
	_, err := os.Stat(path)
	return err == nil
}

func (l Libraries) Downlibrarie(i int, c chan int) error {
	if len(l.librarie.Libraries) == 0 {
		return nil
	}
	g, ctx := errgroup.WithContext(l.cxt)
	g.SetLimit(i)
	n := atomic.Uint64{}

	for _, v := range l.librarie.Libraries {
		v := v
		if !launcher.Ifallow(v) {
			c <- len(l.librarie.Libraries) - int(n.Add(1))
			continue
		}
		path := l.path + `/libraries/` + v.Downloads.Artifact.Path
		if v.Downloads.Artifact.URL == "" {
			c <- len(l.librarie.Libraries) - int(n.Add(1))
			continue
		}
		g.Go(func() error {
			if !ver(path, v.Downloads.Artifact.Sha1) {
				d := downinfo{
					print:    l.print,
					url:      v.Downloads.Artifact.URL,
					path:     path,
					Sha1:     v.Downloads.Artifact.Sha1,
					randurls: l.randurls,
				}
				if err := d.down(ctx); err != nil {
					return err
				}
				c <- len(l.librarie.Libraries) - int(n.Add(1))
			} else {
				c <- len(l.librarie.Libraries) - int(n.Add(1))
			}
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return fmt.Errorf("Downlibrarie: %w", err)
	}
	return nil
}

var FileDownLoadFail = errors.New("file download fail")

func (l Libraries) Downjar(version string) error {
	path := l.path + `/versions/` + version + "/" + version + ".jar"
	if ver(path, l.librarie.Downloads.Client.Sha1) {
		return nil
	}
	_, t := l.auto()
	for i := 0; i < 4; i++ {
		if i == 3 {
			return FileDownLoadFail
		}
		err := get(l.cxt, source(l.librarie.Downloads.Client.URL, t), path)
		if err != nil {
			l.print(lang.Lang("weberr") + " " + source(l.librarie.Downloads.Client.URL, t) + " " + fmt.Errorf("Downjar: %w", err).Error())
			t = l.fail(t)
			continue
		}
		if !ver(path, l.librarie.Downloads.Client.Sha1) {
			l.print(lang.Lang("filecheckerr") + " " + source(l.librarie.Downloads.Client.URL, t))
			t = l.fail(t)
			continue
		}
		break
	}
	return nil
}

type downinfo struct {
	url   string
	path  string
	Sha1  string
	print func(string)
	*randurls
}

func (d downinfo) down(ctx context.Context) error {
	_, f := d.auto()

	err := retry.Do(func() error {
		url := source(d.url, f)
		err := get(ctx, url, d.path)
		if err != nil {
			f = d.fail(f)
			return fmt.Errorf(lang.Lang("weberr")+" "+url+" %w", err)
		}
		if !ver(d.path, d.Sha1) {
			f = d.fail(f)
			return fmt.Errorf(lang.Lang("filecheckerr") + " " + url)
		}
		return nil
	}, append(retryOpts, retry.OnRetry(func(n uint, err error) {
		print(fmt.Sprintf("retry %d: %v", n, err))
	}))...)
	if err != nil {
		return errors.Join(err, FileDownLoadFail)
	}
	d.add(f)
	return nil
}

var retryOpts = []retry.Option{
	retry.Attempts(8),
	retry.LastErrorOnly(true),
}
