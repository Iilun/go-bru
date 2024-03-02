package bru

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodingMultiple(t *testing.T) {
	// Test all files in testFiles folder - from the official bruno repository
	bruFiles, err := filepath.Glob("testFiles/**/*.bru")
	if err != nil {
		t.Fatalf("Could not glob sample bru files: %v", err)
	}
	if len(bruFiles) == 0 {
		t.Fatalf("Could not find any sample bru files")
	}
	for _, fileName := range bruFiles {
		fileContent, err := os.ReadFile(fileName)
		if err != nil {
			t.Fatalf("Could not open bru file '%s': %v", fileName, err)
		}
		// Using the encoding with trailing new line as it is present in the repo,
		// however with bruno v1.10.0 on my machine there are no trailing new lines
		decodeAndEncodeFileWithNewLine(fileContent, t)
	}
}

func decodeAndEncodeFileWithNewLine(file []byte, t *testing.T) {
	read, err := Read(file)
	if err != nil {
		t.Fatal(err.Error())
	}
	encoder := Encoder{addTrailingLineEnd: true}
	encoded, err := encoder.Write(read)
	if err != nil {
		t.Fatal(err.Error())
	}
	if string(encoded) != string(file) {
		t.Log("-----ORIGINAL-----\n" + string(file) + "\n-----END ORIGINAL-----")
		t.Log("-----ENCODED-----\n" + string(encoded) + "\n-----END ENCODED-----")
		t.Fatal("encoded content is different from original file")
	}
}

func decodeAndEncodeFileWithDefault(file []byte, t *testing.T) {
	read, err := Read(file)
	if err != nil {
		t.Fatal(err.Error())
	}
	encoded, err := Write(read)
	if err != nil {
		t.Fatal(err.Error())
	}
	if string(encoded) != string(file) {
		t.Log("-----ORIGINAL-----\n" + string(file) + "\n-----END ORIGINAL-----")
		t.Log("-----ENCODED-----\n" + string(encoded) + "\n-----END ENCODED-----")
		t.Fatal("encoded content is different from original file")
	}
}
