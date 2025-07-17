package repo

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cobratbq/goutils/assert"
	bufio_ "github.com/cobratbq/goutils/std/bufio"
	"github.com/cobratbq/goutils/std/builtin"
	"github.com/cobratbq/goutils/std/builtin/maps"
	maps_ "github.com/cobratbq/goutils/std/builtin/maps"
	"github.com/cobratbq/goutils/std/builtin/set"
	"github.com/cobratbq/goutils/std/errors"
	hash_ "github.com/cobratbq/goutils/std/hash"
	io_ "github.com/cobratbq/goutils/std/io"
	"github.com/cobratbq/goutils/std/log"
	os_ "github.com/cobratbq/goutils/std/os"
	strings_ "github.com/cobratbq/goutils/std/strings"
	"golang.org/x/crypto/blake2b"
)

const SUBDIR_REPO = "repo"
const PREFIX_REPO_TEMP = "temp--"
const SUFFIX_PROPERTIES = ".properties"
const PROP_HASH = "hash"
const PROP_NAME = "name"
const SECTION_TITLES = "titles"
const TAGGROUP_AUTHORS = "authors"

func Props() []string {
	return []string{PROP_HASH, PROP_NAME}
}

func TagGroups() []string {
	return []string{TAGGROUP_AUTHORS}
}

func Hash(location string) ([64]byte, error) {
	if hash, err := hash_.HashFile(builtin.Expect(blake2b.New512(nil)), location); err == nil {
		return [64]byte(hash), nil
	} else {
		return [64]byte{}, errors.Context(err, "hashing content at '"+location+"'")
	}
}

type Repo struct {
	location string
	props    []string
	cats     map[string][]string
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
	if tempf, err = os.CreateTemp(path, PREFIX_REPO_TEMP); err != nil {
		return nil, "", errors.Context(err, "create temp file for acquisition")
	}
	return tempf, tempf.Name(), nil
}

// TODO consider renaming 'titles' to 'archive' or 'all' or something, to indicate that it lists all documents
func listOptions(location string) ([]string, error) {
	if entries, err := os.ReadDir(location); err != nil {
		return nil, errors.Context(err, "directory missing for tag-group: "+location)
	} else {
		var index []string
		for _, e := range entries {
			index = append(index, strings.ToLower(e.Name()))
		}
		return index, nil
	}
}

func OpenRepo(location string) Repo {
	// TODO move default name to template.properties
	props := Props()
	index := map[string][]string{}
	// FIXME do proper error handling
	for _, e := range builtin.Expect(os.ReadDir(location)) {
		if e.Name() == SUBDIR_REPO || e.Name() == SECTION_TITLES {
			continue
		}
		index[strings.ToLower(e.Name())] = builtin.Expect(listOptions(filepath.Join(location, e.Name())))
	}
	log.Traceln("Category-index:", index)
	return Repo{location: location, props: props[:], cats: index}
}

func (r *Repo) Categories() []string {
	keys := maps.ExtractKeys(r.cats)
	slices.Sort(keys)
	return keys
}

func (r *Repo) Tags(category string) []string {
	if index, ok := r.cats[category]; !ok {
		return nil
	} else {
		return index
	}
}

// TODO this is VERY VERY UGLY but functional, so far... Let's get basics working first.
func (r *Repo) Check() error {
	var entries []os.DirEntry
	var err error

	log.Infoln("Starting repository check…")
	defer log.Infoln("Finished repository check.")

	if entries, err = os.ReadDir(r.repofilepath("")); err != nil {
		return errors.Context(err, "failed to open object-repository directory")
	}
	// FIXME checks don't take into account other kinds of file system objects
	for _, e := range entries {
		log.Traceln("Processing repo-entry…", e.Name())
		// Any non-regular file-system object is a foreign entity.
		if !e.Type().IsRegular() {
			log.Infoln(e.Name(), ": not a regular file.")
			continue
		}
		// Check if properties-file has a corresponding repository object.
		if strings.HasSuffix(e.Name(), SUFFIX_PROPERTIES) {
			// properties-files are processed in conjuction with the corresponding binary file.
			if info, err := os.Stat(r.repofilepath(strings.TrimSuffix(e.Name(), SUFFIX_PROPERTIES))); err != nil {
				log.Infoln("Encountered properties-file without corresponding object-binary:", e.Name())
				if err := os.Remove(r.repofilepath(e.Name())); err != nil {
					log.Warnln("Failed to remove orphaned properties-file '"+e.Name()+"' from repository:", err.Error())
				} else {
					log.Traceln("Removed orphaned properties-file:", e.Name())
				}
			} else if info.Mode()&os.ModeType != 0 {
				log.Warnln("Corresponding file-system object is not a regular file:", info.Name())
			}
			continue
		}
		// Remove abandoned temporary repository objects.
		if strings.HasPrefix(e.Name(), PREFIX_REPO_TEMP) {
			if err = os.Remove(r.repofilepath(e.Name())); err != nil {
				log.Warnln("Failed to remove old temporary file '"+e.Name()+"':", err.Error())
			} else {
				log.Traceln("Removed temporary file:", e.Name())
			}
			continue
		}
		// Comparing file content checksum with binary-object name.
		if checksum, err := hash_.HashFile(builtin.Expect(blake2b.New512(nil)), r.repofilepath(e.Name())); err != nil {
			log.Warnln("Failed to hash repo-object:", hex.EncodeToString(checksum))
		} else if e.Name() != hex.EncodeToString(checksum) {
			log.Warnln("Repo-object '" + e.Name() + "': checksum does not match. Possible corruption. (checksum: " + hex.EncodeToString(checksum) + ")")
		}
		// FIXME check for duplicate names, i.e. some documents might not show up when symlinking by duplicate names.
		// Checking characteristics of file properties.
		if info, err := os.Stat(r.repofilepath(e.Name() + SUFFIX_PROPERTIES)); err != nil {
			log.Warnln(e.Name()+SUFFIX_PROPERTIES, ": properties-file is missing.")
		} else if info.Mode()&os.ModeType != 0 {
			log.Warnln(e.Name()+SUFFIX_PROPERTIES, ": properties-file is not a regular file.")
		} else if o, err := r.Open(e.Name()); err != nil {
			log.Warnln(e.Name(), ": failed to parse properties: ", err.Error())
		} else {
			if hashspec, ok := o.Props[PROP_HASH]; !ok {
				log.Warnln(e.Name(), ": missing 'hash'-property.")
			} else if classifier, value, ok := strings.Cut(hashspec, ":"); !ok || classifier != "blake2b" || value != e.Name() {
				log.Warnln(e.Name(), ": invalid properties", e.Name(), value)
			}
			if name, ok := o.Props[PROP_NAME]; !ok {
				log.Warnln(e.Name(), ": missing 'name' property.")
			} else if info, err := os.Lstat(filepath.Join(r.location, SECTION_TITLES, name)); err != nil {
				// Create symlink when one does not exist under the correct name as stated in the properties.
				// Next we will remove symlinks that refer to repo-objects that have a different name-prop.
				if err := os.Symlink(filepath.Join("..", SUBDIR_REPO, e.Name()), filepath.Join(r.location, SECTION_TITLES, name)); err != nil {
					log.Warnln(e.Name(), ": failed to create symlink at expected location:", err.Error())
				} else {
					log.Traceln(e.Name(), ": missing symlink in document titles recreated.")
				}
			} else if info.Mode()&os.ModeSymlink == 0 {
				log.Warnln(info.Name(), ": a foreign file-system object was found where a symlink to a repo-object was expected.")
			}
			// TODO check directory presence for each of the listed tags.
			log.Traceln("Tags:", o.Tags)
			if entries, err := os.ReadDir(r.location); err != nil {
				log.Warnln("Failed to open root repository directory for tags processing:", err.Error())
			} else {
				for _, e := range entries {
					if e.Name() == SUBDIR_REPO || e.Name() == SECTION_TITLES {
						continue
					}
					// TODO we don't currently check if tag-names are used for which there exists no directory
					maybeTag := strings.ToLower(e.Name())
					if tags, ok := o.Tags[maybeTag]; ok {
						if tagdirs, err := os.ReadDir(filepath.Join(r.location, e.Name())); err != nil {
							log.Warnln("Failed to open tag-directory for tag-group", e.Name())
						} else {
							for _, t := range tagdirs {
								// FIXME strictly speaking, o.Props[PROP_NAME] is not guaranteed. Needs fixing/checking/prevention
								path := filepath.Join(r.location, e.Name(), t.Name(), o.Props[PROP_NAME])
								if _, ok := tags[strings.ToLower(t.Name())]; !ok {
									if os_.ExistsIsSymlink(path) {
										// FIXME improve error handling
										assert.Success(os.Remove(path), "Expected removal to succeed.")
									}
								} else {
									if !os_.ExistsIsSymlink(path) {
										// FIXME improve error handling
										assert.Success(os.Symlink(filepath.Join("..", "..", SUBDIR_REPO, o.Id), filepath.Join(r.location, e.Name(), t.Name(), o.Props[PROP_NAME])), "Expected symlink creation to succeed.")
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if entries, err = os.ReadDir(filepath.Join(r.location, SECTION_TITLES)); err != nil {
		return errors.Context(err, "failed to open directory with titles links")
	}
	for _, e := range entries {
		log.Traceln("Processing titles-entry…", e.Name())
		path := filepath.Join(r.location, SECTION_TITLES, e.Name())
		if linkpath, err := os.Readlink(path); err != nil {
			log.Warnln(e.Name(), ": failed to query symlink without error:", err.Error())
		} else if obj, err := r.Open(filepath.Base(linkpath)); err != nil {
			log.Traceln("titles symlink does not correctly link to repo-object. Deleting…")
			if err := os.Remove(path); err != nil {
				log.Warnln("Failed to delete bad symlink in titles:", err.Error())
			}
		} else if obj.Props[PROP_NAME] != e.Name() {
			log.Traceln("titles document name does not match with 'name' property. Removing…")
			if err := os.Remove(path); err != nil {
				// Previously, we created symlinks when they don't exist at expected name. Now we remove
				// existing symlinks which refer to repo-objects with a different name.
				log.Warnln(e.Name(), ": failed to rename object to proper name:", err.Error())
			}
		}
	}
	// TODO count errors and report back
	return nil
}

// FIXME write a version=0 property as an indication of what to expect from repo content.
func (r *Repo) writeProperties(objname, hashspec, name string, tags map[string]map[string]struct{}) error {
	var b = []byte("version=0\n" + PROP_HASH + "=" + hashspec + "\n" + PROP_NAME + "=" + name + "\n")
	for group, g := range tags {
		b = append(b, []byte("tags."+group+"=")...)
		t := maps_.ExtractKeys(g)
		slices.Sort(t)
		b = append(b, []byte(strings.Join(t, ","))...)
		b = append(b, '\n')
	}
	return os.WriteFile(r.repofilepath(objname)+SUFFIX_PROPERTIES, b, 0600)
}

type RepoObj struct {
	Id    string
	Props map[string]string
	Tags  map[string]map[string]struct{}
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
	if err := r.writeProperties(checksumhex, "blake2b:"+checksumhex, name, map[string]map[string]struct{}{}); err != nil {
		return RepoObj{}, errors.Context(err, "failed to write properties-file")
	}
	// TODO get RepoObj instance from writeProperties, instead of going through Open?
	log.Traceln("Completed acquisition. (object: " + checksumhex + ")")
	return r.Open(checksumhex)
}

// Save saves updated repo-object properties to the repository.
func (r *Repo) Save(obj RepoObj) error {
	// FIXME update symlinks?
	return r.writeProperties(obj.Id, obj.Props[PROP_HASH], obj.Props[PROP_NAME], obj.Tags)
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
		if strings.HasPrefix(p[0], "tags.") {
			continue
		}
		propmap[p[0]] = p[1]
	}
	tags := map[string]map[string]struct{}{}
	for cat := range r.cats {
		tags[cat] = map[string]struct{}{}
	}
	for _, p := range props {
		if !strings.HasPrefix(p[0], "tags.") {
			continue
		}
		category := strings.TrimPrefix(p[0], "tags.")
		if _, ok := tags[category]; !ok {
			// FIXME for now skip categories that we don't acknowledge in the repository, i.e. only known in tags-property.
			continue
		}
		group := make(map[string]struct{})
		for _, t := range strings.Split(p[1], ",") {
			set.Insert(group, strings.TrimSpace(t))
		}
		tags[category] = group
	}
	log.Traceln("Current tags:", tags)
	// TODO check allowed properties? (permit unknown properties?, as forward-compatibility?)
	return RepoObj{Id: objname, Props: propmap, Tags: tags}, nil
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
