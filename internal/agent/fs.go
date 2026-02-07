package agent

import "io/fs"

func fsSub(fsys fs.FS, dir string) (fs.FS, error) {
	s, err := fs.Sub(fsys, dir)
	if err != nil {
		return fsys, nil
	}
	return s, nil
}
