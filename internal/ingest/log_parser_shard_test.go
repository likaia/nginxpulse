package ingest

import (
	"strings"
	"testing"
)

func TestReadLineAlignedShardsPreservesWholeLines(t *testing.T) {
	input := "aaa\nbbbb\ncc\n"
	type shard struct {
		base int64
		data string
	}
	var shards []shard

	total, err := readLineAlignedShards(strings.NewReader(input), 5, func(base int64, data []byte) bool {
		shards = append(shards, shard{base: base, data: string(data)})
		return true
	})
	if err != nil {
		t.Fatalf("readLineAlignedShards error: %v", err)
	}
	if total != int64(len(input)) {
		t.Fatalf("total bytes = %d, want %d", total, len(input))
	}
	if len(shards) != 2 {
		t.Fatalf("shard count = %d, want 2", len(shards))
	}
	if shards[0].base != 0 || shards[0].data != "aaa\nbbbb\n" {
		t.Fatalf("first shard = %#v", shards[0])
	}
	if shards[1].base != int64(len(shards[0].data)) || shards[1].data != "cc\n" {
		t.Fatalf("second shard = %#v", shards[1])
	}
	if got := shards[0].data + shards[1].data; got != input {
		t.Fatalf("reassembled input = %q, want %q", got, input)
	}
}

func TestReadLineAlignedShardsDoesNotSplitLongLine(t *testing.T) {
	longLine := strings.Repeat("x", 64) + "\n"
	input := longLine + "tail-without-newline"
	var shards []string
	var bases []int64

	total, err := readLineAlignedShards(strings.NewReader(input), 8, func(base int64, data []byte) bool {
		bases = append(bases, base)
		shards = append(shards, string(data))
		return true
	})
	if err != nil {
		t.Fatalf("readLineAlignedShards error: %v", err)
	}
	if total != int64(len(input)) {
		t.Fatalf("total bytes = %d, want %d", total, len(input))
	}
	if len(shards) != 2 {
		t.Fatalf("shard count = %d, want 2", len(shards))
	}
	if shards[0] != longLine {
		t.Fatalf("long line was split: first shard length = %d, want %d", len(shards[0]), len(longLine))
	}
	if shards[1] != "tail-without-newline" {
		t.Fatalf("final shard = %q", shards[1])
	}
	if bases[1] != int64(len(longLine)) {
		t.Fatalf("second base = %d, want %d", bases[1], len(longLine))
	}
}
