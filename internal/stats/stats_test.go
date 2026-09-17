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

func TestSortReposAndStars(t *testing.T) {
	t.Parallel()

	in := []Repo{
		{Name: "b", Stars: 10},
		{Name: "a", Stars: 10},
		{Name: "c", Stars: 2},
	}
	got := SortRepos(in)
	want := []Repo{
		{Name: "a", Stars: 10},
		{Name: "b", Stars: 10},
		{Name: "c", Stars: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if in[0].Name != "b" {
		t.Fatalf("SortRepos mutated input")
	}
	if TotalStars(in) != 22 {
		t.Fatalf("TotalStars = %d, want 22", TotalStars(in))
	}
	if TotalStars(nil) != 0 {
		t.Fatalf("TotalStars(nil) = %d", TotalStars(nil))
	}
}
