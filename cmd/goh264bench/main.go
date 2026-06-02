// SPDX-License-Identifier: LGPL-2.1-or-later

package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	goh264 "github.com/thesyncim/goh264"
)

type benchReport struct {
	Metadata benchMetadata `json:"metadata"`
	Results  []benchResult `json:"results"`
}

type benchMetadata struct {
	Input          string `json:"input"`
	InputBytes     int64  `json:"input_bytes"`
	InputMD5       string `json:"input_md5"`
	CorpusManifest string `json:"corpus_manifest,omitempty"`
	FailureLedger  string `json:"failure_ledger,omitempty"`
	CorpusFilter   string `json:"corpus_filter,omitempty"`
	CorpusEntries  int    `json:"corpus_entries,omitempty"`
	CorpusDecodeOK int    `json:"corpus_decode_ok_entries,omitempty"`
	CorpusBench    int    `json:"corpus_benchmarked_entries,omitempty"`
	CorpusKnownRed int    `json:"corpus_known_red_entries,omitempty"`
	CorpusSkipped  int    `json:"corpus_skipped_entries,omitempty"`
	FairnessPolicy string `json:"fairness_policy,omitempty"`
	GoVersion      string `json:"go_version"`
	GOOS           string `json:"goos"`
	GOARCH         string `json:"goarch"`
	NumCPU         int    `json:"num_cpu"`
	GOMAXPROCS     int    `json:"gomaxprocs"`
	ModulePath     string `json:"module_path,omitempty"`
	ModuleVersion  string `json:"module_version,omitempty"`
	VCSRevision    string `json:"vcs_revision,omitempty"`
	VCSDirty       bool   `json:"vcs_dirty"`
	FFmpegVersion  string `json:"ffmpeg_version,omitempty"`
	ComparisonKind string `json:"comparison_kind"`
}

type benchResult struct {
	Name              string        `json:"name"`
	EntryID           string        `json:"entry_id,omitempty"`
	Input             string        `json:"input"`
	Iterations        int           `json:"iterations"`
	Repeats           int           `json:"repeats"`
	Warmup            int           `json:"warmup"`
	RawOutput         bool          `json:"raw_output"`
	RawPixelFormat    string        `json:"raw_pixel_format,omitempty"`
	FFmpegPixelFmt    string        `json:"ffmpeg_pixel_format,omitempty"`
	FramesPerIter     int           `json:"frames_per_iter,omitempty"`
	InputBytesPerIter int64         `json:"input_bytes_per_iter,omitempty"`
	BytesPerIter      int64         `json:"bytes_per_iter,omitempty"`
	TotalFrames       int           `json:"total_frames,omitempty"`
	TotalBytes        int64         `json:"total_bytes,omitempty"`
	ElapsedMS         float64       `json:"elapsed_ms"`
	MeanElapsedMS     float64       `json:"mean_elapsed_ms,omitempty"`
	MedianElapsedMS   float64       `json:"median_elapsed_ms,omitempty"`
	MinElapsedMS      float64       `json:"min_elapsed_ms,omitempty"`
	MaxElapsedMS      float64       `json:"max_elapsed_ms,omitempty"`
	StddevElapsedMS   float64       `json:"stddev_elapsed_ms,omitempty"`
	CVElapsed         float64       `json:"cv_elapsed,omitempty"`
	FPS               float64       `json:"fps,omitempty"`
	MiBPerSec         float64       `json:"mib_per_sec,omitempty"`
	NSPerFrame        float64       `json:"ns_per_frame,omitempty"`
	NSPerInputByte    float64       `json:"ns_per_input_byte,omitempty"`
	NSPerRawByte      float64       `json:"ns_per_raw_byte,omitempty"`
	AllocBytes        uint64        `json:"alloc_bytes,omitempty"`
	Allocs            uint64        `json:"allocs,omitempty"`
	RawMD5            string        `json:"raw_md5,omitempty"`
	ExpectedRawMD5    string        `json:"expected_raw_md5,omitempty"`
	ExpectedPixFmt    string        `json:"expected_raw_pixel_format,omitempty"`
	ExpectedFrames    int           `json:"expected_frames_per_iter,omitempty"`
	ExpectedBytes     int64         `json:"expected_bytes_per_iter,omitempty"`
	ParityStatus      string        `json:"parity_status,omitempty"`
	ErrorClass        string        `json:"error_class,omitempty"`
	Surfaces          []string      `json:"surfaces,omitempty"`
	FeatureTags       []string      `json:"feature_tags,omitempty"`
	Source            string        `json:"source,omitempty"`
	Command           string        `json:"command,omitempty"`
	ProcessPerIter    bool          `json:"process_per_iter"`
	InputReadTimed    bool          `json:"input_read_timed"`
	StdoutPipeTimed   bool          `json:"stdout_pipe_timed"`
	BaselineKind      string        `json:"baseline_kind"`
	Skipped           bool          `json:"skipped,omitempty"`
	Error             string        `json:"error,omitempty"`
	Notes             []string      `json:"notes,omitempty"`
	Samples           []benchSample `json:"samples,omitempty"`
}

type benchSample struct {
	ElapsedMS   float64 `json:"elapsed_ms"`
	TotalFrames int     `json:"total_frames,omitempty"`
	TotalBytes  int64   `json:"total_bytes,omitempty"`
	FPS         float64 `json:"fps,omitempty"`
	MiBPerSec   float64 `json:"mib_per_sec,omitempty"`
	AllocBytes  uint64  `json:"alloc_bytes,omitempty"`
	Allocs      uint64  `json:"allocs,omitempty"`
	RawMD5      string  `json:"raw_md5,omitempty"`
}

type benchOptions struct {
	iters         int
	repeats       int
	warmup        int
	rawOutput     bool
	runFFmpeg     bool
	ffmpegBin     string
	ffmpegThreads string
	ffmpegPixFmt  string
	strictPixFmt  bool
	corpusFilter  string
	failureLedger string
	annexBInput   bool
}

type benchCorpusEntry struct {
	ID           string   `json:"id"`
	Path         string   `json:"path"`
	URL          string   `json:"url,omitempty"`
	Format       string   `json:"format"`
	Expect       string   `json:"expect"`
	PixFmt       string   `json:"pix_fmt,omitempty"`
	FrameCount   int      `json:"frame_count,omitempty"`
	FrameSize    int      `json:"frame_size,omitempty"`
	BitstreamMD5 string   `json:"bitstream_md5,omitempty"`
	RawVideoMD5  string   `json:"rawvideo_md5,omitempty"`
	FrameMD5     []string `json:"frame_md5,omitempty"`
	Surfaces     []string `json:"surfaces,omitempty"`
	GuardTags    []string `json:"guard_tags,omitempty"`
	FeatureTags  []string `json:"feature_tags,omitempty"`
	Source       string   `json:"source,omitempty"`
}

func main() {
	input := flag.String("input", "", "H.264 input file")
	manifest := flag.String("manifest", "", "JSONL H.264 corpus manifest; benchmarks decode-ok entries after oracle parity validation")
	maxEntries := flag.Int("max-entries", 0, "maximum decode-ok manifest entries to benchmark; 0 means all")
	corpusFilter := flag.String("filter", os.Getenv("GOH264_CORPUS_FILTER"), "comma/space-separated manifest entry filter; defaults to GOH264_CORPUS_FILTER")
	failureLedger := flag.String("failure-ledger", "auto", "manifest known-red ledger: auto uses failures.jsonl next to the manifest when present, off disables it, otherwise pass a JSONL path")
	iters := flag.Int("iters", 5, "measured iterations")
	repeats := flag.Int("repeats", 1, "measured repeat samples; each sample runs -iters decodes")
	warmup := flag.Int("warmup", 1, "warmup iterations")
	rawOutput := flag.Bool("raw", true, "materialize raw decoded bytes during Go and FFmpeg runs")
	runFFmpeg := flag.Bool("ffmpeg", false, "also run an FFmpeg baseline over the same file")
	ffmpegBin := flag.String("ffmpeg-bin", "ffmpeg", "FFmpeg binary")
	ffmpegThreads := flag.String("ffmpeg-threads", "1", "FFmpeg -threads value")
	ffmpegPixFmt := flag.String("ffmpeg-pix-fmt", "", "FFmpeg output pixel format for -raw mode; defaults to Go raw pixel format when available")
	strictPixFmt := flag.Bool("strict-pix-fmt", false, "reject a user-supplied -ffmpeg-pix-fmt that differs from Go raw pixel format")
	jsonOut := flag.Bool("json", false, "print JSON")
	flag.Parse()

	if (*input == "") == (*manifest == "") || *iters <= 0 || *repeats <= 0 || *warmup < 0 || *maxEntries < 0 {
		fmt.Fprintln(os.Stderr, "usage: goh264bench (-input file.h264 | -manifest corpus.jsonl) [-iters 5] [-repeats 1] [-warmup 1] [-ffmpeg] [-json]")
		os.Exit(2)
	}
	opts := benchOptions{
		iters:         *iters,
		repeats:       *repeats,
		warmup:        *warmup,
		rawOutput:     *rawOutput,
		runFFmpeg:     *runFFmpeg,
		ffmpegBin:     *ffmpegBin,
		ffmpegThreads: *ffmpegThreads,
		ffmpegPixFmt:  *ffmpegPixFmt,
		strictPixFmt:  *strictPixFmt,
		corpusFilter:  *corpusFilter,
		failureLedger: *failureLedger,
	}
	report, err := buildBenchReport(*input, *manifest, *maxEntries, opts)
	if err != nil {
		die("benchmark", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			die("json", err)
		}
		return
	}
	fmt.Printf("input: %s, %d bytes, md5 %s\n", report.Metadata.Input, report.Metadata.InputBytes, report.Metadata.InputMD5)
	for _, r := range report.Results {
		if r.Skipped {
			fmt.Printf("%s: skipped", r.Name)
			if r.EntryID != "" {
				fmt.Printf(", entry %s", r.EntryID)
			}
			if r.ParityStatus != "" {
				fmt.Printf(", parity %s", r.ParityStatus)
			}
			if r.Error != "" {
				fmt.Printf(", error %s", r.Error)
			}
			if r.ErrorClass != "" {
				fmt.Printf(", class %s", r.ErrorClass)
			}
			if len(r.FeatureTags) != 0 {
				fmt.Printf(", features %s", strings.Join(r.FeatureTags, ","))
			}
			if len(r.Surfaces) != 0 {
				fmt.Printf(", surfaces %s", strings.Join(r.Surfaces, ","))
			}
			if r.Source != "" {
				fmt.Printf(", source %q", r.Source)
			}
			for _, note := range r.Notes {
				fmt.Printf("\n  note: %s", note)
			}
			fmt.Println()
			continue
		}
		fmt.Printf("%s: %.2f ms over %d repeat(s) x %d iter", r.Name, r.ElapsedMS, r.Repeats, r.Iterations)
		if r.EntryID != "" {
			fmt.Printf(", entry %s", r.EntryID)
		}
		if r.Repeats > 1 {
			fmt.Printf(", median %.2f ms, cv %.4f", r.MedianElapsedMS, r.CVElapsed)
		}
		if r.FramesPerIter > 0 {
			fmt.Printf(", %d frames/iter, %.2f fps", r.FramesPerIter, r.FPS)
		}
		if r.BytesPerIter > 0 {
			fmt.Printf(", %d bytes/iter, %.2f MiB/s", r.BytesPerIter, r.MiBPerSec)
		}
		if r.NSPerInputByte > 0 {
			fmt.Printf(", %.2f ns/input-byte", r.NSPerInputByte)
		}
		if r.NSPerRawByte > 0 {
			fmt.Printf(", %.2f ns/raw-byte", r.NSPerRawByte)
		}
		if r.Allocs > 0 || r.AllocBytes > 0 {
			fmt.Printf(", %.2f allocs/iter, %.2f MiB alloc/iter",
				float64(r.Allocs)/float64(r.Iterations),
				float64(r.AllocBytes)/float64(r.Iterations)/(1024*1024))
		}
		if r.RawMD5 != "" {
			fmt.Printf(", raw md5 %s", r.RawMD5)
		}
		if r.ParityStatus != "" {
			fmt.Printf(", parity %s", r.ParityStatus)
		}
		if r.Command != "" {
			fmt.Printf("\n  %s", r.Command)
		}
		for _, note := range r.Notes {
			fmt.Printf("\n  note: %s", note)
		}
		fmt.Println()
	}
}

func buildBenchReport(input string, manifest string, maxEntries int, opts benchOptions) (benchReport, error) {
	if manifest != "" {
		return benchManifest(manifest, maxEntries, opts)
	}
	data, err := os.ReadFile(input)
	if err != nil {
		return benchReport{}, fmt.Errorf("read input: %w", err)
	}
	results, err := benchOneInput(input, data, opts)
	if err != nil {
		return benchReport{}, err
	}
	return benchReport{
		Metadata: benchmarkMetadata(input, data, opts.runFFmpeg, opts.ffmpegBin),
		Results:  results,
	}, nil
}

func benchOneInput(input string, data []byte, opts benchOptions) ([]benchResult, error) {
	goResult, err := benchGo(input, data, opts.iters, opts.repeats, opts.warmup, opts.rawOutput, opts.annexBInput)
	if err != nil {
		return nil, fmt.Errorf("goh264: %w", err)
	}
	results := []benchResult{goResult}

	if opts.runFFmpeg && opts.rawOutput && opts.strictPixFmt && opts.ffmpegPixFmt != "" && goResult.RawPixelFormat != "" && opts.ffmpegPixFmt != goResult.RawPixelFormat {
		return nil, fmt.Errorf("-ffmpeg-pix-fmt %q does not match Go raw pixel format %q", opts.ffmpegPixFmt, goResult.RawPixelFormat)
	}
	if opts.runFFmpeg {
		ffmpegResult, err := benchFFmpeg(input, int64(len(data)), opts.iters, opts.repeats, opts.warmup, opts.rawOutput, opts.ffmpegBin, opts.ffmpegThreads, opts.ffmpegPixFmt, goResult.RawPixelFormat)
		if err != nil {
			return nil, fmt.Errorf("ffmpeg: %w", err)
		}
		results = append(results, ffmpegResult)
	}
	return results, nil
}

func benchManifest(path string, maxEntries int, opts benchOptions) (benchReport, error) {
	if !opts.rawOutput {
		return benchReport{}, fmt.Errorf("manifest benchmark mode requires -raw=true so oracle rawvideo parity is checked")
	}
	manifestData, err := os.ReadFile(path)
	if err != nil {
		return benchReport{}, fmt.Errorf("read manifest: %w", err)
	}
	entries, err := readBenchCorpusManifest(path)
	if err != nil {
		return benchReport{}, err
	}
	failureLedger, failureLedgerPath, err := readBenchFailureLedger(path, opts.failureLedger, entries)
	if err != nil {
		return benchReport{}, err
	}
	if filter := benchCorpusFilterTokens(opts.corpusFilter); len(filter) != 0 {
		entries = filterBenchCorpusEntries(entries, filter)
		if len(entries) == 0 {
			return benchReport{}, fmt.Errorf("%s: no corpus entries matched filter %q", path, opts.corpusFilter)
		}
	}

	baseDir := filepath.Dir(path)
	var results []benchResult
	var benchmarked int
	var knownRed int
	var skipped int
	for _, entry := range entries {
		if entry.Expect != "decode-ok" {
			results = append(results, skippedBenchResult(entry, "manifest row is not a decode-ok oracle row and is not a timing sample"))
			skipped++
			continue
		}
		if maxEntries > 0 && benchmarked >= maxEntries {
			break
		}
		if err := validateBenchCorpusEntry(entry); err != nil {
			return benchReport{}, err
		}
		inputPath, err := resolveBenchCorpusPath(baseDir, entry)
		if err != nil {
			if _, ok := failureLedger[entry.ID]; ok {
				results = append(results, knownRedBenchResult(entry, "", nil, err, failureLedgerPath))
				knownRed++
				continue
			}
			return benchReport{}, err
		}
		data, err := os.ReadFile(inputPath)
		if err != nil {
			if _, ok := failureLedger[entry.ID]; ok {
				results = append(results, knownRedBenchResult(entry, inputPath, nil, err, failureLedgerPath))
				knownRed++
				continue
			}
			return benchReport{}, fmt.Errorf("%s: read input: %w", entry.ID, err)
		}
		if err := validateBenchBitstreamMD5(entry, data); err != nil {
			return benchReport{}, err
		}
		staleLedger := false
		if err := preflightBenchGoOracle(inputPath, data, entry); err != nil {
			if _, ok := failureLedger[entry.ID]; ok {
				results = append(results, knownRedBenchResult(entry, inputPath, data, err, failureLedgerPath))
				knownRed++
				continue
			}
			return benchReport{}, fmt.Errorf("%s: goh264 oracle preflight: %w", entry.ID, err)
		} else if _, ok := failureLedger[entry.ID]; ok {
			staleLedger = true
		}
		if opts.runFFmpeg {
			if err := preflightBenchFFmpegOracle(inputPath, entry, opts); err != nil {
				return benchReport{}, fmt.Errorf("%s: ffmpeg oracle preflight: %w", entry.ID, err)
			}
		}
		entryOpts := opts
		entryOpts.annexBInput = entry.Format == "annexb"
		entryResults, err := benchOneInput(inputPath, data, entryOpts)
		if err != nil {
			return benchReport{}, fmt.Errorf("%s: %w", entry.ID, err)
		}
		for i := range entryResults {
			if err := annotateBenchResultWithOracle(&entryResults[i], entry); err != nil {
				return benchReport{}, err
			}
			if staleLedger {
				entryResults[i].ParityStatus = "rawvideo-md5-ok-failure-ledger-stale"
				entryResults[i].Notes = append(entryResults[i].Notes,
					fmt.Sprintf("entry is still listed in %s but passed Go oracle preflight; update the failure ledger before using this as a green benchmark lane", failureLedgerPath),
				)
			}
		}
		results = append(results, entryResults...)
		benchmarked++
	}
	if len(results) == 0 {
		return benchReport{}, fmt.Errorf("%s: no manifest entries selected", path)
	}

	meta := benchmarkMetadata(path, manifestData, opts.runFFmpeg, opts.ffmpegBin)
	meta.CorpusManifest = path
	meta.FailureLedger = failureLedgerPath
	meta.CorpusFilter = opts.corpusFilter
	meta.CorpusEntries = len(entries)
	meta.CorpusDecodeOK = benchmarked
	meta.CorpusBench = benchmarked
	meta.CorpusKnownRed = knownRed
	meta.CorpusSkipped = skipped
	meta.ComparisonKind = "manifest-goh264-in-process"
	if opts.runFFmpeg {
		meta.ComparisonKind = "manifest-goh264-in-process-vs-ffmpeg-cli"
	}
	meta.FairnessPolicy = "Decode-ok corpus entries are benchmarked only after bitstream MD5, Go raw pixel format, frame count, raw byte count, and concatenated rawvideo MD5 pass a preflight against the manifest oracle; manifest rows use their declared input format for the Go decoder path. Known-red ledger rows that do not pass Go oracle preflight are emitted as skipped results with the exact error and are not timing samples. Optional FFmpeg CLI rawvideo output must pass the same rawvideo MD5 preflight before measured FFmpeg samples run. FFmpeg timing remains a process-per-iteration CLI baseline."
	return benchReport{Metadata: meta, Results: results}, nil
}

func readBenchFailureLedger(manifestPath string, mode string, manifestEntries []benchCorpusEntry) (map[string]benchCorpusEntry, string, error) {
	path, err := benchFailureLedgerPath(manifestPath, mode)
	if err != nil {
		return nil, "", err
	}
	if path == "" {
		return nil, "", nil
	}
	entries, err := readBenchCorpusManifest(path)
	if err != nil {
		return nil, "", err
	}
	manifestByID := make(map[string]benchCorpusEntry, len(manifestEntries))
	for _, entry := range manifestEntries {
		manifestByID[entry.ID] = entry
	}
	failures := make(map[string]benchCorpusEntry, len(entries))
	for _, failure := range entries {
		if err := validateBenchCorpusEntry(failure); err != nil {
			return nil, "", fmt.Errorf("%s: failure-ledger row: %w", failure.ID, err)
		}
		if _, ok := failures[failure.ID]; ok {
			return nil, "", fmt.Errorf("%s: duplicate failure-ledger id in %s", failure.ID, path)
		}
		manifestEntry, ok := manifestByID[failure.ID]
		if !ok {
			return nil, "", fmt.Errorf("%s: failure-ledger row missing from %s", failure.ID, manifestPath)
		}
		if !reflect.DeepEqual(failure, manifestEntry) {
			return nil, "", fmt.Errorf("%s: failure-ledger row drifted from %s", failure.ID, manifestPath)
		}
		failures[failure.ID] = failure
	}
	return failures, path, nil
}

func benchFailureLedgerPath(manifestPath string, mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "auto":
		path := filepath.Join(filepath.Dir(manifestPath), "failures.jsonl")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if os.IsNotExist(err) {
			return "", nil
		} else {
			return "", err
		}
	case "off", "none", "false", "0":
		return "", nil
	default:
		return mode, nil
	}
}

func benchCorpusFilterTokens(filter string) []string {
	if filter == "" {
		return nil
	}
	return strings.FieldsFunc(strings.ToLower(filter), func(r rune) bool {
		switch r {
		case ',', ';', ' ', '\t', '\n', '\r':
			return true
		default:
			return false
		}
	})
}

func filterBenchCorpusEntries(entries []benchCorpusEntry, tokens []string) []benchCorpusEntry {
	filtered := entries[:0]
	for _, entry := range entries {
		if benchCorpusEntryMatches(entry, tokens) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func benchCorpusEntryMatches(entry benchCorpusEntry, tokens []string) bool {
	haystack := strings.ToLower(strings.Join(benchCorpusEntrySearchFields(entry), "\x00"))
	for _, token := range tokens {
		if token != "" && !strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}

func benchCorpusEntrySearchFields(entry benchCorpusEntry) []string {
	fields := []string{
		entry.ID,
		entry.Path,
		entry.URL,
		entry.Format,
		entry.Expect,
		entry.PixFmt,
		entry.Source,
	}
	fields = append(fields, entry.Surfaces...)
	fields = append(fields, entry.GuardTags...)
	fields = append(fields, entry.FeatureTags...)
	return fields
}

func readBenchCorpusManifest(path string) ([]benchCorpusEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open manifest %s: %w", path, err)
	}
	defer f.Close()

	var entries []benchCorpusEntry
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var entry benchCorpusEntry
		if err := json.Unmarshal([]byte(text), &entry); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	return entries, nil
}

func validateBenchCorpusEntry(entry benchCorpusEntry) error {
	if entry.ID == "" || entry.Path == "" && entry.URL == "" {
		return fmt.Errorf("manifest entry id and path or url must be set: %+v", entry)
	}
	if entry.Format != "annexb" {
		return fmt.Errorf("%s: format = %q, want annexb", entry.ID, entry.Format)
	}
	if entry.Expect != "decode-ok" {
		return fmt.Errorf("%s: benchmark manifest mode only runs decode-ok entries, got %q", entry.ID, entry.Expect)
	}
	if entry.BitstreamMD5 == "" || entry.RawVideoMD5 == "" || entry.PixFmt == "" {
		return fmt.Errorf("%s: decode-ok entries need bitstream_md5, rawvideo_md5, and pix_fmt", entry.ID)
	}
	if entry.FrameCount <= 0 || entry.FrameSize <= 0 {
		return fmt.Errorf("%s: frame_count/frame_size must be positive", entry.ID)
	}
	if len(entry.FrameMD5) != 0 && len(entry.FrameMD5) != entry.FrameCount {
		return fmt.Errorf("%s: frame_md5 count = %d, want 0 or %d", entry.ID, len(entry.FrameMD5), entry.FrameCount)
	}
	return nil
}

func resolveBenchCorpusPath(baseDir string, entry benchCorpusEntry) (string, error) {
	if entry.Path != "" {
		path := entry.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		if entry.URL == "" {
			return path, nil
		}
	}
	if entry.URL == "" {
		return "", fmt.Errorf("%s: no path or url", entry.ID)
	}
	rel := entry.Path
	if rel == "" {
		rel = filepath.Base(entry.URL)
	}
	rel, err := cleanRelativeBenchCorpusPath(entry.ID, rel)
	if err != nil {
		return "", err
	}
	cacheRoot := os.Getenv("GOH264_CORPUS_CACHE")
	if cacheRoot == "" {
		cacheRoot = filepath.Join(baseDir, "cache")
	}
	path := filepath.Join(cacheRoot, rel)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if os.Getenv("GOH264_CORPUS_FETCH") != "1" {
		return "", fmt.Errorf("%s: missing %s; set GOH264_CORPUS_FETCH=1 to download %s", entry.ID, path, entry.URL)
	}
	if err := downloadBenchCorpusEntry(entry, path); err != nil {
		return "", err
	}
	return path, nil
}

func cleanRelativeBenchCorpusPath(id string, path string) (string, error) {
	clean := filepath.Clean(path)
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("%s: unsafe corpus path %q", id, path)
	}
	return clean, nil
}

func downloadBenchCorpusEntry(entry benchCorpusEntry, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("%s: create corpus cache dir: %w", entry.ID, err)
	}
	resp, err := http.Get(entry.URL)
	if err != nil {
		return fmt.Errorf("%s: download %s: %w", entry.ID, entry.URL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: download %s: status %s", entry.ID, entry.URL, resp.Status)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("%s: create %s: %w", entry.ID, tmp, err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("%s: write %s: %w", entry.ID, tmp, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("%s: close %s: %w", entry.ID, tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("%s: install %s: %w", entry.ID, path, err)
	}
	return nil
}

func validateBenchBitstreamMD5(entry benchCorpusEntry, data []byte) error {
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != entry.BitstreamMD5 {
		return fmt.Errorf("%s: bitstream_md5 = %s, want %s", entry.ID, got, entry.BitstreamMD5)
	}
	return nil
}

func annotateBenchResultWithOracle(result *benchResult, entry benchCorpusEntry) error {
	if result == nil {
		return fmt.Errorf("%s: nil benchmark result", entry.ID)
	}
	expectedBytes := int64(entry.FrameCount * entry.FrameSize)
	result.EntryID = entry.ID
	result.ExpectedRawMD5 = entry.RawVideoMD5
	result.ExpectedPixFmt = entry.PixFmt
	result.ExpectedFrames = entry.FrameCount
	result.ExpectedBytes = expectedBytes
	result.Surfaces = append(result.Surfaces[:0], entry.Surfaces...)
	result.FeatureTags = append(result.FeatureTags[:0], entry.FeatureTags...)
	result.Source = entry.Source
	if result.RawMD5 != entry.RawVideoMD5 {
		return fmt.Errorf("%s %s: raw_md5 = %s, want %s", entry.ID, result.Name, result.RawMD5, entry.RawVideoMD5)
	}
	if result.BytesPerIter != expectedBytes {
		return fmt.Errorf("%s %s: bytes_per_iter = %d, want %d", entry.ID, result.Name, result.BytesPerIter, expectedBytes)
	}
	if result.Name == "goh264" {
		if result.FramesPerIter != entry.FrameCount {
			return fmt.Errorf("%s: Go frames_per_iter = %d, want %d", entry.ID, result.FramesPerIter, entry.FrameCount)
		}
		if result.RawPixelFormat != entry.PixFmt {
			return fmt.Errorf("%s: Go raw_pixel_format = %s, want %s", entry.ID, result.RawPixelFormat, entry.PixFmt)
		}
	}
	result.ParityStatus = "rawvideo-md5-ok"
	return nil
}

func skippedBenchResult(entry benchCorpusEntry, reason string) benchResult {
	result := benchResult{
		Name:            "goh264",
		EntryID:         entry.ID,
		Input:           entry.Path,
		RawOutput:       true,
		RawPixelFormat:  entry.PixFmt,
		ExpectedRawMD5:  entry.RawVideoMD5,
		ExpectedPixFmt:  entry.PixFmt,
		ExpectedFrames:  entry.FrameCount,
		ExpectedBytes:   int64(entry.FrameCount * entry.FrameSize),
		ParityStatus:    entry.Expect,
		Surfaces:        append([]string(nil), entry.Surfaces...),
		FeatureTags:     append([]string(nil), entry.FeatureTags...),
		Source:          entry.Source,
		BaselineKind:    "manifest-skipped",
		ProcessPerIter:  false,
		InputReadTimed:  false,
		StdoutPipeTimed: false,
		Skipped:         true,
	}
	if reason != "" {
		result.Notes = append(result.Notes, reason)
	}
	return result
}

func knownRedBenchResult(entry benchCorpusEntry, input string, data []byte, err error, ledgerPath string) benchResult {
	result := skippedBenchResult(entry, "listed in the known-red failure ledger and not included in timing aggregates")
	if input != "" {
		result.Input = input
	}
	result.InputBytesPerIter = int64(len(data))
	result.ParityStatus = "known-red"
	result.BaselineKind = "oracle-known-red"
	if ledgerPath != "" {
		result.Notes = append(result.Notes, "failure ledger: "+ledgerPath)
	}
	if err != nil {
		result.Error = err.Error()
		result.ErrorClass = benchOracleFailureClass(err.Error())
	}
	return result
}

func benchOracleFailureClass(detail string) string {
	detail = strings.ToLower(detail)
	switch {
	case detail == "":
		return ""
	case strings.Contains(detail, "missing ") || strings.Contains(detail, "no such file"):
		return "input-missing"
	case strings.Contains(detail, "decode") || strings.Contains(detail, "unsupported"):
		return "decode-error"
	case strings.Contains(detail, "frames_per_iter") || strings.Contains(detail, "frames ="):
		return "frame-count-mismatch"
	case strings.Contains(detail, "raw_pixel_format") || strings.Contains(detail, "pix_fmt"):
		return "pixel-format-mismatch"
	case strings.Contains(detail, "bytes_per_iter") || strings.Contains(detail, "raw size") || strings.Contains(detail, "raw total"):
		return "raw-size-mismatch"
	case strings.Contains(detail, "bitstream_md5"):
		return "bitstream-md5-mismatch"
	case strings.Contains(detail, "raw_md5") || strings.Contains(detail, "rawvideo md5") || strings.Contains(detail, "md5 ="):
		return "raw-md5-mismatch"
	default:
		return "oracle-mismatch"
	}
}

func preflightBenchGoOracle(input string, data []byte, entry benchCorpusEntry) error {
	run, err := decodeGoOnceForFormat(data, true, entry.Format == "annexb")
	if err != nil {
		return err
	}
	result := benchResult{
		Name:           "goh264",
		Input:          input,
		RawOutput:      true,
		RawPixelFormat: run.pixFmt,
		FramesPerIter:  run.frames,
		BytesPerIter:   run.bytes,
		RawMD5:         run.md5,
	}
	return annotateBenchResultWithOracle(&result, entry)
}

func preflightBenchFFmpegOracle(input string, entry benchCorpusEntry, opts benchOptions) error {
	pixFmt := opts.ffmpegPixFmt
	if pixFmt == "" {
		pixFmt = entry.PixFmt
	} else if opts.strictPixFmt && pixFmt != entry.PixFmt {
		return fmt.Errorf("-ffmpeg-pix-fmt %q does not match manifest pixel format %q", pixFmt, entry.PixFmt)
	}
	run, err := runFFmpegOnce(opts.ffmpegBin, ffmpegArgs(input, true, opts.ffmpegThreads, pixFmt), true)
	if err != nil {
		return err
	}
	result := benchResult{
		Name:           "ffmpeg",
		Input:          input,
		RawOutput:      true,
		RawPixelFormat: entry.PixFmt,
		FFmpegPixelFmt: pixFmt,
		BytesPerIter:   run.bytes,
		RawMD5:         run.md5,
	}
	return annotateBenchResultWithOracle(&result, entry)
}

func benchGo(input string, data []byte, iters int, repeats int, warmup int, rawOutput bool, annexBInput bool) (benchResult, error) {
	for i := 0; i < warmup; i++ {
		if _, err := decodeGoOnceForFormat(data, rawOutput, annexBInput); err != nil {
			return benchResult{}, err
		}
	}

	var framesPerIter int
	var bytesPerIter int64
	var rawMD5 string
	var pixFmt string
	var samples []benchSample
	for repeat := 0; repeat < repeats; repeat++ {
		sample, frames, bytes, sum, samplePixFmt, err := measureGoSample(data, iters, rawOutput, annexBInput)
		if err != nil {
			return benchResult{}, err
		}
		if repeat == 0 {
			framesPerIter = frames
			bytesPerIter = bytes
			rawMD5 = sum
			pixFmt = samplePixFmt
		}
		if frames != framesPerIter || bytes != bytesPerIter {
			return benchResult{}, fmt.Errorf("unstable decode result at repeat %d: frames/bytes = %d/%d, want %d/%d", repeat, frames, bytes, framesPerIter, bytesPerIter)
		}
		if sum != rawMD5 {
			return benchResult{}, fmt.Errorf("unstable raw md5 at repeat %d: %s, want %s", repeat, sum, rawMD5)
		}
		if samplePixFmt != pixFmt {
			return benchResult{}, fmt.Errorf("unstable raw pixel format at repeat %d: %s, want %s", repeat, samplePixFmt, pixFmt)
		}
		samples = append(samples, sample)
	}

	result := resultFromSamples("goh264", input, iters, repeats, warmup, rawOutput, framesPerIter, bytesPerIter, samples, rawMD5, "")
	result.RawPixelFormat = pixFmt
	result.InputBytesPerIter = int64(len(data))
	result.BaselineKind = "in-process-go"
	result.ProcessPerIter = false
	result.InputReadTimed = false
	result.StdoutPipeTimed = false
	annotateBenchRates(&result)
	return result, nil
}

func measureGoSample(data []byte, iters int, rawOutput bool, annexBInput bool) (benchSample, int, int64, string, string, error) {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()

	var framesPerIter int
	var bytesPerIter int64
	var rawMD5 string
	var pixFmt string
	for i := 0; i < iters; i++ {
		run, err := decodeGoOnceForFormat(data, rawOutput, annexBInput)
		if err != nil {
			return benchSample{}, 0, 0, "", "", err
		}
		if i == 0 {
			framesPerIter = run.frames
			bytesPerIter = run.bytes
			pixFmt = run.pixFmt
		}
		if run.frames != framesPerIter || run.bytes != bytesPerIter {
			return benchSample{}, 0, 0, "", "", fmt.Errorf("unstable decode result at iter %d: frames/bytes = %d/%d, want %d/%d", i, run.frames, run.bytes, framesPerIter, bytesPerIter)
		}
		if run.pixFmt != pixFmt {
			return benchSample{}, 0, 0, "", "", fmt.Errorf("unstable pixel format at iter %d: %s, want %s", i, run.pixFmt, pixFmt)
		}
		rawMD5 = run.md5
	}
	elapsed := time.Since(start)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	sample := sampleFromTotals(iters, framesPerIter, bytesPerIter, elapsed, after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs, rawMD5)
	return sample, framesPerIter, bytesPerIter, rawMD5, pixFmt, nil
}

type decodeGoRun struct {
	frames int
	bytes  int64
	md5    string
	pixFmt string
}

func decodeGoOnce(data []byte, rawOutput bool) (decodeGoRun, error) {
	dec := goh264.NewDecoder()
	frames, err := dec.DecodeFrames(data)
	if err != nil {
		return decodeGoRun{}, err
	}
	delayed, err := dec.DecodeFrames(nil)
	if err != nil {
		return decodeGoRun{}, err
	}
	frames = append(frames, delayed...)
	return summarizeGoFrames(frames, rawOutput)
}

func decodeGoOnceForFormat(data []byte, rawOutput bool, annexBInput bool) (decodeGoRun, error) {
	if !annexBInput {
		return decodeGoOnce(data, rawOutput)
	}
	frames, err := goh264.NewDecoder().DecodeAnnexBFrames(data)
	if err != nil {
		return decodeGoRun{}, err
	}
	return summarizeGoFrames(frames, rawOutput)
}

func summarizeGoFrames(frames []*goh264.Frame, rawOutput bool) (decodeGoRun, error) {
	var pixFmt string
	for i, frame := range frames {
		framePixFmt, err := frame.RawPixelFormat()
		if err != nil {
			return decodeGoRun{}, err
		}
		if i == 0 {
			pixFmt = framePixFmt
		} else if framePixFmt != pixFmt {
			return decodeGoRun{}, fmt.Errorf("mixed raw pixel formats: frame[0]=%s frame[%d]=%s", pixFmt, i, framePixFmt)
		}
	}
	if !rawOutput {
		return decodeGoRun{frames: len(frames), pixFmt: pixFmt}, nil
	}
	h := md5.New()
	var scratch []byte
	var total int64
	for _, frame := range frames {
		scratch = scratch[:0]
		var err error
		scratch, err = frame.AppendRawYUVBytesLE(scratch)
		if err != nil {
			return decodeGoRun{}, err
		}
		total += int64(len(scratch))
		if _, err := h.Write(scratch); err != nil {
			return decodeGoRun{}, err
		}
	}
	return decodeGoRun{frames: len(frames), bytes: total, md5: hashString(h), pixFmt: pixFmt}, nil
}

func benchFFmpeg(input string, inputBytes int64, iters int, repeats int, warmup int, rawOutput bool, bin string, threads string, pixFmt string, goPixFmt string) (benchResult, error) {
	effectivePixFmt := pixFmt
	autoPixFmt := false
	if rawOutput && effectivePixFmt == "" && goPixFmt != "" {
		effectivePixFmt = goPixFmt
		autoPixFmt = true
	}
	args := ffmpegArgs(input, rawOutput, threads, effectivePixFmt)
	for i := 0; i < warmup; i++ {
		if _, err := runFFmpegOnce(bin, args, rawOutput); err != nil {
			return benchResult{}, err
		}
	}

	var bytesPerIter int64
	var rawMD5 string
	var samples []benchSample
	for repeat := 0; repeat < repeats; repeat++ {
		sample, bytes, sum, err := measureFFmpegSample(bin, args, iters, rawOutput)
		if err != nil {
			return benchResult{}, err
		}
		if repeat == 0 {
			bytesPerIter = bytes
			rawMD5 = sum
		}
		if bytes != bytesPerIter {
			return benchResult{}, fmt.Errorf("unstable FFmpeg byte count at repeat %d: %d, want %d", repeat, bytes, bytesPerIter)
		}
		if sum != rawMD5 {
			return benchResult{}, fmt.Errorf("unstable FFmpeg raw md5 at repeat %d: %s, want %s", repeat, sum, rawMD5)
		}
		samples = append(samples, sample)
	}

	result := resultFromSamples("ffmpeg", input, iters, repeats, warmup, rawOutput, 0, bytesPerIter, samples, rawMD5, bin+" "+joinArgs(args))
	result.RawPixelFormat = goPixFmt
	result.FFmpegPixelFmt = effectivePixFmt
	result.InputBytesPerIter = inputBytes
	result.BaselineKind = "ffmpeg-cli"
	result.ProcessPerIter = true
	result.InputReadTimed = true
	result.StdoutPipeTimed = rawOutput
	result.Notes = append(result.Notes,
		"FFmpeg is executed once per timed iteration, so this baseline includes process startup, CLI demux/parser setup, input file reads, and stdout pipe cost.",
	)
	if autoPixFmt {
		result.Notes = append(result.Notes, "FFmpeg -pix_fmt was auto-selected from the Go raw pixel format for raw-MD5 parity.")
	}
	annotateBenchRates(&result)
	return result, nil
}

func measureFFmpegSample(bin string, args []string, iters int, rawOutput bool) (benchSample, int64, string, error) {
	start := time.Now()
	var bytesPerIter int64
	var rawMD5 string
	for i := 0; i < iters; i++ {
		run, err := runFFmpegOnce(bin, args, rawOutput)
		if err != nil {
			return benchSample{}, 0, "", err
		}
		if i == 0 {
			bytesPerIter = run.bytes
		}
		if run.bytes != bytesPerIter {
			return benchSample{}, 0, "", fmt.Errorf("unstable FFmpeg byte count at iter %d: %d, want %d", i, run.bytes, bytesPerIter)
		}
		rawMD5 = run.md5
	}
	elapsed := time.Since(start)
	return sampleFromTotals(iters, 0, bytesPerIter, elapsed, 0, 0, rawMD5), bytesPerIter, rawMD5, nil
}

type ffmpegRun struct {
	bytes int64
	md5   string
}

func runFFmpegOnce(bin string, args []string, rawOutput bool) (ffmpegRun, error) {
	cmd := exec.Command(bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if !rawOutput {
		cmd.Stdout = io.Discard
		if err := cmd.Run(); err != nil {
			return ffmpegRun{}, fmt.Errorf("%w: %s", err, stderr.String())
		}
		return ffmpegRun{}, nil
	}
	h := md5.New()
	counter := &countingWriter{w: h}
	cmd.Stdout = counter
	if err := cmd.Run(); err != nil {
		return ffmpegRun{}, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return ffmpegRun{bytes: counter.n, md5: hashString(h)}, nil
}

func ffmpegArgs(input string, rawOutput bool, threads string, pixFmt string) []string {
	args := []string{"-v", "error", "-nostdin"}
	if threads != "" {
		args = append(args, "-threads", threads)
	}
	args = append(args, "-i", input, "-an", "-sn", "-dn")
	if rawOutput {
		if pixFmt != "" {
			args = append(args, "-pix_fmt", pixFmt)
		}
		return append(args, "-f", "rawvideo", "-")
	}
	return append(args, "-f", "null", "-")
}

func sampleFromTotals(iters int, framesPerIter int, bytesPerIter int64, elapsed time.Duration, allocBytes uint64, allocs uint64, rawMD5 string) benchSample {
	totalFrames := framesPerIter * iters
	totalBytes := bytesPerIter * int64(iters)
	seconds := elapsed.Seconds()
	var fps float64
	if totalFrames > 0 && seconds > 0 {
		fps = float64(totalFrames) / seconds
	}
	var mibPerSec float64
	if totalBytes > 0 && seconds > 0 {
		mibPerSec = float64(totalBytes) / (1024 * 1024) / seconds
	}
	return benchSample{
		ElapsedMS:   float64(elapsed.Microseconds()) / 1000,
		TotalFrames: totalFrames,
		TotalBytes:  totalBytes,
		FPS:         fps,
		MiBPerSec:   mibPerSec,
		AllocBytes:  allocBytes,
		Allocs:      allocs,
		RawMD5:      rawMD5,
	}
}

func resultFromSamples(name string, input string, iters int, repeats int, warmup int, rawOutput bool, framesPerIter int, bytesPerIter int64, samples []benchSample, rawMD5 string, command string) benchResult {
	stats := sampleStats(samples)
	var totalFrames int
	var totalBytes int64
	var allocBytes uint64
	var allocs uint64
	var elapsedMS float64
	for _, sample := range samples {
		totalFrames += sample.TotalFrames
		totalBytes += sample.TotalBytes
		allocBytes += sample.AllocBytes
		allocs += sample.Allocs
		elapsedMS += sample.ElapsedMS
	}
	var fps float64
	if elapsedMS > 0 && totalFrames > 0 {
		fps = float64(totalFrames) / (elapsedMS / 1000)
	}
	var mibPerSec float64
	if elapsedMS > 0 && totalBytes > 0 {
		mibPerSec = float64(totalBytes) / (1024 * 1024) / (elapsedMS / 1000)
	}
	result := benchResult{
		Name:            name,
		Input:           input,
		Iterations:      iters,
		Repeats:         repeats,
		Warmup:          warmup,
		RawOutput:       rawOutput,
		FramesPerIter:   framesPerIter,
		BytesPerIter:    bytesPerIter,
		TotalFrames:     totalFrames,
		TotalBytes:      totalBytes,
		ElapsedMS:       elapsedMS,
		MeanElapsedMS:   stats.mean,
		MedianElapsedMS: stats.median,
		MinElapsedMS:    stats.min,
		MaxElapsedMS:    stats.max,
		StddevElapsedMS: stats.stddev,
		CVElapsed:       stats.cv,
		FPS:             fps,
		MiBPerSec:       mibPerSec,
		AllocBytes:      allocBytes,
		Allocs:          allocs,
		RawMD5:          rawMD5,
		Command:         command,
		Samples:         samples,
	}
	annotateBenchRates(&result)
	return result
}

func annotateBenchRates(result *benchResult) {
	if result == nil || result.ElapsedMS <= 0 {
		return
	}
	elapsedNS := result.ElapsedMS * 1e6
	if result.TotalFrames > 0 {
		result.NSPerFrame = elapsedNS / float64(result.TotalFrames)
	}
	if result.InputBytesPerIter > 0 && result.Iterations > 0 && result.Repeats > 0 {
		totalInputBytes := result.InputBytesPerIter * int64(result.Iterations*result.Repeats)
		if totalInputBytes > 0 {
			result.NSPerInputByte = elapsedNS / float64(totalInputBytes)
		}
	}
	if result.TotalBytes > 0 {
		result.NSPerRawByte = elapsedNS / float64(result.TotalBytes)
	}
}

type benchStats struct {
	mean   float64
	median float64
	min    float64
	max    float64
	stddev float64
	cv     float64
}

func sampleStats(samples []benchSample) benchStats {
	if len(samples) == 0 {
		return benchStats{}
	}
	values := make([]float64, len(samples))
	var sum float64
	for i, sample := range samples {
		values[i] = sample.ElapsedMS
		sum += sample.ElapsedMS
	}
	sort.Float64s(values)
	mean := sum / float64(len(values))
	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	}
	var variance float64
	for _, value := range values {
		diff := value - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	stddev := math.Sqrt(variance)
	var cv float64
	if mean != 0 {
		cv = stddev / mean
	}
	return benchStats{
		mean:   mean,
		median: median,
		min:    values[0],
		max:    values[len(values)-1],
		stddev: stddev,
		cv:     cv,
	}
}

func benchmarkMetadata(input string, data []byte, includeFFmpeg bool, ffmpegBin string) benchMetadata {
	sum := md5.Sum(data)
	revision, dirty := gitMetadata()
	modulePath, moduleVersion := moduleMetadata()
	meta := benchMetadata{
		Input:          input,
		InputBytes:     int64(len(data)),
		InputMD5:       hex.EncodeToString(sum[:]),
		GoVersion:      runtime.Version(),
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		NumCPU:         runtime.NumCPU(),
		GOMAXPROCS:     runtime.GOMAXPROCS(0),
		ModulePath:     modulePath,
		ModuleVersion:  moduleVersion,
		VCSRevision:    revision,
		VCSDirty:       dirty,
		ComparisonKind: "goh264-in-process",
	}
	if includeFFmpeg {
		meta.ComparisonKind = "goh264-in-process-vs-ffmpeg-cli"
		meta.FFmpegVersion = ffmpegVersion(ffmpegBin)
	}
	return meta
}

func moduleMetadata() (string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", ""
	}
	version := info.Main.Version
	if version == "(devel)" {
		version = ""
	}
	return info.Main.Path, version
}

func gitMetadata() (string, bool) {
	revOut, err := exec.Command("git", "rev-parse", "HEAD").Output()
	revision := ""
	if err == nil {
		revision = strings.TrimSpace(string(revOut))
	}
	statusOut, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return revision, false
	}
	return revision, strings.TrimSpace(string(statusOut)) != ""
}

func ffmpegVersion(bin string) string {
	out, err := exec.Command(bin, "-version").Output()
	if err != nil {
		return ""
	}
	line := strings.SplitN(string(out), "\n", 2)[0]
	return strings.TrimSpace(line)
}

type countingWriter struct {
	w hash.Hash
	n int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.n += int64(n)
	return n, err
}

func hashString(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}

func joinArgs(args []string) string {
	var b bytes.Buffer
	for i, arg := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%q", arg)
	}
	return b.String()
}

func die(where string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", where, err)
	os.Exit(1)
}
