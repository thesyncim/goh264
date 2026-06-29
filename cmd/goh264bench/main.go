// SPDX-License-Identifier: LGPL-2.1-or-later

package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	goh264 "github.com/thesyncim/goh264"
	h264internal "github.com/thesyncim/goh264/internal/h264"
)

type benchReport struct {
	Metadata benchMetadata `json:"metadata"`
	Results  []benchResult `json:"results"`
}

type benchMetadata struct {
	Input                  string  `json:"input"`
	InputBytes             int64   `json:"input_bytes"`
	InputMD5               string  `json:"input_md5"`
	CorpusManifest         string  `json:"corpus_manifest,omitempty"`
	FailureLedger          string  `json:"failure_ledger,omitempty"`
	CorpusFilter           string  `json:"corpus_filter,omitempty"`
	CorpusEntries          int     `json:"corpus_entries,omitempty"`
	CorpusSelected         int     `json:"corpus_selected_entries,omitempty"`
	CorpusDecodeOK         int     `json:"corpus_decode_ok_entries,omitempty"`
	CorpusGreen            int     `json:"corpus_green_entries,omitempty"`
	CorpusBench            int     `json:"corpus_benchmarked_entries,omitempty"`
	CorpusKnownRed         int     `json:"corpus_known_red_entries,omitempty"`
	CorpusStaleRed         int     `json:"corpus_stale_known_red_entries,omitempty"`
	CorpusSkipped          int     `json:"corpus_skipped_entries,omitempty"`
	CorpusNotTimed         int     `json:"corpus_not_timed_entries,omitempty"`
	FairnessPolicy         string  `json:"fairness_policy,omitempty"`
	GoVersion              string  `json:"go_version"`
	GOOS                   string  `json:"goos"`
	GOARCH                 string  `json:"goarch"`
	NumCPU                 int     `json:"num_cpu"`
	GOMAXPROCS             int     `json:"gomaxprocs"`
	ModulePath             string  `json:"module_path,omitempty"`
	ModuleVersion          string  `json:"module_version,omitempty"`
	VCSRevision            string  `json:"vcs_revision,omitempty"`
	VCSDirty               bool    `json:"vcs_dirty"`
	FFmpegVersion          string  `json:"ffmpeg_version,omitempty"`
	FFmpegCPUFlags         string  `json:"ffmpeg_cpuflags,omitempty"`
	ComparisonKind         string  `json:"comparison_kind"`
	ForbidGoAllocations    bool    `json:"forbid_go_allocations,omitempty"`
	MaxGoAllocBytesPerIter float64 `json:"max_go_alloc_bytes_per_iter,omitempty"`
	MaxGoAllocsPerIter     float64 `json:"max_go_allocs_per_iter,omitempty"`
}

type benchResult struct {
	Name                 string                 `json:"name"`
	EntryID              string                 `json:"entry_id,omitempty"`
	Input                string                 `json:"input"`
	Iterations           int                    `json:"iterations"`
	Repeats              int                    `json:"repeats"`
	Warmup               int                    `json:"warmup"`
	RawOutput            bool                   `json:"raw_output"`
	RawPixelFormat       string                 `json:"raw_pixel_format,omitempty"`
	FFmpegPixelFmt       string                 `json:"ffmpeg_pixel_format,omitempty"`
	FramesPerIter        int                    `json:"frames_per_iter,omitempty"`
	InputBytesPerIter    int64                  `json:"input_bytes_per_iter,omitempty"`
	BytesPerIter         int64                  `json:"bytes_per_iter,omitempty"`
	TotalFrames          int                    `json:"total_frames,omitempty"`
	TotalBytes           int64                  `json:"total_bytes,omitempty"`
	ElapsedMS            float64                `json:"elapsed_ms"`
	MeanElapsedMS        float64                `json:"mean_elapsed_ms,omitempty"`
	MedianElapsedMS      float64                `json:"median_elapsed_ms,omitempty"`
	MinElapsedMS         float64                `json:"min_elapsed_ms,omitempty"`
	MaxElapsedMS         float64                `json:"max_elapsed_ms,omitempty"`
	StddevElapsedMS      float64                `json:"stddev_elapsed_ms,omitempty"`
	CVElapsed            float64                `json:"cv_elapsed,omitempty"`
	FPS                  float64                `json:"fps,omitempty"`
	MiBPerSec            float64                `json:"mib_per_sec,omitempty"`
	NSPerFrame           float64                `json:"ns_per_frame,omitempty"`
	NSPerInputByte       float64                `json:"ns_per_input_byte,omitempty"`
	NSPerRawByte         float64                `json:"ns_per_raw_byte,omitempty"`
	AllocBytes           uint64                 `json:"alloc_bytes,omitempty"`
	Allocs               uint64                 `json:"allocs,omitempty"`
	AllocBytesPerIter    float64                `json:"alloc_bytes_per_iter,omitempty"`
	AllocsPerIter        float64                `json:"allocs_per_iter,omitempty"`
	AllocBytesPerFrame   float64                `json:"alloc_bytes_per_frame,omitempty"`
	AllocsPerFrame       float64                `json:"allocs_per_frame,omitempty"`
	RawMD5               string                 `json:"raw_md5,omitempty"`
	ExpectedRawMD5       string                 `json:"expected_raw_md5,omitempty"`
	ExpectedPixFmt       string                 `json:"expected_raw_pixel_format,omitempty"`
	ExpectedFrames       int                    `json:"expected_frames_per_iter,omitempty"`
	ExpectedBytes        int64                  `json:"expected_bytes_per_iter,omitempty"`
	ParityStatus         string                 `json:"parity_status,omitempty"`
	QualityStatus        string                 `json:"quality_status,omitempty"`
	QualityMetric        string                 `json:"quality_metric,omitempty"`
	QualityReference     string                 `json:"quality_reference,omitempty"`
	PeerQualityStatus    string                 `json:"peer_quality_status,omitempty"`
	PeerQualityMetric    string                 `json:"peer_quality_metric,omitempty"`
	PeerQualityReference string                 `json:"peer_quality_reference,omitempty"`
	ErrorClass           string                 `json:"error_class,omitempty"`
	Surfaces             []string               `json:"surfaces,omitempty"`
	FeatureTags          []string               `json:"feature_tags,omitempty"`
	Source               string                 `json:"source,omitempty"`
	Command              string                 `json:"command,omitempty"`
	ProcessPerIter       bool                   `json:"process_per_iter"`
	InputReadTimed       bool                   `json:"input_read_timed"`
	StdoutPipeTimed      bool                   `json:"stdout_pipe_timed"`
	BaselineKind         string                 `json:"baseline_kind"`
	BackendKind          string                 `json:"backend_kind,omitempty"`
	CPUFlags             string                 `json:"cpu_flags,omitempty"`
	ComparisonLane       string                 `json:"comparison_lane,omitempty"`
	Skipped              bool                   `json:"skipped,omitempty"`
	Error                string                 `json:"error,omitempty"`
	Notes                []string               `json:"notes,omitempty"`
	Samples              []benchSample          `json:"samples,omitempty"`
	FrameDiagnostics     []benchFrameDiagnostic `json:"frame_diagnostics,omitempty"`
}

type benchSample struct {
	ElapsedMS         float64 `json:"elapsed_ms"`
	TotalFrames       int     `json:"total_frames,omitempty"`
	TotalBytes        int64   `json:"total_bytes,omitempty"`
	FPS               float64 `json:"fps,omitempty"`
	MiBPerSec         float64 `json:"mib_per_sec,omitempty"`
	AllocBytes        uint64  `json:"alloc_bytes,omitempty"`
	Allocs            uint64  `json:"allocs,omitempty"`
	AllocBytesPerIter float64 `json:"alloc_bytes_per_iter,omitempty"`
	AllocsPerIter     float64 `json:"allocs_per_iter,omitempty"`
	RawMD5            string  `json:"raw_md5,omitempty"`
}

type benchFrameDiagnostic struct {
	Index                  int    `json:"index"`
	RawPixelFormat         string `json:"raw_pixel_format,omitempty"`
	ExpectedRawPixelFormat string `json:"expected_raw_pixel_format,omitempty"`
	Bytes                  int64  `json:"bytes,omitempty"`
	ExpectedBytes          int64  `json:"expected_bytes,omitempty"`
	RawMD5                 string `json:"raw_md5,omitempty"`
	ExpectedRawMD5         string `json:"expected_raw_md5,omitempty"`
	ParityStatus           string `json:"parity_status,omitempty"`
}

type benchOptions struct {
	iters                  int
	repeats                int
	warmup                 int
	rawOutput              bool
	runFFmpeg              bool
	ffmpegBin              string
	ffmpegThreads          string
	ffmpegCPUFlags         string
	ffmpegPixFmt           string
	ffmpegProcessPerIter   bool
	fairCPULanes           bool
	strictPixFmt           bool
	corpusFilter           string
	failureLedger          string
	annexBInput            bool
	diagnose               bool
	forbidGoAllocations    bool
	maxGoAllocBytesPerIter float64
	maxGoAllocsPerIter     float64
}

type ffmpegBenchLane struct {
	name           string
	backendKind    string
	cpuFlags       string
	comparisonLane string
}

type benchCorpusEntry struct {
	ID            string             `json:"id"`
	Path          string             `json:"path"`
	URL           string             `json:"url,omitempty"`
	Format        string             `json:"format"`
	Expect        string             `json:"expect"`
	ExpectedError string             `json:"expected_error,omitempty"`
	PixFmt        string             `json:"pix_fmt,omitempty"`
	FrameCount    int                `json:"frame_count,omitempty"`
	FrameSize     int                `json:"frame_size,omitempty"`
	SourceMD5     string             `json:"source_md5,omitempty"`
	BitstreamMD5  string             `json:"bitstream_md5,omitempty"`
	RawVideoMD5   string             `json:"rawvideo_md5,omitempty"`
	Extract       string             `json:"extract,omitempty"`
	FrameMD5      []string           `json:"frame_md5,omitempty"`
	Surfaces      []string           `json:"surfaces,omitempty"`
	GuardTags     []string           `json:"guard_tags,omitempty"`
	FeatureTags   []string           `json:"feature_tags,omitempty"`
	Source        string             `json:"source,omitempty"`
	KnownFailure  *benchKnownFailure `json:"known_failure,omitempty"`
}

type benchKnownFailure struct {
	Class          string `json:"class"`
	DetailContains string `json:"detail_contains"`
}

func main() {
	input := flag.String("input", "", "H.264 input file")
	manifest := flag.String("manifest", "", "JSONL H.264 corpus manifest; benchmarks decode-ok entries after oracle parity validation")
	maxEntries := flag.Int("max-entries", 0, "maximum decode-ok manifest entries to benchmark; 0 means all")
	corpusFilter := flag.String("filter", os.Getenv("GOH264_CORPUS_FILTER"), "comma/space-separated manifest entry filter; defaults to GOH264_CORPUS_FILTER")
	failureLedger := flag.String("failure-ledger", "auto", "manifest known-red ledger: auto uses failures.jsonl next to the manifest when present, off disables it, otherwise pass a JSONL path")
	diagnose := flag.Bool("diagnose", false, "manifest mode: run oracle diagnostics instead of timing samples; includes per-frame raw MD5s when frames decode")
	iters := flag.Int("iters", 5, "measured iterations")
	repeats := flag.Int("repeats", 1, "measured repeat samples; each sample runs -iters decodes")
	warmup := flag.Int("warmup", 1, "warmup iterations")
	rawOutput := flag.Bool("raw", true, "materialize raw decoded bytes during Go and FFmpeg runs")
	runFFmpeg := flag.Bool("ffmpeg", false, "also run an FFmpeg baseline over the same file")
	ffmpegBin := flag.String("ffmpeg-bin", "ffmpeg", "FFmpeg binary")
	ffmpegThreads := flag.String("ffmpeg-threads", "1", "FFmpeg -threads value")
	ffmpegCPUFlags := flag.String("ffmpeg-cpuflags", "", "FFmpeg -cpuflags value; empty uses the binary default native C+asm CPU dispatch, 0 forces pure C")
	ffmpegPureC := flag.Bool("ffmpeg-pure-c", false, "shorthand for -ffmpeg-cpuflags 0")
	fairCPULanes := flag.Bool("fair-cpu-lanes", false, "with -ffmpeg, emit explicit pure-C-vs-pure-Go and native-C+asm-vs-Go+asm backend lanes")
	ffmpegProcessPerIter := flag.Bool("ffmpeg-process-per-iter", false, "with -ffmpeg, run one FFmpeg process per timed iteration; default amortizes one FFmpeg process per repeat sample over a prebuilt repeated input")
	ffmpegPixFmt := flag.String("ffmpeg-pix-fmt", "", "FFmpeg output pixel format for -raw mode; defaults to Go raw pixel format when available")
	strictPixFmt := flag.Bool("strict-pix-fmt", false, "reject a user-supplied -ffmpeg-pix-fmt that differs from Go raw pixel format")
	forbidGoAllocations := flag.Bool("forbid-go-allocations", false, "fail if a measured Go benchmark lane allocates during timed iterations")
	maxGoAllocBytesPerIter := flag.Float64("max-go-alloc-bytes-per-iter", 0, "fail if a timed Go result exceeds this alloc_bytes_per_iter budget; 0 disables")
	maxGoAllocsPerIter := flag.Float64("max-go-allocs-per-iter", 0, "fail if a timed Go result exceeds this allocs_per_iter budget; 0 disables")
	cpuProfile := flag.String("cpuprofile", "", "write Go CPU profile for the benchmark run")
	memProfile := flag.String("memprofile", "", "write Go heap profile after the benchmark run")
	jsonOut := flag.Bool("json", false, "print JSON")
	flag.Parse()

	if (*input == "") == (*manifest == "") || *iters <= 0 || *repeats <= 0 || *warmup < 0 || *maxEntries < 0 ||
		*maxGoAllocBytesPerIter < 0 || *maxGoAllocsPerIter < 0 {
		fmt.Fprintln(os.Stderr, "usage: goh264bench (-input file.h264 | -manifest corpus.jsonl) [-iters 5] [-repeats 1] [-warmup 1] [-ffmpeg] [-json]")
		os.Exit(2)
	}
	if *ffmpegPureC && *fairCPULanes {
		fmt.Fprintln(os.Stderr, "-ffmpeg-pure-c and -fair-cpu-lanes are mutually exclusive; fair lanes already include FFmpeg -cpuflags 0")
		os.Exit(2)
	}
	if *ffmpegPureC {
		*ffmpegCPUFlags = "0"
	}
	opts := benchOptions{
		iters:                  *iters,
		repeats:                *repeats,
		warmup:                 *warmup,
		rawOutput:              *rawOutput,
		runFFmpeg:              *runFFmpeg,
		ffmpegBin:              *ffmpegBin,
		ffmpegThreads:          *ffmpegThreads,
		ffmpegCPUFlags:         *ffmpegCPUFlags,
		ffmpegPixFmt:           *ffmpegPixFmt,
		ffmpegProcessPerIter:   *ffmpegProcessPerIter,
		fairCPULanes:           *fairCPULanes,
		strictPixFmt:           *strictPixFmt,
		corpusFilter:           *corpusFilter,
		failureLedger:          *failureLedger,
		diagnose:               *diagnose,
		forbidGoAllocations:    *forbidGoAllocations,
		maxGoAllocBytesPerIter: *maxGoAllocBytesPerIter,
		maxGoAllocsPerIter:     *maxGoAllocsPerIter,
	}
	profiles, err := startBenchProfiles(*cpuProfile, *memProfile)
	if err != nil {
		die("profile", err)
	}
	report, err := buildBenchReport(*input, *manifest, *maxEntries, opts)
	if err != nil {
		closeBenchProfilesBeforeExit(profiles)
		die("benchmark", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			closeBenchProfilesBeforeExit(profiles)
			die("json", err)
		}
		if err := profiles.Close(); err != nil {
			die("profile", err)
		}
		return
	}
	fmt.Printf("input: %s, %d bytes, md5 %s\n", report.Metadata.Input, report.Metadata.InputBytes, report.Metadata.InputMD5)
	if report.Metadata.CorpusManifest != "" {
		fmt.Printf("corpus: selected %d, decode-ok %d, green %d, benchmarked %d, known-red %d, stale-known-red %d, skipped %d, not-timed %d\n",
			report.Metadata.CorpusSelected, report.Metadata.CorpusDecodeOK, report.Metadata.CorpusGreen,
			report.Metadata.CorpusBench, report.Metadata.CorpusKnownRed, report.Metadata.CorpusStaleRed,
			report.Metadata.CorpusSkipped, report.Metadata.CorpusNotTimed)
		if report.Metadata.FailureLedger != "" {
			fmt.Printf("failure ledger: %s\n", report.Metadata.FailureLedger)
		}
	}
	for _, r := range report.Results {
		if r.Skipped {
			fmt.Printf("%s: skipped", r.Name)
			if r.EntryID != "" {
				fmt.Printf(", entry %s", r.EntryID)
			}
			if r.QualityStatus != "" {
				fmt.Printf(", quality %s", r.QualityStatus)
				if r.QualityReference != "" {
					fmt.Printf(" vs %s", r.QualityReference)
				}
			}
			if r.PeerQualityStatus != "" {
				fmt.Printf(", peer quality %s", r.PeerQualityStatus)
				if r.PeerQualityReference != "" {
					fmt.Printf(" vs %s", r.PeerQualityReference)
				}
			}
			if r.ParityStatus != "" && r.ParityStatus != r.QualityStatus {
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
		if r.BackendKind != "" {
			fmt.Printf(", backend %s", r.BackendKind)
		}
		if r.CPUFlags != "" {
			fmt.Printf(", cpuflags %s", r.CPUFlags)
		}
		if r.ComparisonLane != "" {
			fmt.Printf(", lane %s", r.ComparisonLane)
		}
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
		if r.AllocsPerIter > 0 || r.AllocBytesPerIter > 0 {
			fmt.Printf(", %.2f allocs/iter, %.2f MiB alloc/iter",
				r.AllocsPerIter,
				r.AllocBytesPerIter/(1024*1024))
		}
		if r.RawMD5 != "" {
			fmt.Printf(", raw md5 %s", r.RawMD5)
		}
		if r.QualityStatus != "" {
			fmt.Printf(", quality %s", r.QualityStatus)
			if r.QualityReference != "" {
				fmt.Printf(" vs %s", r.QualityReference)
			}
		}
		if r.PeerQualityStatus != "" {
			fmt.Printf(", peer quality %s", r.PeerQualityStatus)
			if r.PeerQualityReference != "" {
				fmt.Printf(" vs %s", r.PeerQualityReference)
			}
		}
		if r.ParityStatus != "" && r.ParityStatus != r.QualityStatus {
			fmt.Printf(", parity %s", r.ParityStatus)
		}
		if r.Command != "" {
			fmt.Printf("\n  %s", r.Command)
		}
		for _, note := range r.Notes {
			fmt.Printf("\n  note: %s", note)
		}
		for _, frame := range r.FrameDiagnostics {
			printBenchFrameDiagnostic(frame)
		}
		fmt.Println()
	}
	if err := profiles.Close(); err != nil {
		die("profile", err)
	}
}

type benchProfiles struct {
	cpuFile *os.File
	memPath string
	closed  bool
}

func startBenchProfiles(cpuPath string, memPath string) (*benchProfiles, error) {
	profiles := &benchProfiles{memPath: memPath}
	if cpuPath == "" {
		return profiles, nil
	}
	f, err := os.Create(cpuPath)
	if err != nil {
		return nil, fmt.Errorf("create CPU profile: %w", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		closeErr := f.Close()
		return nil, errors.Join(fmt.Errorf("start CPU profile: %w", err), closeErr)
	}
	profiles.cpuFile = f
	return profiles, nil
}

func (p *benchProfiles) Close() error {
	if p == nil || p.closed {
		return nil
	}
	p.closed = true
	var errs []error
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		if err := p.cpuFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close CPU profile: %w", err))
		}
	}
	if p.memPath != "" {
		runtime.GC()
		f, err := os.Create(p.memPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("create heap profile: %w", err))
		} else {
			if err := pprof.WriteHeapProfile(f); err != nil {
				errs = append(errs, fmt.Errorf("write heap profile: %w", err))
			}
			if err := f.Close(); err != nil {
				errs = append(errs, fmt.Errorf("close heap profile: %w", err))
			}
		}
	}
	return errors.Join(errs...)
}

func closeBenchProfilesBeforeExit(profiles *benchProfiles) {
	if err := profiles.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "profile: %v\n", err)
	}
}

func printBenchFrameDiagnostic(frame benchFrameDiagnostic) {
	fmt.Printf("\n  frame[%d]:", frame.Index)
	if frame.RawMD5 != "" {
		fmt.Printf(" md5 %s", frame.RawMD5)
	}
	if frame.ExpectedRawMD5 != "" {
		fmt.Printf(" want %s", frame.ExpectedRawMD5)
	}
	if frame.RawPixelFormat != "" {
		fmt.Printf(" pix_fmt %s", frame.RawPixelFormat)
	}
	if frame.ExpectedRawPixelFormat != "" && frame.ExpectedRawPixelFormat != frame.RawPixelFormat {
		fmt.Printf(" want_pix_fmt %s", frame.ExpectedRawPixelFormat)
	}
	if frame.Bytes != 0 || frame.ExpectedBytes != 0 {
		fmt.Printf(" bytes %d", frame.Bytes)
		if frame.ExpectedBytes != 0 && frame.ExpectedBytes != frame.Bytes {
			fmt.Printf(" want %d", frame.ExpectedBytes)
		}
	}
	if frame.ParityStatus != "" {
		fmt.Printf(" parity %s", frame.ParityStatus)
	}
}

func buildBenchReport(input string, manifest string, maxEntries int, opts benchOptions) (benchReport, error) {
	var report benchReport
	if manifest != "" {
		report, err := benchManifest(manifest, maxEntries, opts)
		if err != nil {
			return benchReport{}, err
		}
		annotateBenchReportQuality(&report)
		if err := enforceBenchAllocationBudgets(report, opts); err != nil {
			return benchReport{}, err
		}
		return report, nil
	}
	data, err := os.ReadFile(input)
	if err != nil {
		return benchReport{}, fmt.Errorf("read input: %w", err)
	}
	results, err := benchOneInput(input, data, opts)
	if err != nil {
		return benchReport{}, err
	}
	report = benchReport{
		Metadata: benchmarkMetadata(input, data, opts),
		Results:  results,
	}
	annotateBenchReportQuality(&report)
	if err := enforceBenchAllocationBudgets(report, opts); err != nil {
		return benchReport{}, err
	}
	return report, nil
}

func enforceBenchAllocationBudgets(report benchReport, opts benchOptions) error {
	if !opts.forbidGoAllocations && opts.maxGoAllocBytesPerIter <= 0 && opts.maxGoAllocsPerIter <= 0 {
		return nil
	}
	var failures []string
	for _, result := range report.Results {
		if !benchResultHasGoAllocationMetrics(result) {
			continue
		}
		prefix := result.Name
		if result.EntryID != "" {
			prefix += " " + result.EntryID
		}
		if opts.forbidGoAllocations {
			if result.AllocBytesPerIter != 0 {
				failures = append(failures, fmt.Sprintf("%s: alloc_bytes_per_iter %.2f exceeds zero-allocation policy",
					prefix, result.AllocBytesPerIter))
			}
			if result.AllocsPerIter != 0 {
				failures = append(failures, fmt.Sprintf("%s: allocs_per_iter %.2f exceeds zero-allocation policy",
					prefix, result.AllocsPerIter))
			}
		}
		if opts.maxGoAllocBytesPerIter > 0 && result.AllocBytesPerIter > opts.maxGoAllocBytesPerIter {
			failures = append(failures, fmt.Sprintf("%s: alloc_bytes_per_iter %.2f exceeds budget %.2f",
				prefix, result.AllocBytesPerIter, opts.maxGoAllocBytesPerIter))
		}
		if opts.maxGoAllocsPerIter > 0 && result.AllocsPerIter > opts.maxGoAllocsPerIter {
			failures = append(failures, fmt.Sprintf("%s: allocs_per_iter %.2f exceeds budget %.2f",
				prefix, result.AllocsPerIter, opts.maxGoAllocsPerIter))
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("Go allocation budget exceeded:\n%s", strings.Join(failures, "\n"))
	}
	return nil
}

func benchResultHasGoAllocationMetrics(result benchResult) bool {
	return !result.Skipped &&
		result.Name == "goh264" &&
		result.BaselineKind == "in-process-go" &&
		!result.ProcessPerIter &&
		result.Iterations > 0 &&
		result.Repeats > 0
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
		for _, lane := range ffmpegBenchLanes(opts) {
			ffmpegResult, err := benchFFmpeg(input, int64(len(data)), opts.iters, opts.repeats, opts.warmup, opts.rawOutput, opts.ffmpegBin, opts.ffmpegThreads, opts.ffmpegPixFmt, goResult.RawPixelFormat, opts.ffmpegProcessPerIter, lane)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", lane.name, err)
			}
			annotateFFmpegPeerQuality(&ffmpegResult, goResult)
			results = append(results, ffmpegResult)
		}
	}
	return results, nil
}

func benchManifest(path string, maxEntries int, opts benchOptions) (benchReport, error) {
	if opts.diagnose {
		return diagnoseBenchManifest(path, maxEntries, opts)
	}
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
	var decodeOKSelected int
	var green int
	var benchmarked int
	var knownRed int
	var staleKnownRed int
	var skipped int
	for _, entry := range entries {
		if entry.Expect != "decode-ok" {
			results = append(results, skippedBenchResult(entry, "manifest row is not a decode-ok oracle row and is not a timing sample"))
			skipped++
			continue
		}
		decodeOKSelected++
		if err := validateBenchCorpusEntry(entry); err != nil {
			return benchReport{}, err
		}
		inputPath, err := resolveBenchCorpusPath(baseDir, entry)
		if err != nil {
			if failure, ok := failureLedger[entry.ID]; ok {
				results = append(results, knownRedBenchResult(failure, "", nil, err, failureLedgerPath))
				knownRed++
				skipped++
				continue
			}
			return benchReport{}, err
		}
		data, err := os.ReadFile(inputPath)
		if err != nil {
			if failure, ok := failureLedger[entry.ID]; ok {
				results = append(results, knownRedBenchResult(failure, inputPath, nil, err, failureLedgerPath))
				knownRed++
				skipped++
				continue
			}
			return benchReport{}, fmt.Errorf("%s: read input: %w", entry.ID, err)
		}
		if err := validateBenchBitstreamMD5(entry, data); err != nil {
			return benchReport{}, err
		}
		if err := preflightBenchGoOracle(inputPath, data, entry); err != nil {
			if failure, ok := failureLedger[entry.ID]; ok {
				results = append(results, knownRedBenchResult(failure, inputPath, data, err, failureLedgerPath))
				knownRed++
				skipped++
				continue
			}
			return benchReport{}, fmt.Errorf("%s: goh264 oracle preflight: %w", entry.ID, err)
		} else if _, ok := failureLedger[entry.ID]; ok {
			results = append(results, staleKnownRedBenchResult(failureLedger[entry.ID], inputPath, data, failureLedgerPath))
			staleKnownRed++
			skipped++
			continue
		}
		green++
		if opts.runFFmpeg {
			for _, lane := range ffmpegBenchLanes(opts) {
				if err := preflightBenchFFmpegOracle(inputPath, entry, opts, lane); err != nil {
					return benchReport{}, fmt.Errorf("%s: %s oracle preflight: %w", entry.ID, lane.name, err)
				}
			}
		}
		if maxEntries > 0 && benchmarked >= maxEntries {
			results = append(results, greenNotTimedBenchResult(entry, inputPath, data, maxEntries))
			skipped++
			continue
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
		}
		results = append(results, entryResults...)
		benchmarked++
	}
	if len(results) == 0 {
		return benchReport{}, fmt.Errorf("%s: no manifest entries selected", path)
	}

	meta := benchmarkMetadata(path, manifestData, opts)
	meta.CorpusManifest = path
	meta.FailureLedger = failureLedgerPath
	meta.CorpusFilter = opts.corpusFilter
	meta.CorpusEntries = len(entries)
	meta.CorpusSelected = len(entries)
	meta.CorpusDecodeOK = decodeOKSelected
	meta.CorpusGreen = green
	meta.CorpusBench = benchmarked
	meta.CorpusKnownRed = knownRed
	meta.CorpusStaleRed = staleKnownRed
	meta.CorpusSkipped = skipped
	meta.CorpusNotTimed = len(entries) - benchmarked
	meta.ComparisonKind = "manifest-goh264-in-process"
	if opts.runFFmpeg {
		meta.ComparisonKind = "manifest-goh264-in-process-vs-ffmpeg-cli-amortized"
		if opts.ffmpegProcessPerIter {
			meta.ComparisonKind = "manifest-goh264-in-process-vs-ffmpeg-cli-process-per-iter"
		}
		if opts.fairCPULanes {
			meta.ComparisonKind += "-fair-cpu-lanes"
		}
	}
	meta.FairnessPolicy = "Decode-ok corpus entries are benchmarked only after bitstream MD5, Go raw pixel format, frame count, raw byte count, and concatenated rawvideo MD5 pass a preflight against the manifest oracle; manifest rows use their declared input format for the Go decoder path. Known-red ledger rows and stale known-red rows are emitted as skipped results with the exact error or stale-ledger note and are not timing samples. -max-entries limits timed green rows only; selected rows beyond that limit remain visible as rawvideo-md5-ok-not-timed skips. Optional FFmpeg CLI rawvideo output must pass the same rawvideo MD5 preflight before measured FFmpeg samples run; fair CPU lanes label each FFmpeg CPU mode against the actual measured Go backend_kind instead of assuming a purego or assembly build. Primary quality_status is the manifest rawvideo oracle when available; peer_quality_status records each FFmpeg lane's rawvideo match or mismatch against the measured Go lane. Go result backend_kind remains explicit, so purego builds report go-pure and default builds with partial assembly report go-partial-asm until all decoder kernels are ported. FFmpeg timing defaults to one CLI process per repeat sample over a prebuilt repeated input file, amortizing process startup and CLI setup across timed iterations; raw-output amortized samples must also match the single-iteration raw output repeated for every timed iteration; -ffmpeg-process-per-iter restores the historical process-per-iteration baseline."
	return benchReport{Metadata: meta, Results: results}, nil
}

func diagnoseBenchManifest(path string, maxEntries int, opts benchOptions) (benchReport, error) {
	if !opts.rawOutput {
		return benchReport{}, fmt.Errorf("manifest diagnostic mode requires -raw=true so per-frame rawvideo diagnostics are available")
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
	var decodeOKSelected int
	var green int
	var diagnosed int
	var knownRed int
	var staleKnownRed int
	var skipped int
	for _, entry := range entries {
		if !benchDiagnosticOracleEntry(entry) {
			results = append(results, skippedBenchResult(entry, "manifest row is not a diagnostic oracle row"))
			skipped++
			continue
		}
		if entry.Expect == "decode-ok" {
			decodeOKSelected++
		}
		if err := validateBenchDiagnosticCorpusEntry(entry); err != nil {
			return benchReport{}, err
		}
		if maxEntries > 0 && diagnosed >= maxEntries {
			results = append(results, skippedBenchResult(entry, fmt.Sprintf("not diagnosed because -max-entries=%d was reached", maxEntries)))
			results[len(results)-1].ParityStatus = "max-entries-not-diagnosed"
			results[len(results)-1].BaselineKind = "oracle-diagnostic-not-run"
			skipped++
			continue
		}
		diagnosed++
		failure, isKnownRed := failureLedger[entry.ID]
		result := diagnoseBenchEntry(baseDir, entry)
		if isKnownRed {
			applyKnownRedDiagnostic(&result, failure, failureLedgerPath)
			if result.ParityStatus == "rawvideo-md5-ok-failure-ledger-stale" {
				staleKnownRed++
			} else {
				knownRed++
			}
			skipped++
		} else if !benchDiagnosticOraclePassed(result) {
			result.Notes = append(result.Notes, "unexpected manifest oracle failure")
		} else {
			green++
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		return benchReport{}, fmt.Errorf("%s: no manifest entries selected", path)
	}

	meta := benchmarkMetadata(path, manifestData, opts)
	meta.CorpusManifest = path
	meta.FailureLedger = failureLedgerPath
	meta.CorpusFilter = opts.corpusFilter
	meta.CorpusEntries = len(entries)
	meta.CorpusSelected = len(entries)
	meta.CorpusDecodeOK = decodeOKSelected
	meta.CorpusGreen = green
	meta.CorpusBench = 0
	meta.CorpusKnownRed = knownRed
	meta.CorpusStaleRed = staleKnownRed
	meta.CorpusSkipped = skipped
	meta.CorpusNotTimed = len(entries)
	meta.ComparisonKind = "manifest-goh264-oracle-diagnostic"
	meta.FairnessPolicy = "Manifest diagnostics run selected decode-ok and expected decode-error rows once from the existing manifest/cache. Decode-ok rows compare Go raw pixel format, frame count, raw byte count, rawvideo MD5, and per-frame raw MD5s when frames decode; expected decode-error rows require the decoder error to contain expected_error. Known-red ledger rows remain marked known-red unless the decoder output actually matches the oracle, in which case they are reported as failure-ledger stale rather than green timing samples."
	return benchReport{Metadata: meta, Results: results}, nil
}

func diagnoseBenchEntry(baseDir string, entry benchCorpusEntry) benchResult {
	result := benchResult{
		Name:            "goh264",
		EntryID:         entry.ID,
		Input:           entry.Path,
		RawOutput:       true,
		ExpectedRawMD5:  entry.RawVideoMD5,
		ExpectedPixFmt:  entry.PixFmt,
		ExpectedFrames:  entry.FrameCount,
		ExpectedBytes:   int64(entry.FrameCount * entry.FrameSize),
		Surfaces:        append([]string(nil), entry.Surfaces...),
		FeatureTags:     append([]string(nil), entry.FeatureTags...),
		Source:          entry.Source,
		ProcessPerIter:  false,
		InputReadTimed:  false,
		StdoutPipeTimed: false,
		BaselineKind:    "oracle-diagnostic",
	}
	inputPath, err := resolveBenchCorpusPath(baseDir, entry)
	if err != nil {
		applyBenchDiagnosticError(&result, err)
		return result
	}
	result.Input = inputPath
	data, err := os.ReadFile(inputPath)
	if err != nil {
		applyBenchDiagnosticError(&result, fmt.Errorf("%s: read input: %w", entry.ID, err))
		return result
	}
	result.InputBytesPerIter = int64(len(data))
	if err := validateBenchBitstreamMD5(entry, data); err != nil {
		applyBenchDiagnosticError(&result, err)
		return result
	}
	if entry.Expect == "decode-error" {
		diagnoseBenchExpectedDecodeError(&result, entry, data)
		return result
	}
	run, err := decodeGoOnceForFormat(data, true, entry.Format == "annexb")
	if err != nil {
		applyBenchDiagnosticError(&result, fmt.Errorf("decode: %w", err))
		return result
	}
	result.RawPixelFormat = run.pixFmt
	result.FramesPerIter = run.frames
	result.BytesPerIter = run.bytes
	result.RawMD5 = run.md5
	result.FrameDiagnostics = append([]benchFrameDiagnostic(nil), run.frameDiagnostics...)
	annotateBenchFrameDiagnostics(&result, entry)
	if detail := benchOracleMismatchDetail(result, entry); detail != "" {
		applyBenchDiagnosticError(&result, errors.New(detail))
		return result
	}
	result.ParityStatus = "rawvideo-md5-ok"
	return result
}

func diagnoseBenchExpectedDecodeError(result *benchResult, entry benchCorpusEntry, data []byte) {
	run, err := decodeGoOnceForFormat(data, true, entry.Format == "annexb")
	if err == nil {
		result.RawPixelFormat = run.pixFmt
		result.FramesPerIter = run.frames
		result.BytesPerIter = run.bytes
		result.RawMD5 = run.md5
		result.FrameDiagnostics = append([]benchFrameDiagnostic(nil), run.frameDiagnostics...)
		applyBenchDiagnosticError(result, fmt.Errorf("decode succeeded with %d frames, want error containing %q", run.frames, entry.ExpectedError))
		return
	}
	result.Error = err.Error()
	result.ErrorClass = benchOracleFailureClass("decode: " + err.Error())
	if !benchExpectedDecodeErrorMatches(entry, err) {
		result.ParityStatus = result.ErrorClass
		result.Notes = append(result.Notes, fmt.Sprintf("expected decode error containing %q", entry.ExpectedError))
		return
	}
	result.ParityStatus = "decode-error-ok"
	result.Notes = append(result.Notes, fmt.Sprintf("matched expected decode error containing %q", entry.ExpectedError))
}

func applyBenchDiagnosticError(result *benchResult, err error) {
	if result == nil || err == nil {
		return
	}
	result.Error = err.Error()
	result.ErrorClass = benchOracleFailureClass(err.Error())
	result.ParityStatus = result.ErrorClass
}

func benchDiagnosticOracleEntry(entry benchCorpusEntry) bool {
	return entry.Expect == "decode-ok" || entry.Expect == "decode-error"
}

func benchDiagnosticOraclePassed(result benchResult) bool {
	return result.ParityStatus == "rawvideo-md5-ok" || result.ParityStatus == "decode-error-ok"
}

func benchExpectedDecodeErrorMatches(entry benchCorpusEntry, err error) bool {
	if err == nil {
		return false
	}
	want := strings.ToLower(entry.ExpectedError)
	return want == "" || strings.Contains(strings.ToLower(err.Error()), want)
}

func applyKnownRedDiagnostic(result *benchResult, failure benchCorpusEntry, ledgerPath string) {
	if result == nil {
		return
	}
	result.Skipped = true
	result.BaselineKind = "oracle-known-red-diagnostic"
	if ledgerPath != "" {
		result.Notes = append(result.Notes, "failure ledger: "+ledgerPath)
	}
	if failure.KnownFailure != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("expected current failure: class=%s contains=%q", failure.KnownFailure.Class, failure.KnownFailure.DetailContains))
	}
	if benchDiagnosticOraclePassed(*result) {
		result.ParityStatus = "rawvideo-md5-ok-failure-ledger-stale"
		result.Notes = append(result.Notes,
			fmt.Sprintf("entry is still listed in %s but passed Go oracle diagnostics; update the failure ledger before using this as a green benchmark lane", ledgerPath),
		)
		return
	}
	if !benchKnownFailureStillCurrent(failure, result.Error) {
		result.ParityStatus = "known-red-signature-drift"
		result.Notes = append(result.Notes, fmt.Sprintf("current failure signature drifted: class=%s detail=%q", result.ErrorClass, result.Error))
		return
	}
	result.ParityStatus = "known-red"
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
		if err := validateBenchFailureLedgerEntry(failure); err != nil {
			return nil, "", fmt.Errorf("%s: failure-ledger row: %w", failure.ID, err)
		}
		if err := validateBenchKnownFailure(failure); err != nil {
			return nil, "", fmt.Errorf("%s: failure-ledger row: %w", failure.ID, err)
		}
		if _, ok := failures[failure.ID]; ok {
			return nil, "", fmt.Errorf("%s: duplicate failure-ledger id in %s", failure.ID, path)
		}
		manifestEntry, ok := manifestByID[failure.ID]
		if !ok {
			return nil, "", fmt.Errorf("%s: failure-ledger row missing from %s", failure.ID, manifestPath)
		}
		if !reflect.DeepEqual(benchCorpusEntryWithoutKnownFailure(failure), benchCorpusEntryWithoutKnownFailure(manifestEntry)) {
			return nil, "", fmt.Errorf("%s: failure-ledger row drifted from %s", failure.ID, manifestPath)
		}
		failures[failure.ID] = failure
	}
	return failures, path, nil
}

func benchCorpusEntryWithoutKnownFailure(entry benchCorpusEntry) benchCorpusEntry {
	entry.KnownFailure = nil
	return entry
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
		entry.ExpectedError,
		entry.PixFmt,
		entry.SourceMD5,
		entry.Extract,
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
	if err := validateBenchCorpusCommon(entry); err != nil {
		return err
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

func validateBenchDiagnosticCorpusEntry(entry benchCorpusEntry) error {
	if err := validateBenchCorpusCommon(entry); err != nil {
		return err
	}
	switch entry.Expect {
	case "decode-ok":
		return validateBenchCorpusEntry(entry)
	case "decode-error":
		if entry.BitstreamMD5 == "" || entry.ExpectedError == "" {
			return fmt.Errorf("%s: decode-error entries need bitstream_md5 and expected_error", entry.ID)
		}
	default:
		return fmt.Errorf("%s: diagnostic manifest mode only runs decode-ok and decode-error entries, got %q", entry.ID, entry.Expect)
	}
	return nil
}

func validateBenchFailureLedgerEntry(entry benchCorpusEntry) error {
	if err := validateBenchCorpusCommon(entry); err != nil {
		return err
	}
	switch entry.Expect {
	case "decode-ok":
		if entry.BitstreamMD5 == "" || entry.RawVideoMD5 == "" || entry.PixFmt == "" {
			return fmt.Errorf("%s: decode-ok entries need bitstream_md5, rawvideo_md5, and pix_fmt", entry.ID)
		}
		if entry.FrameCount <= 0 || entry.FrameSize <= 0 {
			return fmt.Errorf("%s: frame_count/frame_size must be positive", entry.ID)
		}
		if len(entry.FrameMD5) != 0 && len(entry.FrameMD5) != entry.FrameCount {
			return fmt.Errorf("%s: frame_md5 count = %d, want 0 or %d", entry.ID, len(entry.FrameMD5), entry.FrameCount)
		}
	case "metadata-ok":
		if entry.BitstreamMD5 == "" {
			return fmt.Errorf("%s: metadata-ok entries need bitstream_md5", entry.ID)
		}
		if entry.FrameCount <= 0 {
			return fmt.Errorf("%s: frame_count must be positive", entry.ID)
		}
	case "decode-error":
		if entry.BitstreamMD5 == "" || entry.ExpectedError == "" {
			return fmt.Errorf("%s: decode-error entries need bitstream_md5 and expected_error", entry.ID)
		}
	default:
		return fmt.Errorf("%s: failure-ledger row must stay an oracle row, got %q", entry.ID, entry.Expect)
	}
	return nil
}

func validateBenchCorpusCommon(entry benchCorpusEntry) error {
	if entry.ID == "" || entry.Path == "" && entry.URL == "" {
		return fmt.Errorf("manifest entry id and path or url must be set: %+v", entry)
	}
	if entry.Format != "annexb" {
		return fmt.Errorf("%s: format = %q, want annexb", entry.ID, entry.Format)
	}
	switch entry.Extract {
	case "", "h264-annexb":
	default:
		return fmt.Errorf("%s: extract = %q, want h264-annexb or empty", entry.ID, entry.Extract)
	}
	if entry.Extract != "" && entry.SourceMD5 == "" {
		return fmt.Errorf("%s: extracted entries need source_md5", entry.ID)
	}
	return nil
}

func validateBenchKnownFailure(entry benchCorpusEntry) error {
	if entry.KnownFailure == nil {
		return fmt.Errorf("known-red rows must record known_failure")
	}
	if entry.KnownFailure.Class == "" || entry.KnownFailure.DetailContains == "" {
		return fmt.Errorf("known_failure needs class and detail_contains")
	}
	switch entry.KnownFailure.Class {
	case "decode-error", "frame-count-mismatch", "pixel-format-mismatch", "raw-size-mismatch", "source-md5-mismatch", "bitstream-md5-mismatch", "raw-md5-mismatch", "oracle-mismatch", "input-missing":
	default:
		return fmt.Errorf("unknown known_failure class %q", entry.KnownFailure.Class)
	}
	return nil
}

func resolveBenchCorpusPath(baseDir string, entry benchCorpusEntry) (string, error) {
	sourcePath, err := resolveBenchCorpusSourcePath(baseDir, entry)
	if err != nil {
		return "", err
	}
	if entry.Extract == "" {
		return sourcePath, nil
	}
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("%s: read source %s: %w", entry.ID, sourcePath, err)
	}
	if err := validateBenchSourceMD5(entry, sourceData); err != nil {
		return "", err
	}
	return extractBenchCorpusAnnexB(entry, sourcePath)
}

func resolveBenchCorpusSourcePath(baseDir string, entry benchCorpusEntry) (string, error) {
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

func extractBenchCorpusAnnexB(entry benchCorpusEntry, sourcePath string) (string, error) {
	if entry.Extract != "h264-annexb" {
		return "", fmt.Errorf("%s: unsupported extract mode %q", entry.ID, entry.Extract)
	}
	path := sourcePath + ".h264-annexb"
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if os.Getenv("GOH264_CORPUS_FETCH") != "1" && os.Getenv("GOH264_CORPUS_EXTRACT") != "1" {
		return "", fmt.Errorf("%s: missing extracted %s; set GOH264_CORPUS_FETCH=1 or GOH264_CORPUS_EXTRACT=1 to derive it with FFmpeg", entry.ID, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("%s: create extract cache dir: %w", entry.ID, err)
	}
	tmp := path + ".tmp"
	os.Remove(tmp)
	if err := runBenchAnnexBExtract(entry, sourcePath, tmp, true); err != nil {
		if retryErr := runBenchAnnexBExtract(entry, sourcePath, tmp, false); retryErr != nil {
			os.Remove(tmp)
			return "", fmt.Errorf("%s: extract Annex B from %s: with h264_mp4toannexb: %v; without bitstream filter: %v", entry.ID, sourcePath, err, retryErr)
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("%s: install extracted %s: %w", entry.ID, path, err)
	}
	return path, nil
}

func runBenchAnnexBExtract(entry benchCorpusEntry, sourcePath string, outputPath string, withBitstreamFilter bool) error {
	bin := os.Getenv("GOH264_FFMPEG_BIN")
	if bin == "" {
		bin = "ffmpeg"
	}
	args := []string{"-nostdin", "-v", "error", "-y", "-i", sourcePath, "-map", "0:v:0", "-c:v", "copy"}
	if withBitstreamFilter {
		args = append(args, "-bsf:v", "h264_mp4toannexb")
	}
	args = append(args, "-f", "h264", outputPath)
	cmd := exec.Command(bin, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
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

func validateBenchSourceMD5(entry benchCorpusEntry, data []byte) error {
	if entry.SourceMD5 == "" {
		return nil
	}
	sum := md5.Sum(data)
	if got := hex.EncodeToString(sum[:]); got != entry.SourceMD5 {
		return fmt.Errorf("%s: source_md5 = %s, want %s", entry.ID, got, entry.SourceMD5)
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

func annotateBenchFrameDiagnostics(result *benchResult, entry benchCorpusEntry) {
	if result == nil {
		return
	}
	for i := range result.FrameDiagnostics {
		frame := &result.FrameDiagnostics[i]
		if frame.Index >= entry.FrameCount {
			frame.ParityStatus = "extra"
			continue
		}
		frame.ExpectedRawPixelFormat = entry.PixFmt
		if entry.FrameSize > 0 {
			frame.ExpectedBytes = int64(entry.FrameSize)
		}
		if frame.Index < len(entry.FrameMD5) {
			frame.ExpectedRawMD5 = entry.FrameMD5[frame.Index]
		}
		frame.ParityStatus = benchFrameParityStatus(*frame)
	}
	for i := len(result.FrameDiagnostics); i < entry.FrameCount; i++ {
		frame := benchFrameDiagnostic{
			Index:                  i,
			ExpectedRawPixelFormat: entry.PixFmt,
			ExpectedBytes:          int64(entry.FrameSize),
			ParityStatus:           "missing",
		}
		if i < len(entry.FrameMD5) {
			frame.ExpectedRawMD5 = entry.FrameMD5[i]
		}
		result.FrameDiagnostics = append(result.FrameDiagnostics, frame)
	}
}

func benchFrameParityStatus(frame benchFrameDiagnostic) string {
	switch {
	case frame.RawPixelFormat == "":
		return "missing"
	case frame.ExpectedRawPixelFormat != "" && frame.RawPixelFormat != frame.ExpectedRawPixelFormat:
		return "pixel-format-mismatch"
	case frame.ExpectedBytes != 0 && frame.Bytes != frame.ExpectedBytes:
		return "raw-size-mismatch"
	case frame.ExpectedRawMD5 != "" && frame.RawMD5 != frame.ExpectedRawMD5:
		return "raw-md5-mismatch"
	case frame.ExpectedRawMD5 != "":
		return "raw-md5-ok"
	default:
		return "raw-md5-observed"
	}
}

func benchOracleMismatchDetail(result benchResult, entry benchCorpusEntry) string {
	if result.Error != "" {
		return result.Error
	}
	if result.FramesPerIter != entry.FrameCount {
		return fmt.Sprintf("frames_per_iter = %d, want %d", result.FramesPerIter, entry.FrameCount)
	}
	if result.RawPixelFormat != entry.PixFmt {
		return fmt.Sprintf("Go raw_pixel_format = %s, want %s", result.RawPixelFormat, entry.PixFmt)
	}
	expectedBytes := int64(entry.FrameCount * entry.FrameSize)
	if result.BytesPerIter != expectedBytes {
		return fmt.Sprintf("bytes_per_iter = %d, want %d", result.BytesPerIter, expectedBytes)
	}
	if result.RawMD5 != entry.RawVideoMD5 {
		return fmt.Sprintf("raw_md5 = %s, want %s", result.RawMD5, entry.RawVideoMD5)
	}
	return ""
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
	if entry.KnownFailure != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("expected current failure: class=%s contains=%q", entry.KnownFailure.Class, entry.KnownFailure.DetailContains))
	}
	if err != nil {
		result.Error = err.Error()
		result.ErrorClass = benchOracleFailureClass(err.Error())
		if !benchKnownFailureStillCurrent(entry, err.Error()) {
			result.ParityStatus = "known-red-signature-drift"
			result.Notes = append(result.Notes, fmt.Sprintf("current failure signature drifted: class=%s detail=%q", result.ErrorClass, err.Error()))
		}
	}
	return result
}

func staleKnownRedBenchResult(entry benchCorpusEntry, input string, data []byte, ledgerPath string) benchResult {
	result := skippedBenchResult(entry, "listed in the known-red failure ledger and not included in timing aggregates")
	if input != "" {
		result.Input = input
	}
	result.InputBytesPerIter = int64(len(data))
	result.RawMD5 = entry.RawVideoMD5
	result.ParityStatus = "rawvideo-md5-ok-failure-ledger-stale"
	result.BaselineKind = "oracle-known-red-stale"
	if ledgerPath != "" {
		result.Notes = append(result.Notes, "failure ledger: "+ledgerPath)
	}
	result.Notes = append(result.Notes,
		fmt.Sprintf("entry is still listed in %s but passed Go oracle preflight; update the failure ledger before using this as a green benchmark lane", ledgerPath),
	)
	return result
}

func greenNotTimedBenchResult(entry benchCorpusEntry, input string, data []byte, maxEntries int) benchResult {
	result := skippedBenchResult(entry, fmt.Sprintf("oracle-green row not timed because -max-entries=%d was reached", maxEntries))
	if input != "" {
		result.Input = input
	}
	result.InputBytesPerIter = int64(len(data))
	result.RawMD5 = entry.RawVideoMD5
	result.ParityStatus = "rawvideo-md5-ok-not-timed"
	result.BaselineKind = "oracle-green-not-timed"
	return result
}

func benchKnownFailureStillCurrent(entry benchCorpusEntry, detail string) bool {
	if entry.KnownFailure == nil {
		return true
	}
	gotClass := benchOracleFailureClass(detail)
	if gotClass != entry.KnownFailure.Class {
		return false
	}
	return strings.Contains(strings.ToLower(detail), strings.ToLower(entry.KnownFailure.DetailContains))
}

func benchOracleFailureClass(detail string) string {
	detail = strings.ToLower(detail)
	switch {
	case detail == "":
		return ""
	case strings.Contains(detail, "missing ") || strings.Contains(detail, "no such file"):
		return "input-missing"
	case strings.Contains(detail, "decode") || strings.Contains(detail, "unsupported") || strings.Contains(detail, "invalid data"):
		return "decode-error"
	case strings.Contains(detail, "frames_per_iter") || strings.Contains(detail, "frames ="):
		return "frame-count-mismatch"
	case strings.Contains(detail, "raw_pixel_format") || strings.Contains(detail, "pix_fmt"):
		return "pixel-format-mismatch"
	case strings.Contains(detail, "bytes_per_iter") || strings.Contains(detail, "raw size") || strings.Contains(detail, "raw total"):
		return "raw-size-mismatch"
	case strings.Contains(detail, "source_md5"):
		return "source-md5-mismatch"
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

func preflightBenchFFmpegOracle(input string, entry benchCorpusEntry, opts benchOptions, lane ffmpegBenchLane) error {
	pixFmt := opts.ffmpegPixFmt
	if pixFmt == "" {
		pixFmt = entry.PixFmt
	} else if opts.strictPixFmt && pixFmt != entry.PixFmt {
		return fmt.Errorf("-ffmpeg-pix-fmt %q does not match manifest pixel format %q", pixFmt, entry.PixFmt)
	}
	run, err := runFFmpegOnce(opts.ffmpegBin, ffmpegArgs(input, true, opts.ffmpegThreads, pixFmt, lane.cpuFlags), true)
	if err != nil {
		return err
	}
	result := benchResult{
		Name:           lane.name,
		Input:          input,
		RawOutput:      true,
		RawPixelFormat: pixFmt,
		FFmpegPixelFmt: pixFmt,
		BytesPerIter:   run.bytes,
		RawMD5:         run.md5,
		BaselineKind:   "ffmpeg-cli-oracle-preflight",
		BackendKind:    lane.backendKind,
		CPUFlags:       lane.cpuFlags,
		ComparisonLane: lane.comparisonLane,
	}
	return annotateBenchResultWithOracle(&result, entry)
}

func ffmpegBenchLanes(opts benchOptions) []ffmpegBenchLane {
	if !opts.runFFmpeg {
		return nil
	}
	goBackend := h264internal.DecoderBackendKind()
	if opts.fairCPULanes {
		return []ffmpegBenchLane{
			{
				name:           "ffmpeg-pure-c",
				backendKind:    "ffmpeg-pure-c",
				cpuFlags:       "0",
				comparisonLane: ffmpegComparisonLaneForGoBackend("0", goBackend),
			},
			{
				name:           "ffmpeg-native",
				backendKind:    "ffmpeg-native-c+asm",
				cpuFlags:       strings.TrimSpace(opts.ffmpegCPUFlags),
				comparisonLane: ffmpegComparisonLaneForGoBackend(strings.TrimSpace(opts.ffmpegCPUFlags), goBackend),
			},
		}
	}
	flags := strings.TrimSpace(opts.ffmpegCPUFlags)
	return []ffmpegBenchLane{{
		name:           ffmpegLaneName(flags),
		backendKind:    ffmpegBackendKind(flags),
		cpuFlags:       flags,
		comparisonLane: ffmpegComparisonLaneForGoBackend(flags, goBackend),
	}}
}

func ffmpegLaneName(cpuFlags string) string {
	if cpuFlags == "0" {
		return "ffmpeg-pure-c"
	}
	if cpuFlags == "" {
		return "ffmpeg-native"
	}
	return "ffmpeg-cpuflags"
}

func ffmpegBackendKind(cpuFlags string) string {
	if cpuFlags == "0" {
		return "ffmpeg-pure-c"
	}
	if cpuFlags == "" {
		return "ffmpeg-native-c+asm"
	}
	return "ffmpeg-cpuflags-" + cpuFlags
}

func ffmpegComparisonLaneForGoBackend(cpuFlags string, goBackend string) string {
	if goBackend == "" {
		goBackend = "go-unknown"
	}
	if cpuFlags == "0" {
		return "ffmpeg-pure-c-vs-" + goBackend
	}
	if cpuFlags == "" {
		return "ffmpeg-native-c+asm-vs-" + goBackend
	}
	return "ffmpeg-cpuflags-" + cpuFlags + "-vs-" + goBackend
}

func annotateFFmpegPeerQuality(result *benchResult, goResult benchResult) {
	if result == nil || !result.RawOutput || result.RawMD5 == "" || goResult.RawMD5 == "" {
		return
	}
	result.PeerQualityMetric = "rawvideo-md5"
	result.PeerQualityReference = "goh264-rawvideo"
	if result.RawMD5 == goResult.RawMD5 && result.BytesPerIter == goResult.BytesPerIter {
		result.PeerQualityStatus = "rawvideo-md5-match-goh264"
		if result.ParityStatus == "" {
			result.ParityStatus = result.PeerQualityStatus
		}
		return
	}
	result.PeerQualityStatus = "rawvideo-md5-mismatch-goh264"
	if result.ParityStatus == "" {
		result.ParityStatus = result.PeerQualityStatus
	}
	result.ErrorClass = "raw-md5-mismatch"
	result.Notes = append(result.Notes,
		fmt.Sprintf("quality mismatch versus Go output: ffmpeg md5=%s bytes=%d, go md5=%s bytes=%d",
			result.RawMD5, result.BytesPerIter, goResult.RawMD5, goResult.BytesPerIter),
	)
}

func annotateBenchReportQuality(report *benchReport) {
	if report == nil {
		return
	}
	for i := range report.Results {
		annotateBenchResultQuality(&report.Results[i])
	}
}

func annotateBenchResultQuality(result *benchResult) {
	if result == nil || result.QualityStatus != "" {
		return
	}
	if result.ParityStatus == "" {
		return
	}
	result.QualityStatus = result.ParityStatus
	if result.ParityStatus == "decode-error-ok" {
		result.QualityMetric = "decode-error"
		result.QualityReference = "manifest-expected-error"
		return
	}
	if result.ParityStatus == "decode-error" && result.RawMD5 == "" && result.ExpectedRawMD5 == "" {
		result.QualityMetric = "decode-error"
		return
	}
	if result.RawOutput || result.RawMD5 != "" || result.ExpectedRawMD5 != "" {
		result.QualityMetric = "rawvideo-md5"
	}
	switch {
	case strings.Contains(result.ParityStatus, "known-red"):
		result.QualityReference = "failure-ledger"
	case result.ExpectedRawMD5 != "":
		result.QualityReference = "manifest-rawvideo-oracle"
	case strings.Contains(result.ParityStatus, "goh264"):
		result.QualityReference = "goh264-rawvideo"
	case result.RawMD5 != "":
		result.QualityReference = "observed-rawvideo"
	}
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
	if annexBInput {
		if _, _, _, _, _, err := measureGoSample(data, 1, rawOutput, annexBInput); err != nil {
			return benchResult{}, err
		}
	}
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
	if rawOutput && rawMD5 != "" {
		result.ParityStatus = "rawvideo-md5-observed"
	}
	result.InputBytesPerIter = int64(len(data))
	result.BaselineKind = "in-process-go"
	result.BackendKind = h264internal.DecoderBackendKind()
	result.ProcessPerIter = false
	result.InputReadTimed = false
	result.StdoutPipeTimed = false
	if note := h264internal.DecoderBackendNote(); note != "" {
		result.Notes = append(result.Notes, note)
	}
	annotateBenchRates(&result)
	return result, nil
}

func measureGoSample(data []byte, iters int, rawOutput bool, annexBInput bool) (benchSample, int, int64, string, string, error) {
	restoreGOMAXPROCS := 0
	if annexBInput {
		restoreGOMAXPROCS = runtime.GOMAXPROCS(1)
		defer runtime.GOMAXPROCS(restoreGOMAXPROCS)
	}
	var dec *goh264.Decoder
	var borrowed []goh264.Frame
	var borrowedScratch borrowedGoScratch
	if annexBInput {
		dec = goh264.NewDecoder()
		borrowed = make([]goh264.Frame, 0, 16)
		for i := 0; i < 8; i++ {
			run, frames, err := decodeGoAnnexBBorrowedOnce(dec, borrowed, &borrowedScratch, data, rawOutput)
			if err != nil {
				return benchSample{}, 0, 0, "", "", err
			}
			borrowed = frames
			if run.bytes > 0 && int64(cap(borrowedScratch.raw)) < run.bytes && run.bytes <= math.MaxInt {
				borrowedScratch.raw = make([]byte, 0, int(run.bytes))
			}
		}
	}

	runtime.GC()
	if annexBInput {
		run, frames, err := decodeGoAnnexBBorrowedOnce(dec, borrowed, &borrowedScratch, data, rawOutput)
		if err != nil {
			return benchSample{}, 0, 0, "", "", err
		}
		borrowed = frames
		if run.bytes > 0 && int64(cap(borrowedScratch.raw)) < run.bytes && run.bytes <= math.MaxInt {
			borrowedScratch.raw = make([]byte, 0, int(run.bytes))
		}
	}
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()

	var framesPerIter int
	var bytesPerIter int64
	var rawMD5 string
	var rawMD5Sum [md5.Size]byte
	var rawMD5Ready bool
	var pixFmt string
	for i := 0; i < iters; i++ {
		var run decodeGoRun
		var err error
		if annexBInput {
			var frames []goh264.Frame
			frames, err = dec.DecodeAnnexBBorrowedFrames(borrowed, data)
			borrowed = frames
			if err == nil {
				run, err = summarizeGoBorrowedFrames(frames, rawOutput, &borrowedScratch)
			}
		} else {
			run, err = decodeGoOnceForFormat(data, rawOutput, annexBInput)
		}
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
		if annexBInput && rawOutput {
			if !run.hasMD5 {
				return benchSample{}, 0, 0, "", "", fmt.Errorf("missing raw md5 at iter %d", i)
			}
			if !rawMD5Ready {
				rawMD5Sum = run.md5Sum
				rawMD5Ready = true
			} else if run.md5Sum != rawMD5Sum {
				return benchSample{}, 0, 0, "", "", fmt.Errorf("unstable raw md5 at iter %d", i)
			}
		} else {
			rawMD5 = run.md5
		}
	}
	elapsed := time.Since(start)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if annexBInput && rawMD5Ready {
		rawMD5 = hex.EncodeToString(rawMD5Sum[:])
	}
	sample := sampleFromTotals(iters, framesPerIter, bytesPerIter, elapsed, after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs, rawMD5)
	return sample, framesPerIter, bytesPerIter, rawMD5, pixFmt, nil
}

type decodeGoRun struct {
	frames           int
	bytes            int64
	md5              string
	md5Sum           [md5.Size]byte
	hasMD5           bool
	pixFmt           string
	frameDiagnostics []benchFrameDiagnostic
}

type borrowedGoScratch struct {
	raw []byte
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

func decodeGoAnnexBBorrowedOnce(dec *goh264.Decoder, dst []goh264.Frame, scratch *borrowedGoScratch, data []byte, rawOutput bool) (decodeGoRun, []goh264.Frame, error) {
	frames, err := dec.DecodeAnnexBBorrowedFrames(dst, data)
	if err != nil {
		return decodeGoRun{}, frames, err
	}
	run, err := summarizeGoBorrowedFrames(frames, rawOutput, scratch)
	return run, frames, err
}

func summarizeGoFrames(frames []*goh264.Frame, rawOutput bool) (decodeGoRun, error) {
	var pixFmt string
	h := md5.New()
	var scratch []byte
	var total int64
	var frameDiagnostics []benchFrameDiagnostic
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
		if rawOutput {
			scratch = scratch[:0]
			scratch, err = frame.AppendRawYUVBytesLE(scratch)
			if err != nil {
				return decodeGoRun{}, err
			}
			sum := md5.Sum(scratch)
			total += int64(len(scratch))
			if _, err := h.Write(scratch); err != nil {
				return decodeGoRun{}, err
			}
			frameDiagnostics = append(frameDiagnostics, benchFrameDiagnostic{
				Index:          i,
				RawPixelFormat: framePixFmt,
				Bytes:          int64(len(scratch)),
				RawMD5:         hex.EncodeToString(sum[:]),
			})
		}
	}
	if !rawOutput {
		return decodeGoRun{frames: len(frames), pixFmt: pixFmt}, nil
	}
	return decodeGoRun{frames: len(frames), bytes: total, md5: hashString(h), pixFmt: pixFmt, frameDiagnostics: frameDiagnostics}, nil
}

func summarizeGoBorrowedFrames(frames []goh264.Frame, rawOutput bool, scratch *borrowedGoScratch) (decodeGoRun, error) {
	var pixFmt string
	var total int64
	var localScratch borrowedGoScratch
	if scratch == nil {
		scratch = &localScratch
	}
	if rawOutput {
		scratch.raw = scratch.raw[:0]
	}
	for i := range frames {
		frame := &frames[i]
		framePixFmt, err := frame.RawPixelFormat()
		if err != nil {
			return decodeGoRun{}, err
		}
		if i == 0 {
			pixFmt = framePixFmt
		} else if framePixFmt != pixFmt {
			return decodeGoRun{}, fmt.Errorf("mixed raw pixel formats: frame[0]=%s frame[%d]=%s", pixFmt, i, framePixFmt)
		}
		if rawOutput {
			scratch.raw, err = frame.AppendRawYUVBytesLE(scratch.raw)
			if err != nil {
				return decodeGoRun{}, err
			}
			total = int64(len(scratch.raw))
		}
	}
	if !rawOutput {
		return decodeGoRun{frames: len(frames), pixFmt: pixFmt}, nil
	}
	md5Sum := md5.Sum(scratch.raw)
	return decodeGoRun{frames: len(frames), bytes: total, md5Sum: md5Sum, hasMD5: true, pixFmt: pixFmt}, nil
}

func benchFFmpeg(input string, inputBytes int64, iters int, repeats int, warmup int, rawOutput bool, bin string, threads string, pixFmt string, goPixFmt string, processPerIter bool, lane ffmpegBenchLane) (benchResult, error) {
	effectivePixFmt := pixFmt
	autoPixFmt := false
	if rawOutput && effectivePixFmt == "" && goPixFmt != "" {
		effectivePixFmt = goPixFmt
		autoPixFmt = true
	}
	args := ffmpegArgs(input, rawOutput, threads, effectivePixFmt, lane.cpuFlags)
	for i := 0; i < warmup; i++ {
		if _, err := runFFmpegOnce(bin, args, rawOutput); err != nil {
			return benchResult{}, err
		}
	}

	captureSingleRaw := !processPerIter && rawOutput
	singleRun, err := runFFmpegOnceCapture(bin, args, rawOutput, captureSingleRaw)
	if err != nil {
		return benchResult{}, err
	}
	bytesPerIter := singleRun.bytes
	rawMD5 := singleRun.md5
	amortizedRawMD5 := ""
	if captureSingleRaw {
		amortizedRawMD5 = repeatedRawMD5(singleRun.raw, iters)
	}
	amortizedInput := input
	if !processPerIter {
		var cleanup func()
		amortizedInput, cleanup, err = prepareFFmpegAmortizedInput(input, iters)
		if err != nil {
			return benchResult{}, err
		}
		defer cleanup()
	}
	var samples []benchSample
	for repeat := 0; repeat < repeats; repeat++ {
		var sample benchSample
		var bytes int64
		var sum string
		var err error
		if processPerIter {
			sample, bytes, sum, err = measureFFmpegSampleProcessPerIter(bin, args, iters, rawOutput)
		} else {
			amortizedArgs := ffmpegArgs(amortizedInput, rawOutput, threads, effectivePixFmt, lane.cpuFlags)
			sample, bytes, sum, err = measureFFmpegSampleAmortized(bin, amortizedArgs, iters, rawOutput, bytesPerIter, amortizedRawMD5)
		}
		if err != nil {
			return benchResult{}, err
		}
		if bytes != bytesPerIter {
			return benchResult{}, fmt.Errorf("unstable FFmpeg byte count at repeat %d: %d, want %d", repeat, bytes, bytesPerIter)
		}
		if processPerIter && sum != rawMD5 {
			return benchResult{}, fmt.Errorf("unstable FFmpeg raw md5 at repeat %d: %s, want %s", repeat, sum, rawMD5)
		}
		samples = append(samples, sample)
	}

	commandArgs := args
	if !processPerIter {
		commandArgs = ffmpegArgs(amortizedInput, rawOutput, threads, effectivePixFmt, lane.cpuFlags)
	}
	result := resultFromSamples(lane.name, input, iters, repeats, warmup, rawOutput, 0, bytesPerIter, samples, rawMD5, bin+" "+joinArgs(commandArgs))
	result.RawPixelFormat = effectivePixFmt
	result.FFmpegPixelFmt = effectivePixFmt
	result.InputBytesPerIter = inputBytes
	if processPerIter {
		result.BaselineKind = "ffmpeg-cli-process-per-iter"
	} else {
		result.BaselineKind = "ffmpeg-cli-amortized"
	}
	result.BackendKind = lane.backendKind
	result.CPUFlags = lane.cpuFlags
	result.ComparisonLane = lane.comparisonLane
	result.ProcessPerIter = processPerIter
	result.InputReadTimed = true
	result.StdoutPipeTimed = rawOutput
	if processPerIter {
		result.Notes = append(result.Notes,
			"FFmpeg is executed once per timed iteration, so this baseline includes process startup, CLI demux/parser setup, input file reads, and stdout pipe cost per iteration.",
		)
	} else {
		result.Notes = append(result.Notes,
			"FFmpeg is executed once per repeat sample over a prebuilt repeated input file, so process startup and CLI setup are amortized across the sample. Input file reads and stdout pipe cost remain timed.",
		)
		if rawOutput {
			result.Notes = append(result.Notes,
				"Each amortized FFmpeg sample raw-MD5 is checked against the single-iteration raw output repeated for every timed iteration.",
			)
		}
	}
	if autoPixFmt {
		result.Notes = append(result.Notes, "FFmpeg -pix_fmt was auto-selected from the Go raw pixel format for raw-MD5 parity.")
	}
	annotateBenchRates(&result)
	return result, nil
}

func measureFFmpegSampleProcessPerIter(bin string, args []string, iters int, rawOutput bool) (benchSample, int64, string, error) {
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

func measureFFmpegSampleAmortized(bin string, args []string, iters int, rawOutput bool, bytesPerIter int64, expectedRawMD5 string) (benchSample, int64, string, error) {
	start := time.Now()
	run, err := runFFmpegOnce(bin, args, rawOutput)
	if err != nil {
		return benchSample{}, 0, "", err
	}
	elapsed := time.Since(start)
	if rawOutput {
		wantBytes := bytesPerIter * int64(iters)
		if run.bytes != wantBytes {
			return benchSample{}, 0, "", fmt.Errorf("FFmpeg amortized byte count = %d, want %d (%d bytes/iter * %d iters)", run.bytes, wantBytes, bytesPerIter, iters)
		}
		if expectedRawMD5 != "" && run.md5 != expectedRawMD5 {
			return benchSample{}, 0, "", fmt.Errorf("FFmpeg amortized raw md5 = %s, want repeated single-iteration raw md5 %s", run.md5, expectedRawMD5)
		}
	}
	return sampleFromTotals(iters, 0, bytesPerIter, elapsed, 0, 0, run.md5), bytesPerIter, run.md5, nil
}

func repeatedRawMD5(raw []byte, iters int) string {
	h := md5.New()
	for i := 0; i < iters; i++ {
		_, _ = h.Write(raw)
	}
	return hashString(h)
}

func prepareFFmpegAmortizedInput(input string, iters int) (string, func(), error) {
	if iters <= 1 {
		return input, func() {}, nil
	}
	src, err := os.Open(input)
	if err != nil {
		return "", nil, err
	}
	defer src.Close()
	ext := filepath.Ext(input)
	if ext == "" {
		ext = ".h264"
	}
	tmp, err := os.CreateTemp("", "goh264bench-ffmpeg-repeat-*"+ext)
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			cleanup()
		}
	}()
	for i := 0; i < iters; i++ {
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			return "", nil, err
		}
		if _, err := io.Copy(tmp, src); err != nil {
			return "", nil, err
		}
	}
	if err := tmp.Close(); err != nil {
		return "", nil, err
	}
	ok = true
	return tmp.Name(), cleanup, nil
}

type ffmpegRun struct {
	bytes int64
	md5   string
	raw   []byte
}

func runFFmpegOnce(bin string, args []string, rawOutput bool) (ffmpegRun, error) {
	return runFFmpegOnceCapture(bin, args, rawOutput, false)
}

func runFFmpegOnceCapture(bin string, args []string, rawOutput bool, captureRaw bool) (ffmpegRun, error) {
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
	if captureRaw {
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return ffmpegRun{}, fmt.Errorf("%w: %s", err, stderr.String())
		}
		sum := md5.Sum(out.Bytes())
		return ffmpegRun{bytes: int64(out.Len()), md5: hex.EncodeToString(sum[:]), raw: out.Bytes()}, nil
	}
	h := md5.New()
	counter := &countingWriter{w: h}
	cmd.Stdout = counter
	if err := cmd.Run(); err != nil {
		return ffmpegRun{}, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return ffmpegRun{bytes: counter.n, md5: hashString(h)}, nil
}

func ffmpegArgs(input string, rawOutput bool, threads string, pixFmt string, cpuFlags string) []string {
	args := []string{"-v", "error", "-nostdin"}
	if strings.TrimSpace(cpuFlags) != "" {
		args = append(args, "-cpuflags", strings.TrimSpace(cpuFlags))
	}
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
	var allocBytesPerIter float64
	var allocsPerIter float64
	if iters > 0 {
		allocBytesPerIter = float64(allocBytes) / float64(iters)
		allocsPerIter = float64(allocs) / float64(iters)
	}
	return benchSample{
		ElapsedMS:         float64(elapsed.Microseconds()) / 1000,
		TotalFrames:       totalFrames,
		TotalBytes:        totalBytes,
		FPS:               fps,
		MiBPerSec:         mibPerSec,
		AllocBytes:        allocBytes,
		Allocs:            allocs,
		AllocBytesPerIter: allocBytesPerIter,
		AllocsPerIter:     allocsPerIter,
		RawMD5:            rawMD5,
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
	if result == nil {
		return
	}
	if result.Iterations > 0 && result.Repeats > 0 {
		totalIters := result.Iterations * result.Repeats
		result.AllocBytesPerIter = float64(result.AllocBytes) / float64(totalIters)
		result.AllocsPerIter = float64(result.Allocs) / float64(totalIters)
	}
	if result.TotalFrames > 0 {
		result.AllocBytesPerFrame = float64(result.AllocBytes) / float64(result.TotalFrames)
		result.AllocsPerFrame = float64(result.Allocs) / float64(result.TotalFrames)
	}
	if result.ElapsedMS <= 0 {
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

func benchmarkMetadata(input string, data []byte, opts benchOptions) benchMetadata {
	sum := md5.Sum(data)
	revision, dirty := gitMetadata()
	modulePath, moduleVersion := moduleMetadata()
	meta := benchMetadata{
		Input:                  input,
		InputBytes:             int64(len(data)),
		InputMD5:               hex.EncodeToString(sum[:]),
		GoVersion:              runtime.Version(),
		GOOS:                   runtime.GOOS,
		GOARCH:                 runtime.GOARCH,
		NumCPU:                 runtime.NumCPU(),
		GOMAXPROCS:             runtime.GOMAXPROCS(0),
		ModulePath:             modulePath,
		ModuleVersion:          moduleVersion,
		VCSRevision:            revision,
		VCSDirty:               dirty,
		ComparisonKind:         "goh264-in-process",
		MaxGoAllocBytesPerIter: opts.maxGoAllocBytesPerIter,
		MaxGoAllocsPerIter:     opts.maxGoAllocsPerIter,
	}
	if opts.runFFmpeg {
		meta.ComparisonKind = "goh264-in-process-vs-ffmpeg-cli-amortized"
		if opts.ffmpegProcessPerIter {
			meta.ComparisonKind = "goh264-in-process-vs-ffmpeg-cli-process-per-iter"
		}
		if opts.fairCPULanes {
			meta.ComparisonKind += "-fair-cpu-lanes"
		}
		meta.FFmpegVersion = ffmpegVersion(opts.ffmpegBin)
		meta.FFmpegCPUFlags = ffmpegMetadataCPUFlags(opts)
		meta.FairnessPolicy = "Single-input mode reports Go and FFmpeg timing samples with explicit backend_kind/cpu_flags fields. Fair CPU lanes label each FFmpeg CPU mode against the actual measured Go backend_kind; run default and purego builds separately when both Go backend comparisons are needed. FFmpeg peer_quality_status is compared against the Go rawvideo byte count and raw-MD5 when -raw=true; manifest mode is required for an external rawvideo oracle quality_status. FFmpeg timing defaults to one CLI process per repeat sample over a prebuilt repeated input file, amortizing process startup and CLI setup across timed iterations; raw-output amortized samples must also match the single-iteration raw output repeated for every timed iteration; -ffmpeg-process-per-iter restores the historical process-per-iteration baseline."
	}
	meta.ForbidGoAllocations = opts.forbidGoAllocations
	return meta
}

func ffmpegMetadataCPUFlags(opts benchOptions) string {
	if !opts.runFFmpeg {
		return ""
	}
	if opts.fairCPULanes {
		if strings.TrimSpace(opts.ffmpegCPUFlags) == "" {
			return "pure-c:0,native-c+asm:default"
		}
		return "pure-c:0,native-c+asm:" + strings.TrimSpace(opts.ffmpegCPUFlags)
	}
	if strings.TrimSpace(opts.ffmpegCPUFlags) == "" {
		return "default"
	}
	return strings.TrimSpace(opts.ffmpegCPUFlags)
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
	var sum [md5.Size]byte
	return hex.EncodeToString(h.Sum(sum[:0]))
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
