// SPDX-License-Identifier: LGPL-2.1-or-later

package h264

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const cabacOracleC = `
#include <stdint.h>
#include <stdio.h>

#include "libavcodec/cabac.h"
#include "libavcodec/cabac_functions.h"
#include "libavcodec/cabac.c"

static uint8_t *select_start(uint8_t *backing, int want_aligned)
{
    for (int off = 0; off < 8; off++) {
        uint8_t *p = backing + off;
        if ((((uintptr_t)(p + 2) & 1) == 0) == want_aligned)
            return p;
    }
    return NULL;
}

static void run_case(int want_aligned)
{
    const uint8_t fixture[6] = { 0x2a, 0x40, 0x80, 0x11, 0x22, 0x33 };
    uint8_t backing[32] = { 0 };
    uint8_t *buf = select_start(backing, want_aligned);
    CABACContext c;
    uint8_t state = 92;
    int bit1, bit2, bypass, sign, term;

    for (int i = 0; i < 6; i++)
        buf[i] = fixture[i];

    if (ff_init_cabac_decoder(&c, buf, 6) < 0) {
        printf("%d init-error\n", want_aligned);
        return;
    }

    printf("%d init %d %d %ld\n", want_aligned, c.low, c.range,
           (long)(c.bytestream - buf));

    bit1 = get_cabac(&c, &state);
    printf("%d cabac1 %d %u %d %d %ld\n", want_aligned, bit1, state,
           c.low, c.range, (long)(c.bytestream - buf));

    bit2 = get_cabac(&c, &state);
    printf("%d cabac2 %d %u %d %d %ld\n", want_aligned, bit2, state,
           c.low, c.range, (long)(c.bytestream - buf));

    bypass = get_cabac_bypass(&c);
    printf("%d bypass %d %d %d %ld\n", want_aligned, bypass, c.low,
           c.range, (long)(c.bytestream - buf));

    sign = get_cabac_bypass_sign(&c, -3);
    printf("%d sign %d %d %d %ld\n", want_aligned, sign, c.low, c.range,
           (long)(c.bytestream - buf));

    term = get_cabac_terminate(&c);
    printf("%d term %d %d %d %ld\n", want_aligned, term, c.low, c.range,
           (long)(c.bytestream - buf));
}

int main(void)
{
    run_case(1);
    run_case(0);
    return 0;
}
`

func TestCABACPrimitiveSequenceUpstreamOracle(t *testing.T) {
	if os.Getenv("GOH264_ORACLE") != "1" {
		t.Skip("set GOH264_ORACLE=1 to run pinned FFmpeg CABAC oracle")
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("cc not available")
	}

	root := h264RepoRoot(t)
	upstream := filepath.Join(root, ".upstream", "ffmpeg-n8.0.1")
	if _, err := os.Stat(filepath.Join(upstream, "libavcodec", "cabac.c")); err != nil {
		t.Skipf("pinned upstream cache not available: %v", err)
	}

	dir := t.TempDir()
	writeOracleFile(t, filepath.Join(dir, "oracle.c"), cabacOracleC)
	writeOracleFile(t, filepath.Join(dir, "config.h"), strings.Join([]string{
		"#define ARCH_AARCH64 0",
		"#define ARCH_ARM 0",
		"#define ARCH_X86 0",
		"#define ARCH_MIPS 0",
		"#define ARCH_LOONGARCH64 0",
		"#define CONFIG_SAFE_BITSTREAM_READER 1",
		"#define HAVE_FAST_CLZ 0",
		"#define AV_HAVE_BIGENDIAN 0",
		"#define HAVE_BIGENDIAN 0",
		"#define HAVE_FAST_UNALIGNED 1",
		"",
	}, "\n"))
	writeOracleFile(t, filepath.Join(dir, "config_components.h"), "")
	if err := os.Mkdir(filepath.Join(dir, "libavutil"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeOracleFile(t, filepath.Join(dir, "libavutil", "avconfig.h"), strings.Join([]string{
		"#define AV_HAVE_BIGENDIAN 0",
		"#define AV_HAVE_FAST_UNALIGNED 1",
		"",
	}, "\n"))

	bin := filepath.Join(dir, "oracle")
	cmd := exec.Command(cc, "-std=c99", "-I"+dir, "-I"+upstream, filepath.Join(dir, "oracle.c"), "-o", bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile CABAC oracle: %v\n%s", err, out)
	}

	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatalf("run CABAC oracle: %v\n%s", err, out)
	}

	got := strings.TrimSpace(string(out))
	want := strings.TrimSpace(cabacOracleWant(t))
	if got != want {
		t.Fatalf("CABAC oracle mismatch\nC oracle:\n%s\nGo:\n%s", got, want)
	}
}

func cabacOracleWant(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	run := func(aligned bool) {
		wantAligned := 0
		if aligned {
			wantAligned = 1
		}
		c, err := initCABACDecoderAligned([]byte{0x2a, 0x40, 0x80, 0x11, 0x22, 0x33}, aligned)
		if err != nil {
			t.Fatalf("initCABACDecoderAligned(%v): %v", aligned, err)
		}
		fmt.Fprintf(&b, "%d init %d %d %d\n", wantAligned, c.low, c.rng, c.bytestream)
		state := uint8(92)

		bit := c.getCABAC(&state)
		fmt.Fprintf(&b, "%d cabac1 %d %d %d %d %d\n", wantAligned, bit, state, c.low, c.rng, c.bytestream)

		bit = c.getCABAC(&state)
		fmt.Fprintf(&b, "%d cabac2 %d %d %d %d %d\n", wantAligned, bit, state, c.low, c.rng, c.bytestream)

		bit = c.getCABACBypass()
		fmt.Fprintf(&b, "%d bypass %d %d %d %d\n", wantAligned, bit, c.low, c.rng, c.bytestream)

		sign := c.getCABACBypassSign(-3)
		fmt.Fprintf(&b, "%d sign %d %d %d %d\n", wantAligned, sign, c.low, c.rng, c.bytestream)

		term := c.getCABACTerminate()
		fmt.Fprintf(&b, "%d term %d %d %d %d\n", wantAligned, term, c.low, c.rng, c.bytestream)
	}
	run(true)
	run(false)
	return b.String()
}

func h264RepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func writeOracleFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
