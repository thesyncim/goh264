// SPDX-License-Identifier: LGPL-2.1-or-later

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"os/exec"
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
	Name            string        `json:"name"`
	Input           string        `json:"input"`
	Iterations      int           `json:"iterations"`
	Repeats         int           `json:"repeats"`
	Warmup          int           `json:"warmup"`
	RawOutput       bool          `json:"raw_output"`
	RawPixelFormat  string        `json:"raw_pixel_format,omitempty"`
	FFmpegPixelFmt  string        `json:"ffmpeg_pixel_format,omitempty"`
	FramesPerIter   int           `json:"frames_per_iter,omitempty"`
	BytesPerIter    int64         `json:"bytes_per_iter,omitempty"`
	TotalFrames     int           `json:"total_frames,omitempty"`
	TotalBytes      int64         `json:"total_bytes,omitempty"`
	ElapsedMS       float64       `json:"elapsed_ms"`
	MeanElapsedMS   float64       `json:"mean_elapsed_ms,omitempty"`
	MedianElapsedMS float64       `json:"median_elapsed_ms,omitempty"`
	MinElapsedMS    float64       `json:"min_elapsed_ms,omitempty"`
	MaxElapsedMS    float64       `json:"max_elapsed_ms,omitempty"`
	StddevElapsedMS float64       `json:"stddev_elapsed_ms,omitempty"`
	CVElapsed       float64       `json:"cv_elapsed,omitempty"`
	FPS             float64       `json:"fps,omitempty"`
	MiBPerSec       float64       `json:"mib_per_sec,omitempty"`
	AllocBytes      uint64        `json:"alloc_bytes,omitempty"`
	Allocs          uint64        `json:"allocs,omitempty"`
	RawMD5          string        `json:"raw_md5,omitempty"`
	Command         string        `json:"command,omitempty"`
	ProcessPerIter  bool          `json:"process_per_iter"`
	InputReadTimed  bool          `json:"input_read_timed"`
	StdoutPipeTimed bool          `json:"stdout_pipe_timed"`
	BaselineKind    string        `json:"baseline_kind"`
	Notes           []string      `json:"notes,omitempty"`
	Samples         []benchSample `json:"samples,omitempty"`
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

func main() {
	input := flag.String("input", "", "H.264 input file")
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

	if *input == "" || *iters <= 0 || *repeats <= 0 || *warmup < 0 {
		fmt.Fprintln(os.Stderr, "usage: goh264bench -input file.h264 [-iters 5] [-repeats 1] [-warmup 1] [-ffmpeg] [-json]")
		os.Exit(2)
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		die("read input", err)
	}

	var results []benchResult
	goResult, err := benchGo(*input, data, *iters, *repeats, *warmup, *rawOutput)
	if err != nil {
		die("goh264", err)
	}
	results = append(results, goResult)

	if *runFFmpeg && *rawOutput && *strictPixFmt && *ffmpegPixFmt != "" && goResult.RawPixelFormat != "" && *ffmpegPixFmt != goResult.RawPixelFormat {
		fmt.Fprintf(os.Stderr, "-ffmpeg-pix-fmt %q does not match Go raw pixel format %q\n", *ffmpegPixFmt, goResult.RawPixelFormat)
		os.Exit(2)
	}
	if *runFFmpeg {
		ffmpegResult, err := benchFFmpeg(*input, *iters, *repeats, *warmup, *rawOutput, *ffmpegBin, *ffmpegThreads, *ffmpegPixFmt, goResult.RawPixelFormat)
		if err != nil {
			die("ffmpeg", err)
		}
		results = append(results, ffmpegResult)
	}

	report := benchReport{
		Metadata: benchmarkMetadata(*input, data, *runFFmpeg, *ffmpegBin),
		Results:  results,
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
		fmt.Printf("%s: %.2f ms over %d repeat(s) x %d iter", r.Name, r.ElapsedMS, r.Repeats, r.Iterations)
		if r.Repeats > 1 {
			fmt.Printf(", median %.2f ms, cv %.4f", r.MedianElapsedMS, r.CVElapsed)
		}
		if r.FramesPerIter > 0 {
			fmt.Printf(", %d frames/iter, %.2f fps", r.FramesPerIter, r.FPS)
		}
		if r.BytesPerIter > 0 {
			fmt.Printf(", %d bytes/iter, %.2f MiB/s", r.BytesPerIter, r.MiBPerSec)
		}
		if r.Allocs > 0 || r.AllocBytes > 0 {
			fmt.Printf(", %.2f allocs/iter, %.2f MiB alloc/iter",
				float64(r.Allocs)/float64(r.Iterations),
				float64(r.AllocBytes)/float64(r.Iterations)/(1024*1024))
		}
		if r.RawMD5 != "" {
			fmt.Printf(", raw md5 %s", r.RawMD5)
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

func benchGo(input string, data []byte, iters int, repeats int, warmup int, rawOutput bool) (benchResult, error) {
	for i := 0; i < warmup; i++ {
		if _, err := decodeGoOnce(data, rawOutput); err != nil {
			return benchResult{}, err
		}
	}

	var framesPerIter int
	var bytesPerIter int64
	var rawMD5 string
	var pixFmt string
	var samples []benchSample
	for repeat := 0; repeat < repeats; repeat++ {
		sample, frames, bytes, sum, samplePixFmt, err := measureGoSample(data, iters, rawOutput)
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
	result.BaselineKind = "in-process-go"
	result.ProcessPerIter = false
	result.InputReadTimed = false
	result.StdoutPipeTimed = false
	return result, nil
}

func measureGoSample(data []byte, iters int, rawOutput bool) (benchSample, int, int64, string, string, error) {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()

	var framesPerIter int
	var bytesPerIter int64
	var rawMD5 string
	var pixFmt string
	for i := 0; i < iters; i++ {
		run, err := decodeGoOnce(data, rawOutput)
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

func benchFFmpeg(input string, iters int, repeats int, warmup int, rawOutput bool, bin string, threads string, pixFmt string, goPixFmt string) (benchResult, error) {
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
	return benchResult{
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
