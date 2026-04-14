package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/twinbird/mkdmy/internal/cli"
)

func Generate(opts cli.Options) error {
	outputDir, err := prepareOutputDir(opts.OutputDir)
	if err != nil {
		return err
	}

	switch opts.Type {
	case cli.KindDirectory:
		for index := 1; index <= opts.Count; index++ {
			name := renderName(opts.Name, index)
			path := filepath.Join(outputDir, filepath.Clean(name))
			if err := createDirectory(path); err != nil {
				return fmt.Errorf("generate dir %q: %w", path, err)
			}
		}
	case cli.KindText, cli.KindPNG:
		if err := generateFilesParallel(outputDir, opts); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type %q", opts.Type)
	}

	return nil
}

func prepareOutputDir(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if err := os.MkdirAll(cleaned, 0o755); err != nil {
		return "", fmt.Errorf("prepare output dir %q: %w", cleaned, err)
	}
	return cleaned, nil
}

type generationJob struct {
	index int
	path  string
	file  *os.File
}

type generationError struct {
	index int
	err   error
}

func generateFilesParallel(outputDir string, opts cli.Options) error {
	workerCount := parallelismForCount(opts.Count)
	jobs := make(chan generationJob)

	var (
		wg       sync.WaitGroup
		errMu    sync.Mutex
		firstErr *generationError
	)

	recordErr := func(index int, err error) {
		errMu.Lock()
		defer errMu.Unlock()
		if firstErr == nil || index < firstErr.index {
			firstErr = &generationError{index: index, err: err}
		}
	}

	worker := func() {
		defer wg.Done()

		for job := range jobs {
			var err error
			switch opts.Type {
			case cli.KindText:
				err = writeTextFile(job.file, job.path, opts, job.index)
				if err != nil {
					err = fmt.Errorf("generate text %q: %w", job.path, err)
				}
			case cli.KindPNG:
				err = writeImageFile(job.file, opts, job.index)
				if err != nil {
					err = fmt.Errorf("generate png %q: %w", job.path, err)
				}
			default:
				err = fmt.Errorf("unsupported type %q", opts.Type)
			}

			if err != nil {
				recordErr(job.index, err)
			}
		}
	}

	for range workerCount {
		wg.Add(1)
		go worker()
	}

	for index := 1; index <= opts.Count; index++ {
		name := renderName(opts.Name, index)
		path := filepath.Join(outputDir, filepath.Clean(name))

		file, err := reserveOutputFile(path)
		if err != nil {
			close(jobs)
			wg.Wait()
			return wrapGenerationError(opts.Type, path, err)
		}

		jobs <- generationJob{
			index: index,
			path:  path,
			file:  file,
		}
	}

	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return firstErr.err
	}

	return nil
}

func parallelismForCount(count int) int {
	if count < 1 {
		return 1
	}

	parallelism := runtime.GOMAXPROCS(0)
	if parallelism < 2 {
		parallelism = 2
	}
	if parallelism > count {
		return count
	}
	return parallelism
}

func wrapGenerationError(kind cli.FileKind, path string, err error) error {
	switch kind {
	case cli.KindText:
		return fmt.Errorf("generate text %q: %w", path, err)
	case cli.KindPNG:
		return fmt.Errorf("generate png %q: %w", path, err)
	case cli.KindDirectory:
		return fmt.Errorf("generate dir %q: %w", path, err)
	default:
		return err
	}
}
