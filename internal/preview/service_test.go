package preview

import (
	"testing"
)

func TestExtractCodeContext(t *testing.T) {
	tests := []struct {
		name      string
		patch     string
		line      int
		startLine int
		side      string
		want      []string
	}{
		{
			name: "simple addition - single line",
			patch: `@@ -1,5 +1,6 @@
 line1
 line2
+added line
 line3
 line4
 line5`,
			line:      3,
			startLine: 0,
			side:      "RIGHT",
			want:      []string{"3: +added line"},
		},
		{
			name: "multi-line context",
			patch: `@@ -10,5 +10,8 @@
 line10
 line11
+new line 12
+new line 13
+new line 14
 line15
 line16`,
			line:      14,
			startLine: 12,
			side:      "RIGHT",
			want: []string{
				"12: +new line 12",
				"13: +new line 13",
				"14: +new line 14",
			},
		},
		{
			name: "context lines (space prefix)",
			// @@ -20,5 +20,5 @@ means old starts at 20, new starts at 20
			// After change, new file lines are:
			// 20: context20 (context)
			// 21: context21 (context)
			// 22: new line (added)
			// 23: context24 (context, was line 23 in old)
			// 24: context25 (context, was line 24 in old)
			patch: `@@ -20,5 +20,5 @@
 context20
 context21
-old line
+new line
 context24
 context25`,
			line:      23,
			startLine: 21,
			side:      "RIGHT",
			want: []string{
				"21: context21",
				"22: +new line",
				"23: context24",
			},
		},
		{
			name:      "empty patch",
			patch:     "",
			line:      10,
			startLine: 0,
			side:      "RIGHT",
			want:      nil,
		},
		{
			name: "skip deleted lines",
			// @@ -30,5 +30,4 @@: old starts at 30, count 5; new starts at 30, count 4
			// Old file lines 30-34: line30, deleted1, deleted2, line33, line34
			// New file lines 30-33: line30, line33, line34 (deletions removed)
			// After mapping:
			// 30: line30 (context)
			// 31: line33 (context that was at old line 33)
			// 32: line34 (context that was at old line 34)
			patch: `@@ -30,5 +30,4 @@
 line30
-deleted1
-deleted2
 line33
 line34`,
			line:      31,
			startLine: 0,
			side:      "RIGHT",
			want:      []string{"31: line33"},
		},
		{
			name: "multiple hunks - second hunk",
			patch: `@@ -1,5 +1,5 @@
 line1
 line2
 line3
 line4
 line5
@@ -50,5 +50,6 @@
 line50
 line51
+added52
 line52
 line53
 line54`,
			line:      52,
			startLine: 52,
			side:      "RIGHT",
			want:      []string{"52: +added52"},
		},
		{
			name: "no newline at end of file marker",
			// @@ -1,4 +1,5 @@: old starts at 1, new starts at 1
			// Lines in new file:
			// 1: line1 (context)
			// 2: line2 (context)
			// 3: added line (added)
			// 4: line4 (context, was at old line 3)
			patch: `@@ -1,4 +1,5 @@
 line1
 line2
+added line
 line4
 \ No newline at end of file`,
			line:      3,
			startLine: 0,
			side:      "RIGHT",
			want:      []string{"3: +added line"},
		},
		{
			name: "single line selection in range",
			// @@ -100,10 +100,12 @@: old starts at 100, count 10; new starts at 100, count 12
			// New file lines (with additions marked):
			// 100: line100 (context)
			// 101: line101 (context)
			// 102: line102 (context)
			// 103: +line102a (added)
			// 104: +line102b (added)
			// 105: line103 (context, was old line 103)
			patch: `@@ -100,10 +100,12 @@
 line100
 line101
 line102
+line102a
+line102b
 line103
 line104
 line105
 line106
 line107
 line108
 line109`,
			line:      105,
			startLine: 0,
			side:      "RIGHT",
			want:      []string{"105: line103"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeContext(tt.patch, tt.line, tt.startLine, tt.side)

			if len(got) != len(tt.want) {
				t.Errorf("extractCodeContext() got %d lines, want %d lines", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractCodeContext() line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExtractCodeContext_EmptyResult(t *testing.T) {
	// Test when target line is outside hunk range
	patch := `@@ -1,3 +1,4 @@
 line1
+added
 line2
 line3`

	// Line 100 is outside the hunk range
	result := extractCodeContext(patch, 100, 0, "RIGHT")
	if len(result) != 0 {
		t.Errorf("expected empty result for out-of-range line, got %v", result)
	}
}
