package api

import "testing"

func TestAlbum_ShelfCode(t *testing.T) {
	tests := []struct {
		name string
		a    Album
		want string
	}{
		{
			name: "full code with genre",
			a:    Album{GenreName: "Jazz", CodeLetters: "DA", CodeArtistNumber: 11, CodeNumber: 4},
			want: "Jazz DA 11/4",
		},
		{
			name: "missing genre falls back to bare code",
			a:    Album{CodeLetters: "AP", CodeArtistNumber: 2, CodeNumber: 5},
			want: "AP 2/5",
		},
		{
			name: "no code letters yields no shelf location",
			a:    Album{GenreName: "Jazz", CodeArtistNumber: 11, CodeNumber: 4},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.ShelfCode(); got != tt.want {
				t.Errorf("ShelfCode() = %q, want %q", got, tt.want)
			}
		})
	}
}
