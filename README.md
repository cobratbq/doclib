# DocLib

> A directory-based object-library.

DocLib stores objects, e.g. documents, in a repository-directory and creates symlinks as needed for purpose of tagging under any number of categories and tags.

## How does it work?

Currently there are two predefined directories `repo` and `titles`, which contain immutable (read-only) binary content and symlinks by name to every document, respectively. Any other directories are treated as categories with subdirectories for individual tags. The `repo/<checksum>.properties` files contain properties for the binary objects. Directories on the file-system define which categories and tags are available.

The checking process (re)populates the various tag-directories with symlinks to the binary objects in the repository, and general content checking. Categories and tags are stored in sanitized format, allowing for arbitrary capitalization, adaptable to preference, on the file-system and in the management UI.

_DocLib_ provides a basic management interface for managing objects, while expecting access and regular use via the symlinks available on the file-system. Consequently, repositories can be maintained in a git-repository without too much effort.

## Technical

- Checksums are calculated using [BLAKE2b](<https://www.blake2.net/> "BLAKE2 -- fast secure hashing")
- Directories and sub-directories contain symlinks for access through a variety of perspectives.
