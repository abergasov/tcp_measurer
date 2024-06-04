package tcpmeasurer

import (
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"
)

func WithParseFilesInterval(parseDuration time.Duration) Opt {
	return func(s *Service) {
		s.parseFilesInterval = parseDuration
	}
}

func WithFilesPath(path string) Opt {
	return func(s *Service) {
		s.filesPath = path
	}
}

func (s *Service) parsePCAPFiles() {
	ticker := time.NewTicker(s.parseFilesInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkFiles()
		}
	}
}

func (s *Service) checkFiles() {
	dir, err := os.ReadDir(s.filesPath)
	if err != nil {
		s.l.Fatal("failed to read dir", err, slog.String("path", s.filesPath))
	}
	fileNames := make([]string, 0, len(dir))
	for i := range dir {
		if dir[i].IsDir() {
			continue
		}
		if dir[i].Type().IsRegular() {
			fileName := dir[i].Name()
			validFile := strings.HasSuffix(fileName, ".pcap") && strings.HasPrefix(fileName, "caapture")
			if validFile {
				fileNames = append(fileNames, fileName)
			}

		}
	}
	if len(fileNames) < 2 {
		return
	}
	slices.Sort(fileNames)
	fullPath := s.filesPath + "/" + fileNames[0]
	if err = s.ReadFilePureGO(fullPath); err != nil { // process only first file
		s.l.Fatal("failed to read file", err, slog.String("file", fileNames[0]))
	}
	if err = os.Remove(fullPath); err != nil {
		s.l.Fatal("failed to remove file", err, slog.String("file", fileNames[0]))
	}
}
