package sketch

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/macvmio/geranos/pkg/duplicator"
	"github.com/macvmio/geranos/pkg/filesegment"
	"log"
	"os"
	"path/filepath"
)

func NewSketcher(rootDir string, manifestFilename string) *Sketcher {
	return &Sketcher{rootDirectory: rootDir, manifestFileName: manifestFilename}
}

type Sketcher struct {
	rootDirectory    string
	manifestFileName string
}

type cloneCandidate struct {
	descriptors []filesegment.Descriptor
	dirPath     string
	filename    string
}

func (cc *cloneCandidate) FilePath() string {
	return filepath.Join(cc.dirPath, cc.filename)
}

func resizeFile(filePath string, newSize int64) error {
	// Open file with read and write permissions
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Resize file to the specified newSize
	err = file.Truncate(newSize)
	if err != nil {
		return err
	}

	return nil
}

// fileExists checks if a file exists and is not a directory
func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (sc *Sketcher) findBestCloneCandidate(fileBlueprints []*fileBlueprint, cloneCandidates []*cloneCandidate) (*cloneCandidate, error) {
	var bestCloneCandidate *cloneCandidate
	bestScore := 0

	for _, fr := range fileBlueprints {
		// Create a map of digest segments for each file blueprint
		segmentsDigestMap := make(map[string]filesegment.Descriptor)
		for _, seg := range fr.Segments {
			segmentsDigestMap[seg.Digest().String()] = *seg
		}

		// Find the best matching candidate
		for _, cc := range cloneCandidates {
			score := sc.computeScore(segmentsDigestMap, cc)
			if score > bestScore {
				bestScore = score
				bestCloneCandidate = cc
			}
		}
	}

	if bestCloneCandidate == nil {
		return nil, fmt.Errorf("no matching clone candidate found")
	}

	return bestCloneCandidate, nil
}

func (sc *Sketcher) Sketch(dir string, manifest v1.Manifest) (bytesClonedCount int64, matchedSegmentsCount int64, err error) {

	fileBlueprints, err := createBlueprintsFromManifest(manifest)
	if err != nil {
		return 0, 0, err
	}

	cloneCandidates, err := sc.findCloneCandidates()
	if err != nil {
		return 0, 0, fmt.Errorf("encountered error while looking for manifests: %w", err)
	}
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to create directory '%v': %w", dir, err)
	}

	for _, fr := range fileBlueprints {
		if fileExists(filepath.Join(dir, fr.Filename)) {
			continue
		}
		// we will process each FR exactly once
		// fr can easily have 1000 layers,
		// each manifest can also have more than 1000 layers
		// we need to compute best score in expected linear time
		segmentsDigestMap := make(map[string]filesegment.Descriptor)
		for _, seg := range fr.Segments {
			segmentsDigestMap[seg.Digest().String()] = *seg
		}
		bestScore := 0
		var bestCloneCandidate *cloneCandidate
		for _, cc := range cloneCandidates {
			score := sc.computeScore(segmentsDigestMap, cc)
			if score > bestScore {
				bestScore = score
				bestCloneCandidate = cc
			}
		}
		if bestCloneCandidate == nil {
			continue
		}
		bytesClonedCount += fr.Size()
		matchedSegmentsCount += int64(bestScore)
		src := bestCloneCandidate.FilePath()
		dest := filepath.Join(dir, fr.Filename)
		if src == dest {
			continue
		}
		log.Printf("cloning file %s -> %s\n", src, dest)
		err = duplicator.CloneFile(src, dest)
		if err != nil {
			return bytesClonedCount, matchedSegmentsCount, fmt.Errorf("unable to clone source file '%v' to destination '%v': %w", src, dest, err)
		}
		err = resizeFile(dest, fr.Size())
		if err != nil {
			return bytesClonedCount, matchedSegmentsCount, fmt.Errorf("error occured while resizing file '%v' to its new size '%v': %w", dest, fr.Size(), err)
		}
	}
	return bytesClonedCount, matchedSegmentsCount, nil
}

// parseManifestFile represents a placeholder for your actual parsing logic.
func (sc *Sketcher) findCloneCandidates() ([]*cloneCandidate, error) {
	type Job struct {
		path string
	}
	jobs := make(chan Job, 8)

	go func() {
		filepath.Walk(sc.rootDirectory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error accessing path %q: %w\n", path, err)
			}
			if !info.IsDir() && info.Name() == sc.manifestFileName {
				jobs <- Job{path: path}
			}
			return nil
		})
		close(jobs) // Close the jobs channel when done walking the directory
	}()

	candidates := make([]*cloneCandidate, 0)
	for job := range jobs {
		f, err := os.Open(job.path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		manifest, err := v1.ParseManifest(f)
		if err != nil {
			return nil, err
		}
		// Map to group descriptors by filename
		fileDescriptorMap := make(map[string][]filesegment.Descriptor)

		// Parse each layer and group by filename
		for _, l := range manifest.Layers {
			segmentDescriptor, err := filesegment.ParseDescriptor(l)
			if err != nil {
				return nil, fmt.Errorf("unable to parse descriptor: %w", err)
			}
			filename := segmentDescriptor.Filename()
			fileDescriptorMap[filename] = append(fileDescriptorMap[filename], *segmentDescriptor)
		}

		// Create clone candidates for each file
		for filename, descriptors := range fileDescriptorMap {
			candidates = append(candidates, &cloneCandidate{
				descriptors: descriptors,            // All descriptors from the same file
				dirPath:     filepath.Dir(job.path), // Directory path from the job
				filename:    filename,
			})
		}
	}
	return candidates, nil
}

func (sc *Sketcher) computeScore(segmentDigestMap map[string]filesegment.Descriptor, m *cloneCandidate) int {
	score := 0
	seenDigests := make(map[string]struct{})
	for _, descriptor := range m.descriptors {
		digestStr := descriptor.Digest().String()
		if _, alreadySeen := seenDigests[digestStr]; !alreadySeen {
			seenDigests[digestStr] = struct{}{}
			if _, ok := segmentDigestMap[digestStr]; ok {
				score += 1
			}
		}
	}
	return score
}
