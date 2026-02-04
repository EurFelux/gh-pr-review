package preview

import (
	"testing"
)

func TestParseDiffHunk(t *testing.T) {
	tests := []struct {
		name       string
		diffHunk   string
		startLine  int
		targetLine int
		side       string
		want       []string
	}{
		{
			name: "RIGHT side - added line",
			diffHunk: `@@ -1,5 +1,6 @@
 line1
 line2
+added line
 line3
 line4
 line5`,
			startLine:  3,
			targetLine: 3,
			side:       "RIGHT",
			want:       []string{"3: +added line"},
		},
		{
			name: "RIGHT side - multi-line addition",
			diffHunk: `@@ -10,5 +10,8 @@
 line10
 line11
+new line 12
+new line 13
+new line 14
 line15
 line16`,
			startLine:  12,
			targetLine: 14,
			side:       "RIGHT",
			want: []string{
				"12: +new line 12",
				"13: +new line 13",
				"14: +new line 14",
			},
		},
		{
			name: "LEFT side - deleted line",
			diffHunk: `@@ -1,5 +1,4 @@
 line1
-deleted line
 line3
 line4
 line5`,
			startLine:  2,
			targetLine: 2,
			side:       "LEFT",
			want:       []string{"2: -deleted line"},
		},
		{
			name: "LEFT side - multi-line deletion",
			diffHunk: `@@ -1,5 +1,3 @@
 line1
-deleted1
-deleted2
 line4
 line5`,
			startLine:  1,
			targetLine: 2,
			side:       "LEFT",
			want: []string{
				"1: line1",
				"2: -deleted1",
			},
		},
		{
			name:       "empty diffHunk",
			diffHunk:   "",
			startLine:  1,
			targetLine: 1,
			side:       "RIGHT",
			want:       nil,
		},
		{
			name: "context lines only",
			diffHunk: `@@ -20,5 +20,5 @@
 context20
 context21
 context22
 context23
 context24`,
			startLine:  21,
			targetLine: 23,
			side:       "RIGHT",
			want: []string{
				"21: context21",
				"22: context22",
				"23: context23",
			},
		},
		{
			name: "with no newline marker - RIGHT",
			diffHunk: `@@ -1,5 +1,6 @@
 line1
 line2
+added line
 line4
 \ No newline at end of file`,
			startLine:  3,
			targetLine: 3,
			side:       "RIGHT",
			want:       []string{"3: +added line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDiffHunk(tt.diffHunk, tt.startLine, tt.targetLine, tt.side)

			if len(got) != len(tt.want) {
				t.Errorf("parseDiffHunk() got %d lines, want %d lines", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseDiffHunk() line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseDiffHunk_OutOfRange(t *testing.T) {
	diffHunk := `@@ -1,3 +1,4 @@
 line1
+added
 line2
 line3`

	// Line 100 is outside the hunk range
	result := parseDiffHunk(diffHunk, 100, 100, "RIGHT")
	if len(result) != 0 {
		t.Errorf("expected empty result for out-of-range line, got %v", result)
	}
}

func TestExtractCodeContext(t *testing.T) {
	// diff hunk: @@ -1,3 +1,4 @@
	// line1     (unchanged, new line 1)
	// +added    (added, new line 2)
	// line2     (unchanged, new line 3)
	// line3     (unchanged, new line 4)
	thread := threadInfo{
		ID:           "PRRT_test",
		Path:         "test.go",
		DiffSide:     "RIGHT",
		Line:         2, // Target the added line
		StartLine:    0,
		OriginalLine: 0,
		Comments: []commentInfo{
			{
				ID:       "PRRC_test",
				DiffHunk: "@@ -1,3 +1,4 @@\n line1\n+added line\n line2\n line3",
			},
		},
	}

	// Create a mock service
	s := &Service{}

	result := s.extractCodeContext(thread)
	if len(result) != 1 {
		t.Errorf("expected 1 line, got %d: %v", len(result), result)
	}
	if result[0] != "2: +added line" {
		t.Errorf("unexpected result: %q", result[0])
	}
}

func TestThreadInfo_LineSelection(t *testing.T) {
	// Test LEFT side thread
	leftThread := threadInfo{
		DiffSide:          "LEFT",
		Line:              0, // Not used
		StartLine:         0, // Not used
		OriginalLine:      10,
		OriginalStartLine: 5,
	}

	if leftThread.DiffSide != "LEFT" {
		t.Error("expected LEFT side")
	}
	if leftThread.OriginalLine != 10 {
		t.Errorf("expected OriginalLine=10, got %d", leftThread.OriginalLine)
	}
	if leftThread.OriginalStartLine != 5 {
		t.Errorf("expected OriginalStartLine=5, got %d", leftThread.OriginalStartLine)
	}

	// Test RIGHT side thread
	rightThread := threadInfo{
		DiffSide:          "RIGHT",
		Line:              20,
		StartLine:         15,
		OriginalLine:      0, // Not used
		OriginalStartLine: 0, // Not used
	}

	if rightThread.DiffSide != "RIGHT" {
		t.Error("expected RIGHT side")
	}
	if rightThread.Line != 20 {
		t.Errorf("expected Line=20, got %d", rightThread.Line)
	}
	if rightThread.StartLine != 15 {
		t.Errorf("expected StartLine=15, got %d", rightThread.StartLine)
	}
}
