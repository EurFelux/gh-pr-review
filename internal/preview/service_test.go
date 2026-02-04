package preview

import (
	"testing"
)

func TestExtractCodeContext(t *testing.T) {
	tests := []struct {
		name              string
		patch             string
		line              int
		startLine         int
		side              string
		originalLine      int
		originalStartLine int
		want              []string
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
			name: "skip deleted lines on RIGHT side",
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
		{
			name: "multi-line range without additions",
			patch: `@@ -50,7 +50,7 @@
 line50
 line51
 line52
 line53
 line54
 line55
 line56`,
			line:      54,
			startLine: 52,
			side:      "RIGHT",
			want: []string{
				"52: line52",
				"53: line53",
				"54: line54",
			},
		},
		{
			name: "multi-line range with additions",
			patch: `@@ -200,5 +200,8 @@
 line200
+added201
+added202
 line201
 line202
 line203
 line204`,
			line:      203,
			startLine: 201,
			side:      "RIGHT",
			want: []string{
				"201: +added201",
				"202: +added202",
				"203: line201",
			},
		},
		{
			name: "LEFT side - show deleted line",
			patch: `@@ -1,5 +1,4 @@
 line1
-deleted2
-deleted3
+added_new
 line4
 line5`,
			line:              0, // Not used for LEFT side
			startLine:         0, // Not used for LEFT side
			side:              "LEFT",
			originalLine:      2,
			originalStartLine: 0,
			want:              []string{"2: -deleted2"},
		},
		{
			name: "LEFT side - multi-line range",
			patch: `@@ -1,5 +1,4 @@
 line1
-deleted2
-deleted3
+added_new
 line4
 line5`,
			line:              0, // Not used for LEFT side
			startLine:         0, // Not used for LEFT side
			side:              "LEFT",
			originalLine:      3,
			originalStartLine: 2,
			want: []string{
				"2: -deleted2",
				"3: -deleted3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeContext(tt.patch, tt.line, tt.startLine, tt.side, tt.originalLine, tt.originalStartLine)

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
	result := extractCodeContext(patch, 100, 0, "RIGHT", 0, 0)
	if len(result) != 0 {
		t.Errorf("expected empty result for out-of-range line, got %v", result)
	}
}

func TestInferSideFromDiffHunk(t *testing.T) {
	tests := []struct {
		name     string
		diffHunk string
		want     string
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
			want: "RIGHT",
		},
		{
			name: "LEFT side - deleted line",
			diffHunk: `@@ -2,3 +2,2 @@
 line1
-deleted line`,
			want: "LEFT",
		},
		{
			name: "RIGHT side - context line",
			diffHunk: `@@ -2,3 +2,3 @@
 line1
 context line`,
			want: "RIGHT",
		},
		{
			name:     "empty diffHunk",
			diffHunk: "",
			want:     "RIGHT",
		},
		{
			name: "multiple hunks - last is LEFT",
			diffHunk: `@@ -10,3 +10,2 @@
 line10
-deleted`,
			want: "LEFT",
		},
		{
			name: "with no newline marker - added line",
			diffHunk: `@@ -1,5 +1,6 @@
 line1
 line2
+added line
 \ No newline at end of file`,
			want: "RIGHT",
		},
		{
			name: "with no newline marker - deleted line",
			diffHunk: `@@ -1,5 +1,4 @@
 line1
 line2
-deleted line
 \ No newline at end of file`,
			want: "LEFT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferSideFromDiffHunk(tt.diffHunk)
			if got != tt.want {
				t.Errorf("inferSideFromDiffHunk() = %q, want %q", got, tt.want)
			}
		})
	}
}
