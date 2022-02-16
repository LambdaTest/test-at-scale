package fileutils

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func removeCopiedPath(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
	return err
}
func TestCopyFile(t *testing.T) {
	checkOpenErr := func(t *testing.T) {
		src := "../../testUtils/file"
		dst := "./dst"
		err := CopyFile(src, dst, true)
		if errors.Is(err, os.ErrNotExist) == false {
			t.Errorf("Received: %v", err)
		}
	}

	checkCreateErr := func(t *testing.T) {
		src := "../../testUtils/testfile"
		dst := "../xyz/dst"
		err := CopyFile(src, dst, true)
		if errors.Is(err, os.ErrNotExist) == false {
			t.Errorf("Received: %v", err)
		}
	}

	checkFalseChangeMode := func(t *testing.T) {
		src := "../../testUtils/testfile"
		dst := "./dst"
		err := CopyFile(src, dst, false)

		defer removeCopiedPath(dst)

		if err != nil {
			t.Errorf("Received: %v", err)
		}
	}

	checkComplete := func(t *testing.T) {
		src := "../../testUtils/testfile"
		dst := "./copyfile"
		err := CopyFile(src, dst, true)

		defer removeCopiedPath(dst)

		if err != nil {
			t.Errorf("Received: %v", err)
		}
	}

	t.Run("CheckCopyFile for non existing file", func(t *testing.T) {
		checkOpenErr(t)
	})
	t.Run("CheckCopyFile for non destination", func(t *testing.T) {
		checkCreateErr(t)
	})
	t.Run("CheckCopyFile for false changeMode", func(t *testing.T) {
		checkFalseChangeMode(t)
	})
	t.Run("CheckCopyFile for true changeMode, correct src, dst", func(t *testing.T) {
		checkComplete(t)
	})

}

func TestCopyDir(t *testing.T) {
	checkStatErr := func(t *testing.T) {
		src := "../../testUtils/doesNotExist"
		dst := "./dst"
		err := CopyDir(src, dst, true)
		if errors.Is(err, os.ErrNotExist) == false {
			t.Errorf("Received: %v", err)
		}
	}

	checkSrcIsDir := func(t *testing.T) {
		src := "../../testUtils/testfile"
		dst := "../xyz/dst"
		err := CopyDir(src, dst, true)
		want := "source is not a directory"
		if err != nil && err.Error() != want {
			t.Errorf("Received: %v, want: %v", err, want)
		}
	}

	checkDstexist := func(t *testing.T) {
		src := "../../testUtils/testdirectory"
		dst := "./xyz/dst"
		err := CopyDir(src, dst, true)

		defer removeCopiedPath("./xyz")

		if err != nil {
			t.Errorf("Received: %v", err)
		}
	}

	checkExistingDst := func(t *testing.T) {
		src := "../../testUtils/testdirectory"
		dst := src
		err := CopyDir(src, dst, true)
		want := "destination already exists"

		if err != nil && err.Error() != want {
			t.Errorf("Received: %v, want: %v", err, want)
		}
	}

	t.Run("CheckCopyDir for non existing src dir", func(t *testing.T) {
		checkStatErr(t)
	})
	t.Run("CheckCopyDir for if src is actually a dir", func(t *testing.T) {
		checkSrcIsDir(t)
	})
	t.Run("CheckCopyDir for non existing dst directory path, complete test", func(t *testing.T) {
		checkDstexist(t)
	})
	t.Run("CheckCopyDir for already existing dst path", func(t *testing.T) {
		checkExistingDst(t)
	})
}

func TestCheckIfExists(t *testing.T) {

	checkFalsePath := func(t *testing.T) {
		path := "../pathnotexist/dir"
		b, err := CheckIfExists(path)
		if err != nil || b == true {
			t.Errorf("Received: %v", err)
		}
	}

	checkTruePath := func(t *testing.T) {
		path := "../../testUtils/"
		b, err := CheckIfExists(path)
		if err != nil || b != true {
			t.Errorf("Received: %v", err)
		}
	}

	t.Run("CheckIfExists for non existing path", func(t *testing.T) {
		checkFalsePath(t)
	})
	t.Run("CheckIfExists for existing path", func(t *testing.T) {
		checkTruePath(t)
	})
}

func TestCreateIfNotExists(t *testing.T) {

	checkFalsePathDir := func(t *testing.T, isDir bool) {
		path := "../pathnotexist"
		err := CreateIfNotExists(path, isDir)

		defer removeCopiedPath(path)

		if isDir && err != nil {
			t.Errorf("Error in making dir: %v", err)
		}

	}

	checkForMkdirErr := func(t *testing.T, isDir bool) {
		path := "pathnotexist"
		err := CreateIfNotExists(path, isDir)

		defer removeCopiedPath(path)

		if err != nil {
			t.Errorf("Error: %v", err)
		}
	}

	t.Run("CheckIfNotExists for non existing path, isDir = true", func(t *testing.T) {
		checkFalsePathDir(t, true)
	})
	t.Run("CheckIfNotExists for non existing path, isDir = false", func(t *testing.T) {
		checkForMkdirErr(t, false)
	})
}
