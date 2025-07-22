package repo

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

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

const VERSION = "0"
const SUBDIR_REPO = "repo"
const PREFIX_REPO_TEMP = "temp--"
const SUFFIX_PROPERTIES = ".properties"
const PROP_VERSION = "version"
const PROP_HASH = "hash"
const PROP_HASHSPEC_PREFIX = "blake2b:"
const PROP_NAME = "name"
const PROP_PREFIX_TAGS = "tags."
const SECTION_TITLES = "titles"

func Hash(location string) ([64]byte, error) {
	if hash, err := hash_.HashFile(builtin.Expect(blake2b.New512(nil)), location); err == nil {
		return [64]byte(hash), nil
	} else {
		return [64]byte{}, errors.Context(err, "hashing content at '"+location+"'")
	}
}

func isStandardDir(name string) bool {
	return name == SUBDIR_REPO || name == SECTION_TITLES
}

type Tag struct {
	Key   string
	Title string
}

type Repo struct {
	location string
	cats     map[string][]Tag
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
func listOptions(location string) ([]Tag, error) {
	if entries, err := os.ReadDir(location); err != nil {
		return nil, errors.Context(err, "directory missing for tag-group: "+location)
	} else {
		index := []Tag{}
		for _, e := range entries {
			index = append(index, Tag{Key: strings.ToLower(e.Name()), Title: e.Name()})
		}
		return index, nil
	}
}

// FIXME does not check/create subdirs 'repo' and 'titles'
func OpenRepo(location string) (Repo, error) {
	index := map[string][]Tag{}
	entries, err := os.ReadDir(location)
	if err != nil {
		return Repo{}, errors.Context(err, "open repository root-directory")
	}
	for _, e := range entries {
		if !e.IsDir() || isStandardDir(e.Name()) {
			continue
		}
		index[strings.ToLower(e.Name())] = builtin.Expect(listOptions(filepath.Join(location, e.Name())))
	}
	log.Traceln("Category-index:", index)
	return Repo{location: location, cats: index}, nil
}

func (r *Repo) Categories() []string {
	keys := maps.ExtractKeys(r.cats)
	slices.Sort(keys)
	return keys
}

// Tags returns list of Tags (lower-cased identifier and a title, unchanged from the file system representation).
func (r *Repo) Tags(category string) []Tag {
	if index, ok := r.cats[category]; !ok {
		return nil
	} else {
		return index
	}
}

func (r *Repo) checkTagsForObject(id, name string, tagCats map[string]map[string]struct{}) error {
	log.Traceln("Repo-object tags:", tagCats)
	entries, err := os.ReadDir(r.location)
	if err != nil {
		return errors.Context(err, "failed to open root repository directory for tags processing")
	}
	for _, e := range entries {
		if !e.IsDir() || isStandardDir(e.Name()) {
			continue
		}
		// TODO we don't currently check if tag-names are used for which there exists no directory
		tags, ok := tagCats[strings.ToLower(e.Name())]
		if !ok {
			continue
		}
		tagdirs, err := os.ReadDir(filepath.Join(r.location, e.Name()))
		if err != nil {
			log.Warnln("Failed to open tag-directory for tag-group", e.Name())
			continue
		}
		for _, t := range tagdirs {
			if !t.IsDir() {
				continue
			}
			path := filepath.Join(r.location, e.Name(), t.Name(), name)
			linkExists := os_.ExistsIsSymlink(path)
			_, tagExists := tags[strings.ToLower(t.Name())]
			switch {
			case !tagExists && linkExists:
				if err := os.Remove(path); err == nil {
					log.Debugln("Removed symlink for untagged object at:", path)
				} else {
					log.Warnln("Failed to remove unexpected symlink", path, err.Error())
				}
			case tagExists && !linkExists:
				if err := os.Symlink(filepath.Join("..", "..", SUBDIR_REPO, id), path); err == nil {
					log.Debugln("Created symlink for tagged object at:", path)
				} else {
					log.Warnln("Failed to create missing symlink", path, err.Error())
				}
			}
		}
	}
	return nil
}

func (r *Repo) checkBadTags() error {
	entries, err := os.ReadDir(r.location)
	if err != nil {
		return errors.Context(err, "failed to open repository root-directory for tags processing")
	}
	for _, e := range entries {
		if !e.IsDir() || isStandardDir(e.Name()) {
			continue
		}
		log.Traceln("Processing tag-category '" + e.Name() + "'…")
		tagdirs, err := os.ReadDir(filepath.Join(r.location, e.Name()))
		if err != nil {
			log.Warnln("Failed to open tag-directory for tag-group", e.Name())
			continue
		}
		for _, t := range tagdirs {
			if !t.IsDir() {
				continue
			}
			log.Traceln("Processing tag '" + t.Name() + "' in category '" + e.Name() + "'…")
			links, err := os.ReadDir(filepath.Join(r.location, e.Name(), t.Name()))
			if err != nil {
				log.Warnln("Failed to read files in tag-directory:", err.Error())
				continue
			}
			for _, link := range links {
				log.Traceln("Processing symlink '" + link.Name() + "'…")
				linkpath := filepath.Join(r.location, e.Name(), t.Name(), link.Name())
				if _, err := os.Stat(linkpath); err == nil {
					var repoobjpath string
					if repoobjpath, err = os.Readlink(linkpath); err != nil {
						log.Warnln("Failed to read repo-object path from symlink:", err.Error())
						continue
					}
					var repoobj RepoObj
					if repoobj, err = r.Open(filepath.Base(repoobjpath)); err != nil {
						log.Warnln("Failed to open repo-object:", err.Error())
						continue
					}
					if link.Name() != repoobj.Name {
						// Symlink with (most likely) outdated name. Can be removed, as we would already
						// (re)create the missing symlink if we wouldn't find it at the expected name.
						if err = os.Remove(linkpath); err == nil {
							log.Debugln("Removed symlink with incorrect name at:", linkpath)
						} else {
							log.Warnln("Failed to remove symlink with incorrect name at:", linkpath)
						}
					}
				} else {
					// Remove broken symlink.
					if err := os.Remove(linkpath); err == nil {
						log.Debugln("Removed broken symlink at:", linkpath)
					} else {
						log.Warnln("Failed to remove broken symlink at:", linkpath)
					}
				}
			}
		}
	}
	return nil
}

func (r *Repo) Check() error {
	var entries []os.DirEntry
	var err error

	log.Infoln("Starting repository check…")
	defer log.Infoln("Finished repository check.")

	if entries, err = os.ReadDir(r.repofilepath("")); err != nil {
		return errors.Context(err, "failed to open object-repository directory")
	}
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
					log.Debugln("Removed orphaned properties-file:", e.Name())
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
				log.Debugln("Removed temporary file:", e.Name())
			}
			continue
		}
		// Comparing file content checksum with binary-object name.
		if checksum, err := hash_.HashFile(builtin.Expect(blake2b.New512(nil)), r.repofilepath(e.Name())); err != nil {
			log.Warnln("Failed to hash repo-object:", hex.EncodeToString(checksum))
		} else if e.Name() != hex.EncodeToString(checksum) {
			log.Warnln("Repo-object '" + e.Name() + "': checksum does not match. Possible corruption. (checksum: " + hex.EncodeToString(checksum) + ")")
		}
		// Checking characteristics of file properties.
		if info, err := os.Stat(r.repofilepath(e.Name() + SUFFIX_PROPERTIES)); err != nil {
			log.Warnln(e.Name()+SUFFIX_PROPERTIES, ": properties-file is missing.")
		} else if info.Mode()&os.ModeType != 0 {
			log.Warnln(e.Name()+SUFFIX_PROPERTIES, ": properties-file is not a regular file.")
		} else if o, err := r.Open(e.Name()); err != nil {
			log.Warnln(e.Name(), ": failed to parse properties: ", err.Error())
		} else {
			if o.Id != e.Name() {
				log.Warnln(e.Name(), ": invalid properties", e.Name(), o.Id)
			}
			titlepath := filepath.Join(r.location, SECTION_TITLES, o.Name)
			if info, err := os.Lstat(titlepath); err != nil {
				// Create symlink when one does not exist under the correct name as stated in the properties.
				// Next we will remove symlinks that refer to repo-objects that have a different name-prop.
				if err := os.Symlink(filepath.Join("..", SUBDIR_REPO, e.Name()), titlepath); err != nil {
					log.Warnln(e.Name(), ": failed to create symlink at:", titlepath, err.Error())
				} else {
					log.Debugln(e.Name(), ": missing symlink in document titles recreated at:", titlepath)
				}
			} else if info.Mode()&os.ModeSymlink == 0 {
				log.Warnln(info.Name(), ": a foreign file-system object was found where a symlink to a repo-object was expected.")
			} else if targetpath, err := os.Readlink(titlepath); err == nil && filepath.Base(targetpath) != e.Name() {
				log.Warnln("Symlink does not point to expected repo-object. Duplicate names are in use:", targetpath)
			}
			// Verify symlinks for tags that are expected for this specific object.
			if err := r.checkTagsForObject(o.Id, o.Name, o.Tags); err != nil {
				log.Warnln("Failure during tags processing:", err.Error())
			}
		}
	}

	if entries, err = os.ReadDir(filepath.Join(r.location, SECTION_TITLES)); err != nil {
		return errors.Context(err, "failed to open directory with titles links")
	}
	for _, e := range entries {
		log.Traceln("Processing titles-entry…", e.Name())
		path := filepath.Join(r.location, SECTION_TITLES, e.Name())
		if targetpath, err := os.Readlink(path); err != nil {
			log.Warnln(e.Name(), ": failed to query symlink without error:", err.Error())
		} else if obj, err := r.Open(filepath.Base(targetpath)); err != nil {
			// TODO should I be checking that linkpath has characteristics of repo-object before drawing conclusions?
			log.Traceln("titles symlink does not correctly link to repo-object. Deleting…")
			if err := os.Remove(path); err != nil {
				log.Warnln("Failed to delete broken symlink in titles:", err.Error())
				continue
			}
			log.Debugln("Broken symlink in titles successfully removed.", path)
		} else if obj.Name != e.Name() {
			log.Traceln("Titles document name does not match with 'name' property. Removing…")
			// Previously, we created symlinks when they don't exist at expected name. Now we remove existing
			// symlinks which refer to repo-objects with a different name.
			if err := os.Remove(path); err != nil {
				log.Warnln(e.Name(), ": failed to rename object to proper name:", err.Error())
				continue
			}
			log.Debugln("Titles symlink does not have the correct document name. Removed.")
		}
	}

	// FIXME how to handle result from checkBadTags, do we even care?
	if err := r.checkBadTags(); err != nil {
		log.Warnln("Result of checkBrokenTags:", err.Error())
	}

	// TODO count errors and report back
	return nil
}

func (r *Repo) writeProperties(objname, name string, tags map[string]map[string]struct{}) error {
	var buffer = []byte(PROP_VERSION + "=" + VERSION + "\n" + PROP_HASH + "=" + PROP_HASHSPEC_PREFIX + objname + "\n" + PROP_NAME + "=" + name + "\n")
	for group, g := range tags {
		if len(g) == 0 {
			continue
		}
		t := maps_.ExtractKeys(g)
		slices.Sort(t)
		buffer = append(buffer, []byte(PROP_PREFIX_TAGS+group+"="+strings.Join(t, ",")+"\n")...)
	}
	return os.WriteFile(r.repofilepath(objname)+SUFFIX_PROPERTIES, buffer, 0600)
}

type RepoObj struct {
	Id    string
	Name  string
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
	if err := r.writeProperties(checksumhex, name, map[string]map[string]struct{}{}); err != nil {
		return RepoObj{}, errors.Context(err, "failed to write properties-file")
	}
	// TODO get RepoObj instance from writeProperties, instead of going through Open?
	log.Traceln("Completed acquisition. (object: " + checksumhex + ")")
	return r.Open(checksumhex)
}

func (r *Repo) Delete(id string) error {
	path := r.repofilepath(id)
	if err := os.Remove(path); err != nil {
		return errors.Context(err, "Delete repository-object "+id)
	}
	if err := os.Remove(path + SUFFIX_PROPERTIES); err != nil {
		log.Warnln("Failed to delete repo-object properties. Next check, orphaned properties-file will again be deleted.")
	}
	// TODO not yet deleting any symlinks, etc. These will already be deleted during check anyways.
	return nil
}

// Save saves updated repo-object properties to the repository.
func (r *Repo) Save(obj RepoObj) error {
	// FIXME update symlinks?
	return r.writeProperties(obj.Id, obj.Name, obj.Tags)
}

func (r *Repo) Open(objname string) (RepoObj, error) {
	propspath := r.repofilepath(objname + SUFFIX_PROPERTIES)
	props, err := bufio_.OpenFileProcessStringLinesFunc(propspath, '\n', func(s string) ([2]string, error) {
		// TODO fine-tuning trimming whitespace for comment-line matching
		if len(s) == 0 || strings_.AnyPrefix(strings.TrimLeft(s, " \t"), "#", "!") {
			return [2]string{}, bufio_.ErrProcessingIgnore
		}
		// TODO support ':' separator?
		if key, value, ok := strings.Cut(s, "="); ok {
			return [...]string{strings.TrimSpace(key), strings.TrimSpace(value)}, nil
		}
		return [2]string{}, errors.ErrIllegal
	})
	if err != nil {
		return RepoObj{}, errors.Context(err, "failed to parse properties for "+objname)
	}
	var obj RepoObj
	// TODO check allowed properties? (permit unknown properties?, as forward-compatibility?)
	for _, p := range props {
		if strings.HasPrefix(p[0], PROP_PREFIX_TAGS) {
			// Process arbitrary tag-categories later.
			continue
		}
		switch p[0] {
		case PROP_VERSION:
			if p[1] != VERSION {
				return RepoObj{}, errors.Context(errors.ErrIllegal, "version of properties is not supported")
			}
		case PROP_HASH:
			if !strings.HasPrefix(p[1], PROP_HASHSPEC_PREFIX) {
				return RepoObj{}, errors.Context(errors.ErrIllegal, "hashspec must contain prefix for hash function")
			}
			obj.Id = strings.TrimPrefix(p[1], PROP_HASHSPEC_PREFIX)
		case PROP_NAME:
			obj.Name = p[1]
		default:
			// FIXME handle unknown property (should not panic, incomplete implementation)
			panic("To be implemented")
		}
	}
	obj.Tags = map[string]map[string]struct{}{}
	for cat := range r.cats {
		obj.Tags[cat] = map[string]struct{}{}
	}
	for _, p := range props {
		if !strings.HasPrefix(p[0], PROP_PREFIX_TAGS) {
			continue
		}
		cat := strings.TrimPrefix(p[0], PROP_PREFIX_TAGS)
		if _, ok := obj.Tags[cat]; !ok {
			// Primarily, preserve existing tag-properties even if not currently in use.
			obj.Tags[cat] = map[string]struct{}{}
		}
		for _, t := range strings.Split(p[1], ",") {
			if len(t) == 0 {
				continue
			}
			set.Insert(obj.Tags[cat], strings.TrimSpace(t))
		}
	}
	log.Traceln("Current tags:", obj.Tags)
	return obj, nil
}

// TODO could use caching in case the repository has not changed. (Is this really possible if we also expect to read some values from the file system structure?)
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
