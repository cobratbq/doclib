// SPDX-License-Identifier: GPL-3.0-only

package repo

import (
	"slices"
	"strings"

	"github.com/cobratbq/goutils/std/builtin"
	slices_ "github.com/cobratbq/goutils/std/builtin/slices"
)

func objIdEquals(id string) func(o RepoObj) bool {
	return func(o RepoObj) bool {
		return o.Id == id
	}
}

func IndexObjectByID(collection []RepoObj, id string) int {
	return slices_.IndexFunc(collection, objIdEquals(id))
}

func objNameCompare(a, b RepoObj) int {
	return strings.Compare(a.Name, b.Name)
}

func ExtractRepoObjectsSorted(docrepo *Repo) []RepoObj {
	objects := builtin.Expect(docrepo.List())
	slices.SortFunc(objects, objNameCompare)
	return objects
}
