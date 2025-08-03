# DocLib

> A directory-based object-library.

DocLib stores objects, e.g. documents, in a repository-directory and creates symlinks as needed for purpose of tagging under any number of categories and tags.

Currently there are two predefined directories `repo` and `titles`, which contain binary content and symlinks by name to every document, respectively. Any other directories are treated as tag-categories with their subdirectories representing individual tags. The `<checksum>.properties` files contain properties for the binary objects. The checking process functions to (re)populate the various tags-directories with symlinks to the binary objects in the repository. Categories and tags are stored in sanitized format, allowing for arbitrary capitalization on the file-system.
