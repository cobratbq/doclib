package repo

import (
	"bufio"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	bufio_ "github.com/cobratbq/goutils/std/bufio"
	"github.com/cobratbq/goutils/std/builtin"
	"github.com/cobratbq/goutils/std/errors"
	io_ "github.com/cobratbq/goutils/std/io"
	"golang.org/x/crypto/blake2b"
)

const SUBDIR_REPO = "repo"
const SUFFIX_PROPERTIES = ".properties"

func Props() [1]string {
	return [...]string{"hash"}
}

func Hash(location string) ([64]byte, error) {
	hash := builtin.Expect(blake2b.New512(nil))
	f, err := os.Open(location)
	if err != nil {
		return [64]byte{}, errors.Context(err, "Failed to open file: "+location)
	}
	defer io_.CloseLogged(f, "Failed to gracefully close file: "+location)
	var buf [4096]byte
	if err = bufio_.ReadProcessBytesBatchFunc(bufio.NewReader(f), buf[:], func(buf []byte) error {
		return builtin.Error(hash.Write(buf))
	}); err != nil {
		return [64]byte{}, errors.Context(err, "failed to hash file contents")
	}
	return [64]byte(hash.Sum(nil)), nil
}

type Repo struct {
	location string
	props    []string
}

func (r *Repo) repofile(path string) string {
	if path == "" || path == "." {
		return filepath.Join(r.location, SUBDIR_REPO)
	}
	return filepath.Join(r.location, SUBDIR_REPO, path)
}

func OpenRepo(location string) Repo {
	// TODO move default name to template.properties
	return Repo{location: location, props: []string{"hash"}}
}

func (r *Repo) Acquire(location string, name string) error {
	var err error
	var checksum [64]byte
	if checksum, err = Hash(location); err != nil {
		return errors.Context(err, "failed to hash document contents")
	}
	checksumhex := hex.EncodeToString(checksum[:])
	if err = os.Rename(location, r.repofile(checksumhex)); err != nil {
		return errors.Context(err, "failed to move file to repo")
	}
	if err = os.WriteFile(r.repofile(checksumhex+".properties"), []byte("hash=blake2b:"+checksumhex+"\n"+"name="+name+"\n"), 0600); err != nil {
		return errors.Context(err, "failed to write properties to repo")
	}
	return nil
}

func (r *Repo) Len() int {
	return 5
}

func (r *Repo) Check() error {
	var entries []os.DirEntry
	var err error
	if entries, err = os.ReadDir(r.repofile("")); err != nil {
		// FIXME fine-tune error handling
		return err
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			os.Stderr.WriteString(e.Name() + ": not a regular file.")
		}
		if strings.HasSuffix(e.Name(), SUFFIX_PROPERTIES) {
			// properties-files are processed in conjuction with the corresponding binary file.
			continue
		}
		// Checking characteristics of file properties.
		if info, err := os.Stat(r.repofile(e.Name() + SUFFIX_PROPERTIES)); err != nil || info.Mode()&fs.ModeType != 0 {
			os.Stderr.WriteString(e.Name() + SUFFIX_PROPERTIES + ": expected a properties-file.")
		}
	}
	return nil
}

type RepoObj struct {
	Name  string
	Props map[string]string
}

func (r *Repo) Open(objname string) (RepoObj, error) {
	propspath := r.repofile(objname + SUFFIX_PROPERTIES)
	props, err := bufio_.OpenFileProcessStringLinesFunc(propspath, '\n', func(s string) ([2]string, error) {
		if key, value, ok := strings.Cut(s, "="); ok {
			return [...]string{key, value}, nil
		}
		return [2]string{}, errors.ErrIllegal
	})
	if err != nil {
		return RepoObj{}, errors.Context(err, "failed to parse properties for "+objname)
	}
	propmap := make(map[string]string, len(props))
	for _, p := range props {
		propmap[p[0]] = p[1]
	}
	return RepoObj{Name: objname, Props: propmap}, nil
}
