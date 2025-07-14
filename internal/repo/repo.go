package repo

import (
	"bufio"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	bufio_ "github.com/cobratbq/goutils/std/bufio"
	"github.com/cobratbq/goutils/std/builtin"
	"github.com/cobratbq/goutils/std/errors"
	hash_ "github.com/cobratbq/goutils/std/hash"
	io_ "github.com/cobratbq/goutils/std/io"
	"github.com/cobratbq/goutils/std/log"
	os_ "github.com/cobratbq/goutils/std/os"
	strings_ "github.com/cobratbq/goutils/std/strings"
	"golang.org/x/crypto/blake2b"
)

const SUBDIR_REPO = "repo"
const PREFIX_TEMPREPOFILE = "temp--"
const SUFFIX_PROPERTIES = ".properties"
const PROP_HASH = "hash"
const PROP_NAME = "name"
const SECTION_TITLES = "titles"

func Props() [2]string {
	return [...]string{PROP_HASH, PROP_NAME}
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

func (r *Repo) repofilepath(path string) string {
	if path == "" || path == "." {
		return filepath.Join(r.location, SUBDIR_REPO)
	}
	return filepath.Join(r.location, SUBDIR_REPO, path)
}

func (r *Repo) temprepofile() (*os.File, string, error) {
	var err error
	var tempf *os.File
	path := r.repofilepath("")
	if tempf, err = os.CreateTemp(path, PREFIX_TEMPREPOFILE); err != nil {
		return nil, "", errors.Context(err, "failed to create temp file for acquisition")
	}
	return tempf, tempf.Name(), nil
}

func OpenRepo(location string) Repo {
	// TODO move default name to template.properties
	props := Props()
	return Repo{location: location, props: props[:]}
}

func (r *Repo) Abort(tempid string) error {
	// FIXME locate and delete tempid document
	return errors.ErrFailure
}

func (r *Repo) Check() error {
	var entries []os.DirEntry
	var err error

	if entries, err = os.ReadDir(r.repofilepath("")); err != nil {
		// FIXME fine-tune error handling
		return err
	}
	for _, e := range entries {
		log.Traceln("Processing repo-entry…", e.Name())
		if !e.Type().IsRegular() {
			log.Infoln(e.Name(), ": not a regular file.")
		}
		if strings.HasSuffix(e.Name(), SUFFIX_PROPERTIES) {
			// properties-files are processed in conjuction with the corresponding binary file.
			if !os_.ExistsFile(r.repofilepath(strings.TrimSuffix(e.Name(), SUFFIX_PROPERTIES))) {
				log.Infoln("Encountered properties-file without corresponding object binary:", e.Name())
				// FIXME remove this file?
			}
			continue
		}
		if strings.HasPrefix(e.Name(), PREFIX_TEMPREPOFILE) {
			if err = os.Remove(r.repofilepath(e.Name())); err == nil {
				log.Traceln("Removed temporary file:", e.Name())
			} else {
				log.Infoln("Failed to remove old temporary file '"+e.Name()+"':", err.Error())
			}
			continue
		}
		if checksum, err := hash_.HashFile(builtin.Expect(blake2b.New512(nil)), r.repofilepath(e.Name())); err != nil {
			log.Infoln("Failed to checksum repo-object:", hex.EncodeToString(checksum))
			continue
		} else if e.Name() != hex.EncodeToString(checksum) {
			log.Infoln("Repo-object '"+e.Name()+"': checksum does not match:", hex.EncodeToString(checksum))
			// FIXME what to message: corruption? allow renaming? (don't forget corresponding properties)
			continue
		}
		// FIXME check for duplicate names, i.e. some documents might not show up when symlinking by duplicate names.
		// Checking characteristics of file properties.
		if !os_.ExistsFile(r.repofilepath(e.Name() + SUFFIX_PROPERTIES)) {
			// TODO check if valid '.properties' file, i.e. readable, parsable content.
			log.Infoln(e.Name()+SUFFIX_PROPERTIES, ": expected a properties-file.")
		}
		if o, err := r.Open(e.Name()); err == nil {
			if hashspec, ok := o.Props[PROP_HASH]; !ok {
				log.Infoln(e.Name(), ": missing 'hash' property.")
			} else if classifier, value, ok := strings.Cut(hashspec, ":"); !ok || classifier != "blake2b" || value != e.Name() {
				log.Infoln(e.Name(), ": invalid properties")
			}
			if name, ok := o.Props[PROP_NAME]; !ok {
				log.Infoln(e.Name(), ": missing 'name' property.")
			} else if !os_.ExistsIsSymlink(filepath.Join(r.location, SECTION_TITLES, name)) {
				if err := os.Symlink(filepath.Join("..", SUBDIR_REPO, e.Name()), filepath.Join(r.location, SECTION_TITLES, name)); err != nil {
					log.Infoln(e.Name(), ": failed to create symlink at expected location:", err.Error())
				} else {
					log.Infoln(e.Name(), ": missing symlink in document titles recreated.")
				}
			}
		} else {
			log.Infoln(e.Name(), ": failed to parse properties: ", err.Error())
		}
	}

	if entries, err = os.ReadDir(filepath.Join(r.location, SECTION_TITLES)); err != nil {
		// FIXME fine-tune error handling
		return err
	}
	for _, e := range entries {
		log.Traceln("Processing titles-entry…", e.Name())
		path := filepath.Join(r.location, SECTION_TITLES, e.Name())
		log.Infoln("Path: ", path)
		if linkpath, err := os.Readlink(path); err == nil {
			objName := filepath.Base(linkpath)
			log.Infoln("Source-path: ", objName)
			if o, err := r.Open(objName); err == nil {
				if o.Props[PROP_NAME] != e.Name() {
					log.Traceln("CHECK: titles document name does not match with 'name' property. Renaming…")
					// FIXME check for symlink targets and if target does not match names, correct/remove? (We also make new symlinks if one does not exist for PROP_NAME)
					if err := os.Rename(path, filepath.Join(r.location, SECTION_TITLES, o.Props[PROP_NAME])); err != nil {
						log.Infoln(e.Name(), ": failed to rename object to proper name: ", err.Error())
					}
				}
			} else {
				log.Traceln("CHECK: titles document does not correctly link to repo-object. Deleting…")
				log.Infoln(e.Name(), ": failed to open corresponding repo-object: ", err.Error())
				if err := os.Remove(path); err != nil {
					log.Infoln("Failed to delete bad symlink in titles: ", err.Error())
				}
			}
		} else {
			log.Traceln("Failed to read link for '" + path + "'. Deleting bad link.")
			log.Infoln(e.Name(), ": failed to query symlink without error: ", err.Error())
		}
	}
	// TODO count errors and report back
	return nil
}

func (r *Repo) writeProperties(objname, checksumhex, name string) error {
	return os.WriteFile(r.repofilepath(objname)+SUFFIX_PROPERTIES, []byte(PROP_HASH+"=blake2b:"+checksumhex+"\n"+PROP_NAME+"="+name+"\n"), 0600)
}

type RepoObj struct {
	Name  string
	Props map[string]string
}

func (r *Repo) Acquire(reader io.Reader, name string) (RepoObj, error) {
	log.Traceln("Acquiring new document into repository…")
	tempf, tempfname, err := r.temprepofile()
	if err != nil {
		return RepoObj{}, errors.Context(err, "failed to create temporary file for storing content in repo")
	}
	defer io_.CloseLogged(tempf, "Failed to gracefully close temporary file")
	log.Traceln("Tempf:", tempfname)
	fhash := builtin.Expect(blake2b.New512(nil))
	if _, err := io.Copy(io.MultiWriter(tempf, fhash), reader); err != nil {
		return RepoObj{}, errors.Context(err, "error while copying contents into repository")
	}
	checksumhex := hex.EncodeToString(fhash.Sum(nil))
	log.Traceln("checksum:", checksumhex)
	if err := os.Rename(tempfname, r.repofilepath(checksumhex)); err != nil {
		return RepoObj{}, errors.Context(err, "failed to move temporary file '"+tempfname+"' to definite repo-object location '"+checksumhex+"'")
	}
	if err := r.writeProperties(checksumhex, checksumhex, name); err != nil {
		return RepoObj{}, errors.Context(err, "failed to write properties-file")
	}
	// FIXME get RepoObj instance from writeProperties, instead of going through Open?
	log.Traceln("Completed acquisition. (object: " + checksumhex + ")")
	return r.Open(checksumhex)
	// FIXME where to start symlinking the repo content?
	//if err := os.Symlink(filepath.Join("..", SUBDIR_REPO, checksumhex), filepath.Join(r.location, SECTION_TITLES, name)); err != nil {
	//	return errors.Context(err, "failed to create symlink in titles")
	//}
}

// Save saves updated repo-object properties to the repository.
func (r *Repo) Save(obj RepoObj) error {
	// FIXME update symlinks?
	return r.writeProperties(obj.Name, obj.Props[PROP_HASH], obj.Props[PROP_NAME])
}

func (r *Repo) Open(objname string) (RepoObj, error) {
	propspath := r.repofilepath(objname + SUFFIX_PROPERTIES)
	props, err := bufio_.OpenFileProcessStringLinesFunc(propspath, '\n', func(s string) ([2]string, error) {
		// TODO fine-tuning trimming whitespace for comment-line matching
		if len(s) == 0 || strings_.AnyPrefix(strings.TrimLeft(s, " \t"), "#", "!") {
			return [2]string{}, bufio_.ErrProcessingIgnore
		}
		// TODO trim whitespacing?
		// TODO support ':' separator?
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
	// TODO check allowed properties? (permit unknown properties?, as forward-compatibility?)
	return RepoObj{Name: objname, Props: propmap}, nil
}

func (r *Repo) List() ([]RepoObj, error) {
	var err error
	var direntries []os.DirEntry
	if direntries, err = os.ReadDir(r.repofilepath("")); err != nil {
		return nil, errors.Context(err, "failed to open repo-data for listing content")
	}
	var objects []RepoObj
	for _, e := range direntries {
		if strings.HasSuffix(e.Name(), SUFFIX_PROPERTIES) {
			continue
		}
		if obj, err := r.Open(e.Name()); err == nil {
			objects = append(objects, obj)
		} else {
			log.Infoln("Skipping", e.Name(), ": failed to open repo-object:", err.Error())
		}
	}
	return objects, nil
}
