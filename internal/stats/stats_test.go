package stats

import (
	"reflect"
	"testing"
)

func TestSumLanguages(t *testing.T) {
	t.Parallel()

	got := SumLanguages([]map[string]int64{
		{"Go": 100, "Python": 50},
		{"Go": 20},
		nil,
		{},
	})
	want := map[string]int64{"Go": 120, "Python": 50}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if len(SumLanguages(nil)) != 0 {
		t.Fatalf("empty input should yield empty map")
	}
}

func TestTopN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		bytes map[string]int64
		n     int
		want  []Language
	}{
		{
			name:  "empty",
			bytes: map[string]int64{},
			n:     8,
			want:  []Language{},
		},
		{
			name:  "fewer than n no other",
			bytes: map[string]int64{"Go": 12000, "C": 1000},
			n:     8,
			want: []Language{
				{Name: "Go", Bytes: 12000, Percent: 92.3},
				{Name: "C", Bytes: 1000, Percent: 7.7},
			},
		},
		{
			name: "top n plus other",
			bytes: map[string]int64{
				"Go": 100, "C": 80, "Rust": 50, "Python": 10,
			},
			n: 2,
			want: []Language{
				{Name: "Go", Bytes: 100, Percent: 41.7},
				{Name: "C", Bytes: 80, Percent: 33.3},
				{Name: "Other", Bytes: 60, Percent: 25.0},
			},
		},
		{
			name: "all remainder in other",
			bytes: map[string]int64{
				"Go": 5, "C": 4, "Rust": 3,
			},
			n: 1,
			want: []Language{
				{Name: "Go", Bytes: 5, Percent: 41.7},
				{Name: "Other", Bytes: 7, Percent: 58.3},
			},
		},
		{
			name:  "tie broken by name",
			bytes: map[string]int64{"B": 10, "A": 10, "C": 1},
			n:     2,
			want: []Language{
				{Name: "A", Bytes: 10, Percent: 47.6},
				{Name: "B", Bytes: 10, Percent: 47.6},
				{Name: "Other", Bytes: 1, Percent: 4.8},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TopN(tt.bytes, tt.n)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestExcludeHTMLCSSAndUpdated(t *testing.T) {
	t.Parallel()

	sum := SumLanguages([]map[string]int64{
		{"Go": 100, "HTML": 80, "CSS": 20, "Python": 50},
	})
	got := TopN(Exclude(sum, markupLangs...), 8)
	for _, l := range got {
		if l.Name == "HTML" || l.Name == "CSS" || l.Name == "Other" {
			t.Fatalf("unexpected %q in %#v", l.Name, got)
		}
	}
	if len(got) != 2 || got[0].Name != "Go" || got[1].Name != "Python" {
		t.Fatalf("got %#v", got)
	}
	if got[0].Percent != 66.7 || got[1].Percent != 33.3 {
		t.Fatalf("percents %#v (should ignore HTML/CSS bytes)", got)
	}

	in := []Repo{
		{Name: "b", UpdatedAt: "2026-01-02"},
		{Name: "a", UpdatedAt: "2026-01-02"},
		{Name: "old", UpdatedAt: "2020-01-01"},
	}
	sorted := SortByUpdated(in)
	if sorted[0].Name != "a" || sorted[1].Name != "b" || sorted[2].Name != "old" {
		t.Fatalf("sort %#v", sorted)
	}
	if in[0].Name != "b" {
		t.Fatal("SortByUpdated mutated input")
	}
	vis := Visible(sorted, false)
	if len(vis) != 3 {
		t.Fatalf("short list Visible = %d", len(vis))
	}
	var many []Repo
	for i := 0; i < 10; i++ {
		many = append(many, Repo{Name: string(rune('a' + i)), UpdatedAt: "2026-01-02"})
	}
	if n := len(Visible(many, false)); n != 5 {
		t.Fatalf("Visible default %d", n)
	}
	if n := len(Visible(many, true)); n != 10 {
		t.Fatalf("Visible all %d", n)
	}
	if TotalStars([]Repo{{Stars: 10}, {Stars: 12}}) != 22 || TotalStars(nil) != 0 {
		t.Fatal("TotalStars")
	}
}
