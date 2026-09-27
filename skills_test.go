package tblfmt

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestSkillsAreCopies makes sure that each skill in skills-lock.json is an
// ordinary folder under .agents/skills and under .claude/skills, and that the
// two folders hold the same files.
//
// A symbolic link fails the test. A Windows checkout writes a symbolic link as
// a text file that holds the target path. Claude Code then finds a file where
// it expects a folder, and it loads no skill and reports nothing.
func TestSkillsAreCopies(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("skills-lock.json")
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Skills map[string]jsontext.Value `json:"skills"`
	}
	if err := json.Unmarshal(body, &lock); err != nil {
		t.Fatalf("reading skills-lock.json: %v", err)
	}
	if len(lock.Skills) == 0 {
		t.Fatal("skills-lock.json names no skill")
	}
	roots := []string{filepath.Join(".agents", "skills"), filepath.Join(".claude", "skills")}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if _, ok := lock.Skills[e.Name()]; !ok {
				t.Errorf("%s is not in skills-lock.json, so nobody can install it again. "+
					"Add it with the command in CONTRIBUTING.md", filepath.Join(root, e.Name()))
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(lock.Skills)) {
		agents := skillFiles(t, filepath.Join(roots[0], name))
		claude := skillFiles(t, filepath.Join(roots[1], name))
		if agents == nil || claude == nil {
			continue
		}
		for _, path := range slices.Sorted(maps.Keys(agents)) {
			switch other, ok := claude[path]; {
			case !ok:
				t.Errorf("%s: %s is in %s and not in %s", name, path, roots[0], roots[1])
			case other != agents[path]:
				t.Errorf("%s: %s differs between %s and %s", name, path, roots[0], roots[1])
			}
		}
		for _, path := range slices.Sorted(maps.Keys(claude)) {
			if _, ok := agents[path]; !ok {
				t.Errorf("%s: %s is in %s and not in %s", name, path, roots[1], roots[0])
			}
		}
	}
}

// skillFiles returns the content of each file in one copy of a skill, keyed by
// its path in the copy. It reports a copy that is missing or that is a
// symbolic link, and then returns nil.
func skillFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	fi, err := os.Lstat(dir)
	switch {
	case err != nil:
		t.Errorf("%s is missing. Install the skill with the command in CONTRIBUTING.md", dir)
		return nil
	case !fi.IsDir():
		t.Errorf("%s is not a folder. A Windows checkout writes a symbolic link as a text file, "+
			"so install the skill with --copy. See CONTRIBUTING.md", dir)
		return nil
	}
	out := map[string]string{}
	err = fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.Type()&fs.ModeSymlink != 0:
			t.Errorf("%s is a symbolic link. Install the skill with --copy", filepath.Join(dir, path))
		case d.Type().IsRegular():
			body, err := fs.ReadFile(os.DirFS(dir), path)
			if err != nil {
				return err
			}
			out[path] = string(body)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	return out
}
