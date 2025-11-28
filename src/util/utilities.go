package util

import "os"

func GenerateFile(file_name string, file_size int) (string, error) {
	// Generate the file first
	var filepath string  = ""
	var err error
	filepath, err = os.Getwd()
	println(os.Getwd())
	if err != nil {
		return "", err
	}
	filepath = filepath + "/" + file_name

	err = os.WriteFile(filepath, make([]byte, file_size), os.FileMode(0o777))
	if err != nil {
		return "", err
	}
	return filepath, nil
}