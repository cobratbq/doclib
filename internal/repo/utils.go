package repo

import (
	"slices"
	"strings"

	"github.com/cobratbq/goutils/std/builtin"
)

func IndexObjectByID(collection []RepoObj, id string) int {
	for i, obj := range collection {
		if obj.Id == id {
			return i
		}
	}
	return -1
}

func ExtractRepoObjectsSorted(docrepo *Repo) []RepoObj {
	objects := builtin.Expect(docrepo.List())
	slices.SortFunc(objects, func(a, b RepoObj) int { return strings.Compare(a.Name, b.Name) })
	return objects
}
