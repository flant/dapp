package file_reader

import (
	"context"
	"path/filepath"
)

func (r FileReader) configurationFilesGlob(ctx context.Context, pattern string, isFileAcceptedFunc func(relPath string) (bool, error), readCommitFileFunc func(ctx context.Context, relPath string) ([]byte, error), handleFileFunc func(relPath string, data []byte, err error) error, uncommittedFileErrorFunc func(relPath string) error) error {
	processedFiles := map[string]bool{}

	isFileProcessedFunc := func(relPath string) bool {
		return processedFiles[filepath.ToSlash(relPath)]
	}

	readFileBeforeHookFunc := func(relPath string) {
		processedFiles[filepath.ToSlash(relPath)] = true
	}

	readFileFunc := func(relPath string) ([]byte, error) {
		readFileBeforeHookFunc(relPath)
		return r.readFile(relPath)
	}

	readCommitFileWrapperFunc := func(relPath string) ([]byte, error) {
		readFileBeforeHookFunc(relPath)
		return readCommitFileFunc(ctx, relPath)
	}

	fileRelPathListFromFS, err := r.filesGlob(pattern)
	if err != nil {
		return err
	}

	if r.manager.LooseGiterminism() {
		for _, relPath := range fileRelPathListFromFS {
			data, err := readFileFunc(relPath)
			if err := handleFileFunc(relPath, data, err); err != nil {
				return err
			}
		}

		return nil
	}

	fileRelPathListFromCommit, err := r.commitFilesGlob(ctx, pattern)
	if err != nil {
		return err
	}

	for _, relPath := range fileRelPathListFromCommit {
		if accepted, err := isFileAcceptedFunc(relPath); err != nil {
			return err
		} else if accepted {
			continue
		}

		data, err := readCommitFileWrapperFunc(relPath)
		if err := handleFileFunc(relPath, data, err); err != nil {
			return err
		}
	}

	for _, relPath := range fileRelPathListFromFS {
		accepted, err := isFileAcceptedFunc(relPath)
		if err != nil {
			return err
		}

		if !accepted {
			if !isFileProcessedFunc(relPath) {
				return uncommittedFileErrorFunc(relPath)
			}

			continue
		}

		data, err := readFileFunc(relPath)
		if err := handleFileFunc(relPath, data, err); err != nil {
			return err
		}
	}

	return nil
}