# DocLib

> A directory-based object-library

_DocLib_ stores objects, e.g. documents, in a repository-directory and creates symlinks as needed for purpose of tagging under any number of categories and tags.

## How does it work?

Currently there are two predefined directories `repo` and `titles`, which contain immutable (read-only) binary content and symlinks by name to every document, respectively. Any other directories are treated as categories with sub-directories for individual tags. The `repo/<checksum>.properties` files contain properties for their corresponding binary objects. Directories on the file-system define which categories and tags are available.

The checking process (re)populates the various tag-directories with symlinks to the binary objects in the repository, and does general content checking. Categories and tags are stored in sanitized format, allowing for arbitrary capitalization, adaptable to preference, on the file-system and in the management UI.

_DocLib_ provides a basic management interface for managing objects, while the user is expected to access content via the symlinks available on the file-system. Consequently, repositories can be maintained in a git-repository without too much effort.

## Getting started

Use flag `-repo` to specify an existing, empty directory for use as object-repository. For example `./doclib -repo data/`.

_This application is still in development. Although "checking" is highly specific, it does make changes in the repository-directory. Results may be unpredictable if an arbitrary directory is chosen._

## Technical

- Directories and sub-directories contain symlinks for access to objects from a variety of perspectives.
- Checksums are calculated using [BLAKE2b](<https://www.blake2.net/> "BLAKE2 -- fast secure hashing")
- Build without `tracelog` build-tag, to disable `[trace]` log entries for reduced verbosity.

__note__ The _Check_-process produces output on the commandline to report on issues.

## License

See [LICENSE](<LICENSE>) for the license under which this software is made available.
