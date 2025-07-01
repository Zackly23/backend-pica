package utils

import (
	"mime/multipart"
	"os"
	"io"
)

// saveMultipartFile saves the uploaded file to the specified destination path.
func SaveMultipartFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := CreateFile(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// createFile is a helper to create a file at the given path.
func CreateFile(path string) (*os.File, error) {
	return os.Create(path)
}