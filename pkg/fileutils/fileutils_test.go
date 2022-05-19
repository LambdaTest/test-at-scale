package fileutils

import (
	"fmt"
	"os"
	"testing"
)

func removeCopiedPath(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
}

// nolint:dupl
func TestCopyFile(t *testing.T) {
	type args struct {
		src        string
		dst        string
		changeMode bool
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		requireDelete bool // if new file is created, we need to delete for clean up
	}{
		{
			"Check open error",
			args{src: "../../testutils/file", dst: "./dst", changeMode: true},
			true,
			false,
		}, // this file is not present

		{
			"Check create error for invalid path",
			args{src: "../../testutils/testfile", dst: "../xyz/dst", changeMode: true},
			true,
			false,
		}, // file present at given args.src

		{
			"Check fasle change mode",
			args{src: "../../testutils/testfile", dst: "./dst", changeMode: true},
			false,
			true,
		}, // new file will be created, delete it

		{
			"Check success",
			args{src: "../../testutils/testfile", dst: "./copyfile", changeMode: true},
			false,
			true,
		}, // new file will be created, delete it
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyFile(tt.args.src, tt.args.dst, tt.args.changeMode)

			if tt.requireDelete {
				defer removeCopiedPath(tt.args.dst)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("CopyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// nolint:dupl
func TestCopyDir(t *testing.T) {
	type args struct {
		src        string
		dst        string
		changeMode bool
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		requireDelete bool // if new path/directory is created, we need to delete it for clean up
	}{
		{
			"Check status error",
			args{src: "../../testutils/dne/file", dst: "./dst", changeMode: true},
			true,
			false,
		}, // this dir is not present

		{
			"Check for src is not a directory",
			args{src: "../../testutils/testfile", dst: "../xyz/dst", changeMode: true},
			true,
			false,
		}, // file present at given args.src

		{
			"Check for non-exist dst directory",
			args{src: "../../testutils/testdirectory", dst: "./xyz", changeMode: true},
			false,
			true,
		}, // new dir will be created

		{
			"Check existing dst",
			args{src: "../../testutils/testdirectory", dst: "../../testutils/testdirectory", changeMode: true},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CopyDir(tt.args.src, tt.args.dst, tt.args.changeMode)

			if tt.requireDelete {
				defer removeCopiedPath(tt.args.dst)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("CopyDir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckIfExists(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"Check false path error",
			args{path: "../pathnotexist/dir"},
			false,
			false,
		}, // this dir is not present

		{
			"Check for existing path, should not give error",
			args{path: "../../testutils/"},
			true,
			false,
		}, // this dir is present
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckIfExists(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckIfExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckIfExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateIfNotExists(t *testing.T) {
	type args struct {
		path  string
		isDir bool
	}
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		requireDelete bool
	}{
		{
			"Check false path directory",
			args{path: "../pathnotexist", isDir: true},
			false,
			true,
		}, // new dir will be created

		{
			"Check make directory error",
			args{path: "pathnotexist", isDir: true},
			false,
			true,
		}, // new dir will be created
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateIfNotExists(tt.args.path, tt.args.isDir)

			if tt.requireDelete {
				defer removeCopiedPath(tt.args.path)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIfNotExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
