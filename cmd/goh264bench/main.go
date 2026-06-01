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
	"os"
	"os/exec"
	"runtime"
	"time"

	goh264 "github.com/thesyncim/goh264"
)

type benchResult struct {
	Name          string  `json:"name"`
	Input         string  `json:"input"`
	Iterations    int     `json:"iterations"`
	Warmup        int     `json:"warmup"`
	RawOutput     bool    `json:"raw_output"`
	FramesPerIter int     `json:"frames_per_iter,omitempty"`
	BytesPerIter  int64   `json:"bytes_per_iter,omitempty"`
	TotalFrames   int     `json:"total_frames,omitempty"`
	TotalBytes    int64   `json:"total_bytes,omitempty"`
	ElapsedMS     float64 `json:"elapsed_ms"`
	FPS           float64 `json:"fps,omitempty"`
	MiBPerSec     float64 `json:"mib_per_sec,omitempty"`
	AllocBytes    uint64  `json:"alloc_bytes,omitempty"`
	Allocs        uint64  `json:"allocs,omitempty"`
	RawMD5        string  `json:"raw_md5,omitempty"`
	Command       string  `json:"command,omitempty"`
}

func main() {
	input := flag.String("input", "", "H.264 input file")
	iters := flag.Int("iters", 5, "measured iterations")
	warmup := flag.Int("warmup", 1, "warmup iterations")
	rawOutput := flag.Bool("raw", true, "materialize raw decoded bytes during Go and FFmpeg runs")
	runFFmpeg := flag.Bool("ffmpeg", false, "also run an FFmpeg baseline over the same file")
	ffmpegBin := flag.String("ffmpeg-bin", "ffmpeg", "FFmpeg binary")
	ffmpegThreads := flag.String("ffmpeg-threads", "1", "FFmpeg -threads value")
	ffmpegPixFmt := flag.String("ffmpeg-pix-fmt", "", "optional FFmpeg output pixel format for -raw mode")
	jsonOut := flag.Bool("json", false, "print JSON")
	flag.Parse()

	if *input == "" || *iters <= 0 || *warmup < 0 {
		fmt.Fprintln(os.Stderr, "usage: goh264bench -input file.h264 [-iters 5] [-warmup 1] [-ffmpeg] [-json]")
		os.Exit(2)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		die("read input", err)
	}

	var results []benchResult
	goResult, err := benchGo(*input, data, *iters, *warmup, *rawOutput)
	if err != nil {
		die("goh264", err)
	}
	results = append(results, goResult)

	if *runFFmpeg {
		ffmpegResult, err := benchFFmpeg(*input, *iters, *warmup, *rawOutput, *ffmpegBin, *ffmpegThreads, *ffmpegPixFmt)
		if err != nil {
			die("ffmpeg", err)
		}
		results = append(results, ffmpegResult)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			die("json", err)
		}
		return
	}
	for _, r := range results {
		fmt.Printf("%s: %.2f ms over %d iter", r.Name, r.ElapsedMS, r.Iterations)
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
		fmt.Println()
	}
}

func benchGo(input string, data []byte, iters int, warmup int, rawOutput bool) (benchResult, error) {
	for i := 0; i < warmup; i++ {
		if _, _, _, err := decodeGoOnce(data, rawOutput); err != nil {
			return benchResult{}, err
		}
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()

	var framesPerIter int
	var bytesPerIter int64
	var rawMD5 string
	for i := 0; i < iters; i++ {
		frames, bytes, sum, err := decodeGoOnce(data, rawOutput)
		if err != nil {
			return benchResult{}, err
		}
		if i == 0 {
			framesPerIter = frames
			bytesPerIter = bytes
		}
		if frames != framesPerIter || bytes != bytesPerIter {
			return benchResult{}, fmt.Errorf("unstable decode result at iter %d: frames/bytes = %d/%d, want %d/%d", i, frames, bytes, framesPerIter, bytesPerIter)
		}
		rawMD5 = sum
	}

	elapsed := time.Since(start)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	return resultFromTotals("goh264", input, iters, warmup, rawOutput, framesPerIter, bytesPerIter, elapsed, after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs, rawMD5, ""), nil
}

func decodeGoOnce(data []byte, rawOutput bool) (int, int64, string, error) {
	dec := goh264.NewDecoder()
	frames, err := dec.DecodeFrames(data)
	if err != nil {
		return 0, 0, "", err
	}
	delayed, err := dec.DecodeFrames(nil)
	if err != nil {
		return 0, 0, "", err
	}
	frames = append(frames, delayed...)

	if !rawOutput {
		return len(frames), 0, "", nil
	}
	h := md5.New()
	var scratch []byte
	var total int64
	for _, frame := range frames {
		scratch = scratch[:0]
		scratch, err = frame.AppendRawYUVBytesLE(scratch)
		if err != nil {
			return 0, 0, "", err
		}
		total += int64(len(scratch))
		if _, err := h.Write(scratch); err != nil {
			return 0, 0, "", err
		}
	}
	return len(frames), total, hashString(h), nil
}

func benchFFmpeg(input string, iters int, warmup int, rawOutput bool, bin string, threads string, pixFmt string) (benchResult, error) {
	args := ffmpegArgs(input, rawOutput, threads, pixFmt)
	for i := 0; i < warmup; i++ {
		if _, err := runFFmpegOnce(bin, args, rawOutput); err != nil {
			return benchResult{}, err
		}
	}

	start := time.Now()
	var bytesPerIter int64
	var rawMD5 string
	for i := 0; i < iters; i++ {
		run, err := runFFmpegOnce(bin, args, rawOutput)
		if err != nil {
			return benchResult{}, err
		}
		if i == 0 {
			bytesPerIter = run.bytes
		}
		if run.bytes != bytesPerIter {
			return benchResult{}, fmt.Errorf("unstable FFmpeg byte count at iter %d: %d, want %d", i, run.bytes, bytesPerIter)
		}
		rawMD5 = run.md5
	}
	elapsed := time.Since(start)

	return resultFromTotals("ffmpeg", input, iters, warmup, rawOutput, 0, bytesPerIter, elapsed, 0, 0, rawMD5, bin+" "+joinArgs(args)), nil
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

func resultFromTotals(name string, input string, iters int, warmup int, rawOutput bool, framesPerIter int, bytesPerIter int64, elapsed time.Duration, allocBytes uint64, allocs uint64, rawMD5 string, command string) benchResult {
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
	return benchResult{
		Name:          name,
		Input:         input,
		Iterations:    iters,
		Warmup:        warmup,
		RawOutput:     rawOutput,
		FramesPerIter: framesPerIter,
		BytesPerIter:  bytesPerIter,
		TotalFrames:   totalFrames,
		TotalBytes:    totalBytes,
		ElapsedMS:     float64(elapsed.Microseconds()) / 1000,
		FPS:           fps,
		MiBPerSec:     mibPerSec,
		AllocBytes:    allocBytes,
		Allocs:        allocs,
		RawMD5:        rawMD5,
		Command:       command,
	}
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
