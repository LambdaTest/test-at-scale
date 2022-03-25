package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func removeCreatedFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Println("error in removing!!")
	}
}

func TestMin(t *testing.T) {
	type args struct {
		x int
		y int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"x: 5, y: -1", args{5, -1}, -1},
		{"x: 0, y: 0", args{0, 0}, 0},
		{"x: -293836, y: 0", args{-293836, 0}, -293836},
		{"x: 2545, y: 374", args{2545, 374}, 374},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Min(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("Min() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	_, err := os.Create("dummy_file")
	if err != nil {
		fmt.Printf("Error in creating file, error: %v", err)
	}
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"dummy_file_name", args{"dummy_file_name"}, "", true},
		{"dummy_file", args{"dummy_file"}, "d41d8cd98f00b204e9800998ecf8427e", false},
	}
	defer removeCreatedFile("dummy_file")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeChecksum(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComputeChecksum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ComputeChecksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateDirectory(t *testing.T) {
	newDir := "../../testutils/nonExistingDir"
	existDir := "../../testutils/testdirectory"
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"Existing directory: ../../testutils/testdirecotry", args{existDir}, false},
		{"Non-existing directory: ../../testutils/nonExistingDir", args{newDir}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateDirectory(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("CreateDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.args.path == newDir {
				if _, err := os.Stat(newDir); err != nil {
					t.Errorf("Directory did not exist, error: %v", err)
					return
				}
				defer removeCreatedFile(newDir)
			}
		})
	}
}

func TestWriteFileToDirectory(t *testing.T) {
	path := "../../testutils/testdirectory"
	filename := "writeFileToDirectory"
	data := []byte("Hello world!")
	err := WriteFileToDirectory(path, filename, data)
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	defer removeCreatedFile(filepath.Join(path, filename))
	checkData, err := ioutil.ReadFile(filepath.Join(path, filename))
	if err != nil {
		t.Errorf("Error: %v", err)
		return
	}
	if string(checkData) != "Hello world!" {
		t.Errorf("expected file contents: Hello world!, got: %s", string(checkData))
	}
}

func TestGetOutboundIP(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Test1", "http://synapse:8000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOutboundIP(); got != tt.want {
				t.Errorf("GetOutboundIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
